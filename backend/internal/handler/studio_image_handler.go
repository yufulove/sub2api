package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type studioImageGenerationRequest struct {
	GroupID int64 `json:"group_id"`
}

type studioImageHistoryCaptureInput struct {
	UserID  int64
	GroupID int64
	Model   string
	Size    string
	Prompt  string
}

const studioImageCaptureMaxBytes = 192 << 20

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

	historyInput := studioImageHistoryCaptureInput{
		UserID:  subject.UserID,
		GroupID: studioReq.GroupID,
		Model:   parsed.Model,
		Size:    parsed.Size,
		Prompt:  parsed.Prompt,
	}
	if apiKey.Group.Platform == service.PlatformOpenAI {
		h.forwardStudioImageRequestWithHistory(c, historyInput, func() {
			h.OpenAIGateway.Images(c)
		})
		return
	}
	h.forwardStudioImageRequestWithHistory(c, historyInput, func() {
		h.Gateway.ImageGenerations(c)
	})
}

// StudioImageEdits edits an existing image through the logged-in Studio entrypoint.
// The request is billed through the same hidden internal API key used by Studio
// image generation, while keeping API keys out of the browser.
func (h *Handlers) StudioImageEdits(c *gin.Context) {
	if h == nil || h.APIKey == nil || h.APIKey.apiKeyService == nil || h.OpenAIGateway == nil || h.OpenAIGateway.gatewayService == nil {
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

	groupID, err := extractStudioImageGroupID(body, c.GetHeader("Content-Type"))
	if err != nil {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	if groupID <= 0 {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "image group is required")
		return
	}

	parsed, err := h.OpenAIGateway.gatewayService.ParseOpenAIImagesRequest(c, body)
	if err != nil {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	parsed.Model = service.CanonicalImageGenerationModel(parsed.Model)
	if !service.IsImageGenerationModel(parsed.Model) {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "model must be an image generation model")
		return
	}

	apiKey, err := h.APIKey.apiKeyService.GetOrCreateStudioImageAPIKey(c.Request.Context(), subject.UserID, groupID)
	if err != nil {
		studioImageError(c, http.StatusForbidden, "invalid_request_error", err.Error())
		return
	}
	if apiKey == nil || apiKey.User == nil || apiKey.Group == nil || apiKey.GroupID == nil {
		studioImageError(c, http.StatusInternalServerError, "api_error", "Image route is not available")
		return
	}
	if apiKey.Group.Platform != service.PlatformOpenAI {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "image edits require an OpenAI image route")
		return
	}
	if !studioImageGroupSupportsModel(apiKey.Group, parsed.Model) {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "selected image route is not compatible with model")
		return
	}

	sanitizedBody, sanitizedContentType, err := sanitizeStudioImageEditBody(body, c.GetHeader("Content-Type"), parsed.Model)
	if err != nil {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "Failed to build image edit request")
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
	c.Request.Header.Set("Content-Type", sanitizedContentType)
	_ = h.APIKey.apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)

	h.forwardStudioImageRequestWithHistory(c, studioImageHistoryCaptureInput{
		UserID:  subject.UserID,
		GroupID: groupID,
		Model:   parsed.Model,
		Size:    parsed.Size,
		Prompt:  parsed.Prompt,
	}, func() {
		h.OpenAIGateway.Images(c)
	})
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

func extractStudioImageGroupID(body []byte, contentType string) (int64, error) {
	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return 0, nil
		}
		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return 0, err
			}
			if strings.TrimSpace(part.FormName()) != "group_id" || part.FileName() != "" {
				_ = part.Close()
				continue
			}
			data, readErr := io.ReadAll(io.LimitReader(part, 128))
			_ = part.Close()
			if readErr != nil {
				return 0, readErr
			}
			groupID, parseErr := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
			if parseErr != nil {
				return 0, parseErr
			}
			return groupID, nil
		}
		return 0, nil
	}

	var studioReq studioImageGenerationRequest
	if err := json.Unmarshal(body, &studioReq); err != nil {
		return 0, err
	}
	return studioReq.GroupID, nil
}

