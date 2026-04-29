package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// ImageGenerations is the dedicated image-generation entrypoint for non-OpenAI groups.
// It reuses the existing Gemini/Antigravity native forwarders, but buffers the provider
// response so we can return a stable OpenAI Images-style payload to clients.
func (h *GatewayHandler) ImageGenerations(c *gin.Context) {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	reqLog := requestLogger(
		c,
		"handler.gateway.image_generations",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	req, err := service.ParseOpenAIImageGenerationRequest(body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	if !service.IsImageGenerationModel(req.Model) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model must be an image generation model")
		return
	}
	req.Model = service.CanonicalImageGenerationModel(req.Model)
	if h.gatewayService == nil || h.geminiCompatService == nil {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "Image generation handler dependencies are not configured")
		return
	}

	setOpsRequestContext(c, req.Model, false, body)
	setOpsEndpointContext(c, "", int16(service.RequestTypeSync))
	reqLog = reqLog.With(zap.String("model", req.Model))

	geminiBody, normalizedImageSize, err := service.BuildGeminiImageGenerationRequest(req)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	subscription, _ := middleware.GetSubscriptionFromContext(c)
	if h.billingCacheService != nil {
		if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
			status, code, message, retryAfter := billingErrorDetails(err)
			if retryAfter > 0 {
				c.Header("Retry-After", strconv.Itoa(retryAfter))
			}
			h.errorResponse(c, status, code, message)
			return
		}
	}

	streamStarted := false
	if h.concurrencyHelper != nil {
		userReleaseFunc, acquireErr := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, false, &streamStarted)
		if acquireErr != nil {
			h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", acquireErr.Error())
			return
		}
		userReleaseFunc = wrapReleaseOnDone(c.Request.Context(), userReleaseFunc)
		if userReleaseFunc != nil {
			defer userReleaseFunc()
		}
	}

	sessionHash := service.DeriveSessionHashFromSeed(req.Model + ":" + req.Prompt)
	selection, err := h.gatewayService.SelectAccountWithLoadAwareness(
		c.Request.Context(),
		apiKey.GroupID,
		sessionHash,
		req.Model,
		nil,
		"",
		subject.UserID,
	)
	if err != nil {
		reqLog.Warn("image_generations.account_select_failed", zap.Error(err))
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available image generation accounts")
		return
	}
	if selection == nil || selection.Account == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available image generation accounts")
		return
	}

	account := selection.Account
	setOpsSelectedAccount(c, account.ID, account.Platform)

	var accountReleaseFunc func()
	if selection.WaitPlan == nil {
		accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), selection.ReleaseFunc)
	} else if h.concurrencyHelper != nil {
		canWait, waitErr := h.concurrencyHelper.IncrementAccountWaitCount(c.Request.Context(), account.ID, selection.WaitPlan.MaxWaiting)
		if waitErr != nil {
			reqLog.Warn("image_generations.account_wait_counter_failed", zap.Int64("account_id", account.ID), zap.Error(waitErr))
		} else if !canWait {
			h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later")
			return
		}
		if waitErr == nil {
			defer h.concurrencyHelper.DecrementAccountWaitCount(c.Request.Context(), account.ID)
		}

		accountReleaseFunc, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(
			c,
			account.ID,
			selection.WaitPlan.MaxConcurrency,
			selection.WaitPlan.Timeout,
			false,
			&streamStarted,
		)
		if err != nil {
			h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", err.Error())
			return
		}
		accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)
	}
	if accountReleaseFunc != nil {
		defer accountReleaseFunc()
	}

	proxyCtx, rec := newBufferedForwardContext(c, geminiBody)

	var result *service.ForwardResult
	switch {
	case account.Platform == service.PlatformAntigravity && account.Type != service.AccountTypeAPIKey:
		if h.antigravityGatewayService == nil {
			h.errorResponse(c, http.StatusInternalServerError, "api_error", "Antigravity image generation is not configured")
			return
		}
		result, err = h.antigravityGatewayService.ForwardGemini(
			c.Request.Context(),
			proxyCtx,
			account,
			req.Model,
			"generateContent",
			false,
			geminiBody,
			false,
		)
	default:
		result, err = h.geminiCompatService.ForwardNative(
			c.Request.Context(),
			proxyCtx,
			account,
			req.Model,
			"generateContent",
			false,
			geminiBody,
		)
	}
	if err != nil {
		status := rec.Code
		if status == 0 {
			status = http.StatusBadGateway
		}
		message := extractBufferedForwardError(rec.Body.Bytes(), err)
		errType := "api_error"
		if status >= 400 && status < 500 {
			errType = "invalid_request_error"
		}
		reqLog.Warn("image_generations.forward_failed",
			zap.Int64("account_id", account.ID),
			zap.Int("status", status),
			zap.Error(err),
		)
		h.errorResponse(c, status, errType, message)
		return
	}

	imageResp, err := service.ParseGeminiImageGenerationResponse(rec.Body.Bytes())
	if err != nil {
		reqLog.Warn("image_generations.parse_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		h.errorResponse(c, http.StatusBadGateway, "api_error", err.Error())
		return
	}

	if result != nil {
		if result.ImageCount <= 0 {
			result.ImageCount = len(imageResp.Data)
		}
		if strings.TrimSpace(result.ImageSize) == "" {
			result.ImageSize = normalizedImageSize
		}
		if strings.TrimSpace(result.ImageRequestedSize) == "" {
			result.ImageRequestedSize = req.Size
		}
		if strings.TrimSpace(result.ImagePrompt) == "" {
			result.ImagePrompt = req.Prompt
		}
		if strings.TrimSpace(result.ImageRevisedPrompt) == "" {
			result.ImageRevisedPrompt = firstRevisedImagePrompt(imageResp)
		}
	}

	c.JSON(http.StatusOK, imageResp)

	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	h.submitUsageRecordTask(func(ctx context.Context) {
		if result == nil {
			return
		}
		if err := h.gatewayService.RecordUsageWithLongContext(ctx, &service.RecordUsageLongContextInput{
			Result:                result,
			APIKey:                apiKey,
			User:                  apiKey.User,
			Account:               account,
			Subscription:          subscription,
			UserAgent:             userAgent,
			IPAddress:             clientIP,
			InboundEndpoint:       GetInboundEndpoint(c),
			UpstreamEndpoint:      GetUpstreamEndpoint(c, account.Platform),
			LongContextThreshold:  200000,
			LongContextMultiplier: 2.0,
			APIKeyService:         h.apiKeyService,
		}); err != nil {
			logger.L().With(
				zap.String("component", "handler.gateway.image_generations"),
				zap.Int64("user_id", subject.UserID),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Any("group_id", apiKey.GroupID),
				zap.String("model", req.Model),
				zap.Int64("account_id", account.ID),
			).Error("image_generations.record_usage_failed", zap.Error(err))
		}
	})
}

func (h *OpenAIGatewayHandler) ImageGenerations(c *gin.Context) {
	h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Image generation for OpenAI upstream groups is not implemented yet")
}

func newBufferedForwardContext(parent *gin.Context, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	proxyCtx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, service.DedicatedImageGenerationEndpoint, bytes.NewReader(body))
	if parent != nil && parent.Request != nil {
		req = req.WithContext(parent.Request.Context())
		req.Header = parent.Request.Header.Clone()
	}
	proxyCtx.Request = req
	return proxyCtx, rec
}

func extractBufferedForwardError(body []byte, err error) string {
	for _, path := range []string{"error.message", "error.error.message"} {
		if msg := strings.TrimSpace(gjson.GetBytes(body, path).String()); msg != "" {
			return msg
		}
	}
	if err != nil {
		return err.Error()
	}
	return "Upstream image generation request failed"
}

func firstRevisedImagePrompt(resp *service.OpenAIImageGenerationResponse) string {
	if resp == nil {
		return ""
	}
	for _, item := range resp.Data {
		if revised := strings.TrimSpace(item.RevisedPrompt); revised != "" {
			return revised
		}
	}
	return ""
}
