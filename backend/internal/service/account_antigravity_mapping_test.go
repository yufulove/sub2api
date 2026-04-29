package service

import "testing"

func TestAccountGetModelMappingAntigravityRepairsStaleProImageMappingRuntime(t *testing.T) {
	account := &Account{
		Platform: PlatformAntigravity,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"gemini-3-pro-image":         "gemini-3.1-flash-image",
				"gemini-3-pro-image-preview": "gemini-3.1-flash-image",
			},
		},
	}

	if mapped := account.GetMappedModel("gemini-3-pro-image"); mapped != "gemini-3-pro-image" {
		t.Fatalf("expected stale pro image mapping to be repaired, got: %q", mapped)
	}
	if mapped := account.GetMappedModel("gemini-3-pro-image-preview"); mapped != "gemini-3-pro-image" {
		t.Fatalf("expected stale pro image preview mapping to canonical pro image, got: %q", mapped)
	}
}

func TestAccountGetModelMappingAntigravityPreservesCustomProImageMappingRuntime(t *testing.T) {
	account := &Account{
		Platform: PlatformAntigravity,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"gemini-3-pro-image": "custom-pro-image",
			},
		},
	}

	if mapped := account.GetMappedModel("gemini-3-pro-image"); mapped != "custom-pro-image" {
		t.Fatalf("expected custom pro image mapping to be preserved, got: %q", mapped)
	}
}