func sanitizeStudioImageEditBody(body []byte, contentType string, canonicalModel string) ([]byte, string, error) {
	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil || !strings.EqualFold(mediaType, "multipart/form-data") {
		sanitized, jsonErr := sanitizeStudioImageBody(body, canonicalModel)
		return sanitized, "application/json", jsonErr
	}

	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return nil, "", io.ErrUnexpectedEOF
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	modelWritten := false

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}

		formName := strings.TrimSpace(part.FormName())
		if formName == "group_id" && part.FileName() == "" {
			_ = part.Close()
			continue
		}

		target, err := writer.CreatePart(part.Header)
		if err != nil {
			_ = part.Close()
			return nil, "", err
		}
		if formName == "model" && part.FileName() == "" {
			if _, err := target.Write([]byte(canonicalModel)); err != nil {
				_ = part.Close()
				return nil, "", err
			}
			modelWritten = true
			_ = part.Close()
			continue
		}
		if _, err := io.Copy(target, part); err != nil {
			_ = part.Close()
			return nil, "", err
		}
		_ = part.Close()
	}

	if !modelWritten {
		if err := writer.WriteField("model", canonicalModel); err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return buffer.Bytes(), writer.FormDataContentType(), nil
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

type studioImageResponseCaptureWriter struct {
	gin.ResponseWriter
	body      bytes.Buffer
	maxBytes  int
	truncated bool
}

func (w *studioImageResponseCaptureWriter) Write(data []byte) (int, error) {
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *studioImageResponseCaptureWriter) WriteString(value string) (int, error) {
	w.capture([]byte(value))
	return w.ResponseWriter.WriteString(value)
}

func (w *studioImageResponseCaptureWriter) capture(data []byte) {
	if w == nil || w.maxBytes <= 0 || len(data) == 0 {
		return
	}
	remaining := w.maxBytes - w.body.Len()
	if remaining <= 0 {
		w.truncated = true
		return
	}
	if len(data) > remaining {
		w.body.Write(data[:remaining])
		w.truncated = true
		return
	}
	w.body.Write(data)
}

func (h *Handlers) forwardStudioImageRequestWithHistory(c *gin.Context, input studioImageHistoryCaptureInput, forward func()) {
	if c == nil || forward == nil {
		return
	}

	originalWriter := c.Writer
	capture := &studioImageResponseCaptureWriter{
		ResponseWriter: originalWriter,
		maxBytes:       studioImageCaptureMaxBytes,
	}
	c.Writer = capture
	defer func() {
		c.Writer = originalWriter
	}()

	forward()

	status := capture.Status()
	if status < http.StatusOK || status >= http.StatusMultipleChoices || capture.truncated || capture.body.Len() == 0 {
		if capture.truncated {
			logger.L().With(zap.String("component", "handler.studio_image")).Warn("studio_image.history_capture_truncated")
		}
		return
	}
	h.persistStudioImageHistoryFromResponse(input, capture.body.Bytes())
}

func (h *Handlers) persistStudioImageHistoryFromResponse(input studioImageHistoryCaptureInput, responseBody []byte) {
	if h == nil || h.StudioImageHistory == nil || input.UserID <= 0 || len(responseBody) == 0 {
		return
	}

	var response service.OpenAIImageGenerationResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		logger.L().With(zap.String("component", "handler.studio_image")).Warn("studio_image.history_response_parse_failed", zap.Error(err))
		return
	}
	if response.Created <= 0 {
		response.Created = time.Now().Unix()
	}

	images := make([]service.StudioImageHistorySaveImage, 0, len(response.Data))
	for index, item := range response.Data {
		if strings.TrimSpace(item.B64JSON) == "" {
			continue
		}
		images = append(images, service.StudioImageHistorySaveImage{
			ClientID:      fmt.Sprintf("server-%d-%d", response.Created, index),
			B64JSON:       item.B64JSON,
			RevisedPrompt: strings.TrimSpace(item.RevisedPrompt),
		})
	}
	if len(images) == 0 {
		return
	}

	groupID := input.GroupID
	requestID := fmt.Sprintf("studio-%d-%d-%d", input.UserID, response.Created, time.Now().UnixNano())
	saveInput := service.StudioImageHistorySaveInput{
		UserID:    input.UserID,
		GroupID:   &groupID,
		RequestID: requestID,
		Model:     strings.TrimSpace(input.Model),
		Size:      strings.TrimSpace(input.Size),
		Prompt:    strings.TrimSpace(input.Prompt),
		Created:   response.Created,
		Images:    images,
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		if _, err := h.StudioImageHistory.Save(ctx, saveInput); err != nil {
			logger.L().With(
				zap.String("component", "handler.studio_image"),
				zap.Int64("user_id", input.UserID),
				zap.Int64("group_id", input.GroupID),
				zap.String("model", input.Model),
			).Warn("studio_image.history_save_failed", zap.Error(err))
		}
	}()
}

func studioImageError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}
