package service

import (
	"sort"
	"strings"

	openaipkg "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

var openAIOAuthCodexCompatibleModelIDs = []string{
	"gpt-5.5",
	"gpt-5.4",
	"gpt-5.4-mini",
	"gpt-5.3-codex",
	"gpt-5.2",
	"gpt-image-1",
	"gpt-image-1.5",
	"gpt-image-2",
}

var strictOpenAIOAuthLegacyCodexModelMap = map[string]string{
	"gpt-5":              "gpt-5.4",
	"gpt-5-mini":         "gpt-5.4-mini",
	"gpt-5.1":            "gpt-5.4",
	"gpt-5.1-codex":      "gpt-5.3-codex",
	"gpt-5.1-codex-max":  "gpt-5.3-codex",
	"gpt-5.1-codex-mini": "gpt-5.3-codex",
	"gpt-5.2-codex":      "gpt-5.2",
	"gpt-5-codex":        "gpt-5.3-codex",
	"codex-mini-latest":  "gpt-5.3-codex",
}

var strictOpenAIOAuthImageModelMap = buildStrictOpenAIOAuthImageModelMap()
var openAIOAuthCodexCompatibleModelsByID = buildOpenAIOAuthCodexCompatibleModelsByID()

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

func buildOpenAIOAuthCodexCompatibleModelsByID() map[string]openaipkg.Model {
	modelsByID := make(map[string]openaipkg.Model, len(openAIOAuthCodexCompatibleModelIDs))
	for _, model := range openaipkg.DefaultModels {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		modelsByID[id] = model
	}

	curated := make(map[string]openaipkg.Model, len(openAIOAuthCodexCompatibleModelIDs))
	for _, id := range openAIOAuthCodexCompatibleModelIDs {
		if model, ok := modelsByID[id]; ok {
			curated[id] = model
		}
	}
	return curated
}

func OpenAIOAuthCodexCompatibleModels() []openaipkg.Model {
	models := make([]openaipkg.Model, 0, len(openAIOAuthCodexCompatibleModelIDs))
	for _, id := range openAIOAuthCodexCompatibleModelIDs {
		if model, ok := openAIOAuthCodexCompatibleModelsByID[id]; ok {
			models = append(models, model)
		}
	}
	return models
}

func mergeOpenAIOAuthCodexCompatibleModels(base []openaipkg.Model, discovered []openaipkg.Model) []openaipkg.Model {
	models := make([]openaipkg.Model, 0, len(base)+len(discovered))
	indexByID := make(map[string]int, len(base)+len(discovered))

	for _, model := range base {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, exists := indexByID[key]; exists {
			continue
		}
		indexByID[key] = len(models)
		models = append(models, model)
	}

	for _, model := range discovered {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if index, exists := indexByID[key]; exists {
			models[index] = model
			continue
		}
		indexByID[key] = len(models)
		models = append(models, model)
	}

	return models
}

func isOpenAIOAuthCodexCompatibleCanonicalModelID(id string) bool {
	_, ok := openAIOAuthCodexCompatibleModelsByID[strings.TrimSpace(id)]
	return ok
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

	lower := strings.ToLower(modelID)
	if mapped, ok := strictOpenAIOAuthLegacyCodexModelMap[lower]; ok {
		return mapped, true
	}
	if imageModel, ok := strictOpenAIOAuthImageModelMap[lower]; ok {
		return imageModel, true
	}
	if isOpenAIOAuthCodexCompatibleCanonicalModelID(modelID) {
		return modelID, true
	}
	if mapped := getNormalizedCodexModel(modelID); mapped != "" && isOpenAIOAuthCodexCompatibleCanonicalModelID(mapped) {
		return mapped, true
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
		canonicalID, ok := resolveOpenAIOAuthCodexTestModel(id)
		if !ok {
			continue
		}

		key := strings.ToLower(canonicalID)
		if _, exists := seen[key]; exists {
			continue
		}

		if canonicalModel, exists := openAIOAuthCodexCompatibleModelsByID[canonicalID]; exists {
			filtered = append(filtered, canonicalModel)
		} else {
			model.ID = canonicalID
			if strings.TrimSpace(model.Object) == "" {
				model.Object = "model"
			}
			if strings.TrimSpace(model.Type) == "" {
				model.Type = "model"
			}
			if strings.TrimSpace(model.DisplayName) == "" {
				model.DisplayName = canonicalID
			}
			if strings.TrimSpace(model.OwnedBy) == "" {
				model.OwnedBy = "openai"
			}
			filtered = append(filtered, model)
		}
		seen[key] = struct{}{}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].ID < filtered[j].ID
	})

	return filtered
}
