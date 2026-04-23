package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOpenAIAvailableModels_ParsesChatGPTModelPayload(t *testing.T) {
	body := []byte(`{
		"categories": [
			{
				"title": "GPT",
				"models": [
					{"slug": "gpt-5.5", "title": "GPT-5.5"},
					{"slug": "gpt-image-2", "title": "GPT Image 2"},
					{"slug": "internal-tooling"}
				]
			}
		]
	}`)

	models, err := parseOpenAIAvailableModels(body)
	require.NoError(t, err)
	require.Len(t, models, 2)
	require.Equal(t, "gpt-5.5", models[0].ID)
	require.Equal(t, "GPT-5.5", models[0].DisplayName)
	require.Equal(t, "gpt-image-2", models[1].ID)
	require.Equal(t, "GPT Image 2", models[1].DisplayName)
}

func TestParseOpenAIAvailableModels_PreservesUnknownQueriedOpenAIIDs(t *testing.T) {
	body := []byte(`{
		"data": [
			{"id": "gpt-5.5", "display_name": "GPT-5.5"},
			{"id": "gpt-5.5", "display_name": "GPT-5.5"},
			{"id": "gpt-5.2-pro", "display_name": "GPT-5.2 Pro"}
		]
	}`)

	models, err := parseOpenAIAvailableModels(body)
	require.NoError(t, err)
	require.Len(t, models, 2)
	require.Equal(t, "gpt-5.2-pro", models[0].ID)
	require.Equal(t, "gpt-5.5", models[1].ID)
}
