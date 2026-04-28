package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

const DedicatedImageGenerationEndpoint = "/v1/images/generations"

type OpenAIImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size,omitempty"`
	N              int    `json:"n,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	User           string `json:"user,omitempty"`
}

type OpenAIImageGenerationData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type OpenAIImageGenerationResponse struct {
	Created int64                       `json:"created"`
	Data    []OpenAIImageGenerationData `json:"data"`
}

func DedicatedImageGenerationEndpointMessage() string {
	return "Image generation requests must use " + DedicatedImageGenerationEndpoint
}

func IsImageGenerationModel(model string) bool {
	switch normalizeImageGenerationModelName(model) {
	case "gpt-image-1",
		"dall-e-2",
		"dall-e-3",
		"gemini-3.1-flash-image",
		"gemini-3.1-flash-image-preview",
		"gemini-3-pro-image",
		"gemini-3-pro-image-preview",
		"gemini-2.5-flash-image",
		"gemini-2.5-flash-image-preview":
		return true
	default:
		return false
	}
}

func normalizeImageGenerationModelName(model string) string {
	normalized := strings.ToLower(strings.TrimSpace(model))
	normalized = strings.TrimPrefix(normalized, "models/")
	return normalized
}

func ParseOpenAIImageGenerationRequest(body []byte) (*OpenAIImageGenerationRequest, error) {
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("failed to parse request body")
	}

	var req OpenAIImageGenerationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse request body")
	}

	req.Model = strings.TrimSpace(req.Model)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.Size = strings.TrimSpace(req.Size)
	req.ResponseFormat = strings.TrimSpace(req.ResponseFormat)

	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	if req.N == 0 {
		req.N = 1
	}
	if req.N != 1 {
		return nil, fmt.Errorf("only n=1 is currently supported")
	}
	if req.ResponseFormat == "" {
		req.ResponseFormat = "b64_json"
	}
	if req.ResponseFormat != "b64_json" {
		return nil, fmt.Errorf("only response_format=b64_json is currently supported")
	}

	return &req, nil
}

func BuildGeminiImageGenerationRequest(req *OpenAIImageGenerationRequest) ([]byte, string, error) {
	if req == nil {
		return nil, "", fmt.Errorf("request is required")
	}

	imageSize, aspectRatio, err := NormalizeImageGenerationSize(req.Size)
	if err != nil {
		return nil, "", err
	}

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": req.Prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"responseModalities": []string{"TEXT", "IMAGE"},
			"imageConfig": map[string]any{
				"aspectRatio": aspectRatio,
				"imageSize":   imageSize,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build gemini image request")
	}
	return body, imageSize, nil
}

func NormalizeImageGenerationSize(raw string) (string, string, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "2K", "AUTO":
		return "2K", "1:1", nil
	case "1K", "1024X1024":
		return "1K", "1:1", nil
	case "4K", "4096X4096":
		return "4K", "1:1", nil
	case "1536X1024", "1792X1024":
		return "2K", "3:2", nil
	case "1024X1536", "1024X1792":
		return "2K", "2:3", nil
	case "2048X2048":
		return "2K", "1:1", nil
	default:
		return "", "", fmt.Errorf("unsupported image size: %s", raw)
	}
}

func ParseGeminiImageGenerationResponse(body []byte) (*OpenAIImageGenerationResponse, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty upstream response")
	}

	root := gjson.ParseBytes(body)
	if response := root.Get("response"); response.Exists() {
		root = response
	}

	candidates := root.Get("candidates")
	if !candidates.Exists() || !candidates.IsArray() {
		return nil, fmt.Errorf("upstream image response did not contain candidates")
	}

	data := make([]OpenAIImageGenerationData, 0, 1)
	for _, candidate := range candidates.Array() {
		var revisedPrompt string
		parts := candidate.Get("content.parts")
		if !parts.Exists() || !parts.IsArray() {
			continue
		}
		for _, part := range parts.Array() {
			text := strings.TrimSpace(part.Get("text").String())
			if text != "" && revisedPrompt == "" {
				revisedPrompt = text
			}
			mimeType := strings.ToLower(strings.TrimSpace(part.Get("inlineData.mimeType").String()))
			imageData := strings.TrimSpace(part.Get("inlineData.data").String())
			if !strings.HasPrefix(mimeType, "image/") || imageData == "" {
				continue
			}
			data = append(data, OpenAIImageGenerationData{
				B64JSON:       imageData,
				RevisedPrompt: revisedPrompt,
			})
		}
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("upstream image response did not contain image data")
	}

	return &OpenAIImageGenerationResponse{
		Created: time.Now().Unix(),
		Data:    data,
	}, nil
}
