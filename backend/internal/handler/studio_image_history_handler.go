package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type studioImageHistorySaveRequest struct {
	RequestID string                        `json:"request_id"`
	GroupID   *int64                        `json:"group_id"`
	Model     string                        `json:"model"`
	Size      string                        `json:"size"`
	Prompt    string                        `json:"prompt"`
	Created   int64                         `json:"created"`
	Images    []studioImageHistorySaveImage `json:"images"`
}

type studioImageHistorySaveImage struct {
	ClientID      string `json:"client_id"`
	B64JSON       string `json:"b64_json"`
	RevisedPrompt string `json:"revised_prompt"`
}

func (h *Handlers) SaveStudioImageHistory(c *gin.Context) {
	if h == nil || h.StudioImageHistory == nil {
		studioImageError(c, http.StatusInternalServerError, "api_error", "Image history is not configured")
		return
	}

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		studioImageError(c, http.StatusUnauthorized, "authentication_error", "Login is required")
		return
	}

	var req studioImageHistorySaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	if len(req.Images) == 0 {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "image data is required")
		return
	}
	if len(req.Images) > 8 {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "too many images in one history save request")
		return
	}

	images := make([]service.StudioImageHistorySaveImage, 0, len(req.Images))
	for _, image := range req.Images {
		images = append(images, service.StudioImageHistorySaveImage{
			ClientID:      strings.TrimSpace(image.ClientID),
			B64JSON:       image.B64JSON,
			RevisedPrompt: strings.TrimSpace(image.RevisedPrompt),
		})
	}

	assets, err := h.StudioImageHistory.Save(c.Request.Context(), service.StudioImageHistorySaveInput{
		UserID:    subject.UserID,
		GroupID:   req.GroupID,
		RequestID: strings.TrimSpace(req.RequestID),
		Model:     strings.TrimSpace(req.Model),
		Size:      strings.TrimSpace(req.Size),
		Prompt:    strings.TrimSpace(req.Prompt),
		Created:   req.Created,
		Images:    images,
	})
	if err != nil {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": assets})
}

func (h *Handlers) ListStudioImageHistory(c *gin.Context) {
	if h == nil || h.StudioImageHistory == nil {
		studioImageError(c, http.StatusInternalServerError, "api_error", "Image history is not configured")
		return
	}

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		studioImageError(c, http.StatusUnauthorized, "authentication_error", "Login is required")
		return
	}

	limit := 60
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			studioImageError(c, http.StatusBadRequest, "invalid_request_error", "invalid limit")
			return
		}
		limit = parsedLimit
	}

	assets, err := h.StudioImageHistory.ListByUser(c.Request.Context(), subject.UserID, limit)
	if err != nil {
		studioImageError(c, http.StatusInternalServerError, "api_error", "Failed to load image history")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": assets})
}

func (h *Handlers) GetStudioImageHistoryAsset(c *gin.Context) {
	if h == nil || h.StudioImageHistory == nil {
		studioImageError(c, http.StatusInternalServerError, "api_error", "Image history is not configured")
		return
	}

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		studioImageError(c, http.StatusUnauthorized, "authentication_error", "Login is required")
		return
	}

	assetID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || assetID <= 0 {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "invalid image id")
		return
	}

	file, err := h.StudioImageHistory.AssetFile(c.Request.Context(), subject.UserID, assetID)
	if errors.Is(err, service.ErrStudioImageAssetNotFound) {
		studioImageError(c, http.StatusNotFound, "not_found_error", "Image asset not found")
		return
	}
	if err != nil {
		studioImageError(c, http.StatusInternalServerError, "api_error", "Failed to load image asset")
		return
	}

	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("Content-Type", file.ContentType)
	c.File(file.Path)
}

func (h *Handlers) GetStudioImageHistoryThumbnail(c *gin.Context) {
	if h == nil || h.StudioImageHistory == nil {
		studioImageError(c, http.StatusInternalServerError, "api_error", "Image history is not configured")
		return
	}

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		studioImageError(c, http.StatusUnauthorized, "authentication_error", "Login is required")
		return
	}

	assetID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || assetID <= 0 {
		studioImageError(c, http.StatusBadRequest, "invalid_request_error", "invalid image id")
		return
	}

	file, err := h.StudioImageHistory.ThumbnailFile(c.Request.Context(), subject.UserID, assetID)
	if errors.Is(err, service.ErrStudioImageAssetNotFound) {
		studioImageError(c, http.StatusNotFound, "not_found_error", "Image asset not found")
		return
	}
	if err != nil {
		studioImageError(c, http.StatusInternalServerError, "api_error", "Failed to load image thumbnail")
		return
	}

	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("Content-Type", file.ContentType)
	c.File(file.Path)
}
