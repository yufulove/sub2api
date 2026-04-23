package service

import (
	"testing"

	openaipkg "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
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

func TestFilterOpenAIOAuthCodexCompatibleModels_RejectsUnsupportedCatalogModels(t *testing.T) {
	models := filterOpenAIOAuthCodexCompatibleModels([]openaipkg.Model{
		{ID: "gpt-5.5", DisplayName: "GPT-5.5"},
		{ID: "o3", DisplayName: "OpenAI o3"},
		{ID: "gpt-5.2-pro", DisplayName: "GPT-5.2 Pro"},
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
	})

	require.Len(t, models, 2)
	require.Equal(t, "gpt-5.5", models[0].ID)
	require.Equal(t, "gpt-image-2", models[1].ID)
}

func TestResolveOpenAIOAuthCodexTestModel_NormalizesSupportedAliases(t *testing.T) {
	modelID, ok := resolveOpenAIOAuthCodexTestModel("gpt-5.1-codex")
	require.True(t, ok)
	require.Equal(t, "gpt-5.3-codex", modelID)

	modelID, ok = resolveOpenAIOAuthCodexTestModel("o3")
	require.False(t, ok)
	require.Empty(t, modelID)
}
