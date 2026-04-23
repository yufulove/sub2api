package service

import (
	"sort"
	"strings"

	openaipkg "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

var strictOpenAIOAuthLegacyCodexModelMap = map[string]string{
	"gpt-5":              "gpt-5.4",
	"gpt-5-mini":         "gpt-5.4",
	"gpt-5-nano":         "gpt-5.4",
	"gpt-5.1":            "gpt-5.4",
	"gpt-5.1-codex":      "gpt-5.3-codex",
	"gpt-5.1-codex-max":  "gpt-5.3-codex",
	"gpt-5.1-codex-mini": "gpt-5.3-codex",
	"gpt-5.2-codex":      "gpt-5.2",
	"gpt-5-codex":        "gpt-5.3-codex",
	"codex-mini-latest":  "gpt-5.3-codex",
}

var strictOpenAIOAuthImageModelMap = buildStrictOpenAIOAuthImageModelMap()

func buildStrictOpenAIOAuthImageModelMap() map[string]string {
	supported := make(map[string]string)
	for _, model := range openaipkg.DefaultModels {
		id := strings.TrimSpace(model.ID)
		if !isOpenAIImageGenerationModel(id) {
			continue
		}
		supported[strings.ToLower(id)] = id
	}
	return supported
}

func resolveOpenAIOAuthCodexTestModel(modelID string) (string, bool) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return openaipkg.DefaultTestModel, true
	}

	if strings.Contains(modelID, "/") {
		parts := strings.Split(modelID, "/")
		modelID = strings.TrimSpace(parts[len(parts)-1])
	}
	if modelID == "" {
		return "", false
	}

	if mapped := getNormalizedCodexModel(modelID); mapped != "" {
		return mapped, true
	}

	lower := strings.ToLower(modelID)
	if mapped, ok := strictOpenAIOAuthLegacyCodexModelMap[lower]; ok {
		return mapped, true
	}
	if imageModel, ok := strictOpenAIOAuthImageModelMap[lower]; ok {
		return imageModel, true
	}
	return "", false
}

func filterOpenAIOAuthCodexCompatibleModels(models []openaipkg.Model) []openaipkg.Model {
	filtered := make([]openaipkg.Model, 0, len(models))
	seen := make(map[string]struct{}, len(models))

	for _, model := range models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		if _, ok := resolveOpenAIOAuthCodexTestModel(id); !ok {
			continue
		}

		key := strings.ToLower(id)
		if _, exists := seen[key]; exists {
			continue
		}

		model.ID = id
		if strings.TrimSpace(model.Object) == "" {
			model.Object = "model"
		}
		if strings.TrimSpace(model.Type) == "" {
			model.Type = "model"
		}
		if strings.TrimSpace(model.DisplayName) == "" {
			model.DisplayName = id
		}
		if strings.TrimSpace(model.OwnedBy) == "" {
			model.OwnedBy = "openai"
		}

		filtered = append(filtered, model)
		seen[key] = struct{}{}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].ID < filtered[j].ID
	})

	return filtered
}
