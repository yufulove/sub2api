package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type studioImageGenerationRequest struct {
	GroupID int64 `json:"group_id"`
}

// StudioImageGenerations is the image workspace entrypoint for logged-in users.
// It keeps image generation billed through the existing usage pipeline without
// exposing API keys in the Studio UI.
func (h *Handlers) StudioImageGenerations(c *gin.Context) {
	if h == nil || h.APIKey == nil || h.APIKey.apiKeyService == nil || h.Gateway == nil || h.OpenAIGateway == nil {
		studioImageError(c, http.StatusInternalServerError, "api_error", "Image studio is not configured")
		return
	}

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		studioImageError(c, http.StatusUnauthorized, "authentication_error", "Login is required")
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			studioImageError(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	var studioReq studioImageGenerationRequest
	if err := json.Unmarshal(body, &studioReq); err != nil {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	if studioReq.GroupID <= 0 {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "image group is required")
		return
	}

	parsed, err := service.ParseOpenAIImageGenerationRequest(body)
	if err != nil {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	if !service.IsImageGenerationModel(parsed.Model) {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "model must be an image generation model")
		return
	}
	parsed.Model = service.CanonicalImageGenerationModel(parsed.Model)

	apiKey, err := h.APIKey.apiKeyService.GetOrCreateStudioImageAPIKey(c.Request.Context(), subject.UserID, studioReq.GroupID)
	if err != nil {
		studioImageError(c, http.StatusForbidden, "invalid_request_error", err.Error())
		return
	}
	if apiKey == nil || apiKey.User == nil || apiKey.Group == nil || apiKey.GroupID == nil {
		studioImageError(c, http.StatusInternalServerError, "api_error", "Image route is not available")
		return
	}
	if !studioImageGroupSupportsModel(apiKey.Group, parsed.Model) {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "selected image route is not compatible with model")
		return
	}

	sanitizedBody, err := sanitizeStudioImageBody(body, parsed.Model)
	if err != nil {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "Failed to build image request")
		return
	}

	if apiKey.Group.IsSubscriptionType() {
		subscription, err := h.APIKey.apiKeyService.GetActiveSubscriptionForGroup(c.Request.Context(), apiKey.UserID, apiKey.Group.ID)
		if err != nil {
			studioImageError(c, http.StatusForbidden, "invalid_request_error", "No active subscription found for this image route")
			return
		}
		c.Set(string(middleware.ContextKeySubscription), subscription)
	}

	c.Set(string(middleware.ContextKeyAPIKey), apiKey)
	ctx := context.WithValue(c.Request.Context(), ctxkey.Group, apiKey.Group)
	c.Request = c.Request.WithContext(ctx)
	c.Request.Body = io.NopCloser(bytes.NewReader(sanitizedBody))
	c.Request.ContentLength = int64(len(sanitizedBody))
	c.Request.Header.Set("Content-Type", "application/json")
	_ = h.APIKey.apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)

	if apiKey.Group.Platform == service.PlatformOpenAI {
		h.OpenAIGateway.Images(c)
		return
	}
	h.Gateway.ImageGenerations(c)
}

func sanitizeStudioImageBody(body []byte, canonicalModel string) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	delete(payload, "group_id")
	payload["model"] = canonicalModel
	if _, ok := payload["response_format"]; !ok {
		payload["response_format"] = "b64_json"
	}
	return json.Marshal(payload)
}

func studioImageGroupSupportsModel(group *service.Group, model string) bool {
	if group == nil || group.Status != service.StatusActive {
		return false
	}

	platform := strings.ToLower(strings.TrimSpace(group.Platform))
	model = strings.ToLower(service.CanonicalImageGenerationModel(model))
	switch {
	case strings.HasPrefix(model, "gpt-image-"), model == "dall-e-2", model == "dall-e-3":
		return platform == service.PlatformOpenAI
	case strings.HasPrefix(model, "gemini-") && strings.Contains(model, "-image"):
		return platform == service.PlatformGemini || platform == service.PlatformAntigravity
	default:
		return false
	}
}

func studioImageError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}
