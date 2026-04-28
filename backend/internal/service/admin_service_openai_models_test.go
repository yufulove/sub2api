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
		{ID: "gpt-5", DisplayName: "GPT-5"},
		{ID: "gpt-5-mini", DisplayName: "GPT-5 Mini"},
		{ID: "o3", DisplayName: "OpenAI o3"},
		{ID: "gpt-5.3-codex-spark", DisplayName: "GPT-5.3 Codex Spark"},
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
	})

	require.Len(t, models, 4)
	require.Equal(t, "gpt-5.4", models[0].ID)
	require.Equal(t, "gpt-5.4-mini", models[1].ID)
	require.Equal(t, "gpt-5.5", models[2].ID)
	require.Equal(t, "gpt-image-2", models[3].ID)
}

func TestMergeOpenAIOAuthCodexCompatibleModels_PreservesCuratedFallback(t *testing.T) {
	models := mergeOpenAIOAuthCodexCompatibleModels(OpenAIOAuthCodexCompatibleModels(), []openaipkg.Model{
		{ID: "gpt-5.4", DisplayName: "GPT-5.4 discovered"},
		{ID: "gpt-5.4-mini", DisplayName: "GPT-5.4 Mini discovered"},
	})

	var ids []string
	for _, model := range models {
		ids = append(ids, model.ID)
	}

	require.Contains(t, ids, "gpt-5.5")
	require.Contains(t, ids, "gpt-5.4")
	require.Contains(t, ids, "gpt-5.4-mini")
	require.Contains(t, ids, "gpt-image-1")
	require.Contains(t, ids, "gpt-image-1.5")
	require.Contains(t, ids, "gpt-image-2")
	require.Equal(t, "GPT-5.4 discovered", models[1].DisplayName)
}

func TestResolveOpenAIOAuthCodexTestModel_NormalizesSupportedAliases(t *testing.T) {
	modelID, ok := resolveOpenAIOAuthCodexTestModel("gpt-5.1-codex")
	require.True(t, ok)
	require.Equal(t, "gpt-5.3-codex", modelID)

	modelID, ok = resolveOpenAIOAuthCodexTestModel("gpt-5-mini")
	require.True(t, ok)
	require.Equal(t, "gpt-5.4-mini", modelID)

	modelID, ok = resolveOpenAIOAuthCodexTestModel("gpt-5.3-codex-spark")
	require.False(t, ok)
	require.Empty(t, modelID)

	modelID, ok = resolveOpenAIOAuthCodexTestModel("o3")
	require.False(t, ok)
	require.Empty(t, modelID)
}

func TestOpenAIOAuthCodexCompatibleModels_ExcludeUnsupportedSparkVariant(t *testing.T) {
	models := OpenAIOAuthCodexCompatibleModels()
	require.NotEmpty(t, models)

	var ids []string
	for _, model := range models {
		ids = append(ids, model.ID)
	}

	require.Contains(t, ids, "gpt-5.4")
	require.Contains(t, ids, "gpt-5.4-mini")
	require.Contains(t, ids, "gpt-5.5")
	require.Contains(t, ids, "gpt-5.3-codex")
	require.Contains(t, ids, "gpt-5.2")
	require.NotContains(t, ids, "gpt-5.3-codex-spark")
}
