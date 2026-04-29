package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setGatewayAuthContext(c *gin.Context, apiKey *service.APIKey) {
	c.Set(string(middleware.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7, Concurrency: 1})
}

func decodeGatewayError(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(body, &parsed))
	return parsed
}

func TestGatewayMessagesRejectsImageGenerationModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gemini-2.5-flash-image","messages":[{"role":"user","content":[{"type":"text","text":"draw a cat"}]}]}`))
	setGatewayAuthContext(c, &service.APIKey{ID: 11, User: &service.User{ID: 7}})

	h := &GatewayHandler{}
	h.Messages(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	parsed := decodeGatewayError(t, rec.Body.Bytes())
	errorObj := parsed["error"].(map[string]any)
	require.Equal(t, service.DedicatedImageGenerationEndpointMessage(), errorObj["message"])
}

func TestOpenAIResponsesRejectsImageGenerationModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-image-1","input":"draw a cat"}`))
	setGatewayAuthContext(c, &service.APIKey{ID: 12, User: &service.User{ID: 7}})

	h := &OpenAIGatewayHandler{
		gatewayService:      &service.OpenAIGatewayService{},
		billingCacheService: &service.BillingCacheService{},
		apiKeyService:       &service.APIKeyService{},
		concurrencyHelper:   NewConcurrencyHelper(&service.ConcurrencyService{}, SSEPingFormatComment, 0),
	}
	h.Responses(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	parsed := decodeGatewayError(t, rec.Body.Bytes())
	errorObj := parsed["error"].(map[string]any)
	require.Equal(t, service.DedicatedImageGenerationEndpointMessage(), errorObj["message"])
}

func TestGeminiV1BetaModelsRejectsImageGenerationModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash-image:generateContent", nil)
	c.Params = gin.Params{{Key: "modelAction", Value: "/gemini-2.5-flash-image:generateContent"}}
	setGatewayAuthContext(c, &service.APIKey{
		ID:   13,
		User: &service.User{ID: 7},
		Group: &service.Group{
			Platform: service.PlatformGemini,
		},
	})

	h := &GatewayHandler{}
	h.GeminiV1BetaModels(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	parsed := decodeGatewayError(t, rec.Body.Bytes())
	errorObj := parsed["error"].(map[string]any)
	require.Equal(t, service.DedicatedImageGenerationEndpointMessage(), errorObj["message"])
}

func TestGatewayImageGenerationsRejectsMultipleImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"gemini-2.5-flash-image","prompt":"draw a cat","n":2}`))
	setGatewayAuthContext(c, &service.APIKey{ID: 14, User: &service.User{ID: 7}})

	h := &GatewayHandler{}
	h.ImageGenerations(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	parsed := decodeGatewayError(t, rec.Body.Bytes())
	errorObj := parsed["error"].(map[string]any)
	require.Equal(t, "only n=1 is currently supported", errorObj["message"])
}

func TestBufferedForwardErrorStatusNormalizesSuccessRecorderCode(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Code = http.StatusOK

	require.Equal(t, http.StatusBadGateway, bufferedForwardErrorStatus(rec))

	rec = httptest.NewRecorder()
	rec.WriteHeader(http.StatusTooManyRequests)
	require.Equal(t, http.StatusTooManyRequests, bufferedForwardErrorStatus(rec))
}

func TestOpenAIImageGenerationsNotImplemented(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	h := &OpenAIGatewayHandler{}
	h.ImageGenerations(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	parsed := decodeGatewayError(t, rec.Body.Bytes())
	errorObj := parsed["error"].(map[string]any)
	require.Equal(t, "Image generation for OpenAI upstream groups is not implemented yet", errorObj["message"])
}
