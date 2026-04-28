package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestIsImageGenerationModel(t *testing.T) {
	require.True(t, IsImageGenerationModel("gpt-image-1"))
	require.True(t, IsImageGenerationModel("models/gemini-2.5-flash-image"))
	require.False(t, IsImageGenerationModel("gpt-5"))
	require.False(t, IsImageGenerationModel("gemini-2.5-flash"))
}

func TestParseOpenAIImageGenerationRequest(t *testing.T) {
	req, err := ParseOpenAIImageGenerationRequest([]byte(`{"model":"gemini-2.5-flash-image","prompt":"draw a cat"}`))
	require.NoError(t, err)
	require.Equal(t, 1, req.N)
	require.Equal(t, "b64_json", req.ResponseFormat)

	_, err = ParseOpenAIImageGenerationRequest([]byte(`{"model":"gemini-2.5-flash-image","prompt":"draw a cat","n":2}`))
	require.EqualError(t, err, "only n=1 is currently supported")
}

func TestBuildGeminiImageGenerationRequest(t *testing.T) {
	body, imageSize, err := BuildGeminiImageGenerationRequest(&OpenAIImageGenerationRequest{
		Model:  "gemini-2.5-flash-image",
		Prompt: "draw a cat",
		Size:   "1536x1024",
		N:      1,
	})
	require.NoError(t, err)
	require.Equal(t, "2K", imageSize)
	require.Equal(t, "draw a cat", gjson.GetBytes(body, "contents.0.parts.0.text").String())
	require.Equal(t, "3:2", gjson.GetBytes(body, "generationConfig.imageConfig.aspectRatio").String())
	require.Equal(t, "2K", gjson.GetBytes(body, "generationConfig.imageConfig.imageSize").String())
}

func TestParseGeminiImageGenerationResponse(t *testing.T) {
	resp, err := ParseGeminiImageGenerationResponse([]byte(`{
		"candidates":[{
			"content":{
				"parts":[
					{"text":"updated prompt"},
					{"inlineData":{"mimeType":"image/png","data":"ZmFrZQ=="}}
				]
			}
		}]
	}`))
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	require.Equal(t, "ZmFrZQ==", resp.Data[0].B64JSON)
	require.Equal(t, "updated prompt", resp.Data[0].RevisedPrompt)
}
