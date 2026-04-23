package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	openaipkg "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

const (
	adminOpenAIAvailableModelsURL           = "https://chatgpt.com/backend-api/models"
	adminOpenAIAvailableModelsTimeout       = 15 * time.Second
	adminOpenAIAvailableModelsHeaderTimeout = 10 * time.Second
	adminOpenAIAvailableModelsMaxBodyBytes  = 1 << 20
	adminOpenAIAvailableModelsDefaultUA     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"
)

// OpenAIAvailableModelsProvider is an optional capability exposed by admin services
// that can probe an OpenAI account and return the currently available model list.
type OpenAIAvailableModelsProvider interface {
	GetOpenAIAvailableModels(ctx context.Context, account *Account) ([]openaipkg.Model, error)
}

func (s *adminServiceImpl) GetOpenAIAvailableModels(ctx context.Context, account *Account) ([]openaipkg.Model, error) {
	if s == nil {
		return nil, fmt.Errorf("admin service is nil")
	}
	if account == nil || !account.IsOpenAI() {
		return nil, fmt.Errorf("account is not openai")
	}

	switch {
	case account.IsOpenAIOAuth():
		return OpenAIOAuthCodexCompatibleModels(), nil
	case account.IsOpenAIApiKey():
		return s.fetchOpenAIApiKeyAvailableModels(ctx, account)
	default:
		return nil, fmt.Errorf("unsupported openai account type: %s", strings.TrimSpace(account.Type))
	}
}

func (s *adminServiceImpl) fetchOpenAIOAuthAvailableModels(ctx context.Context, account *Account) ([]openaipkg.Model, error) {
	accessToken := strings.TrimSpace(account.GetOpenAIAccessToken())
	if accessToken == "" {
		return nil, fmt.Errorf("openai oauth access token is missing")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adminOpenAIAvailableModelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create openai oauth models request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://chatgpt.com")
	req.Header.Set("Referer", "https://chatgpt.com/")
	req.Header.Set("User-Agent", coalesceOpenAIModelListUserAgent(account.GetOpenAIUserAgent()))
	if chatgptAccountID := strings.TrimSpace(account.GetChatGPTAccountID()); chatgptAccountID != "" {
		req.Header.Set("chatgpt-account-id", chatgptAccountID)
	}

	models, err := s.doOpenAIAvailableModelsRequest(req, account)
	if err != nil {
		return nil, err
	}

	models = filterOpenAIOAuthCodexCompatibleModels(models)
	if len(models) == 0 {
		return nil, fmt.Errorf("openai models response did not include codex-compatible models")
	}
	return models, nil
}

func (s *adminServiceImpl) fetchOpenAIApiKeyAvailableModels(ctx context.Context, account *Account) ([]openaipkg.Model, error) {
	apiKey := strings.TrimSpace(account.GetOpenAIApiKey())
	if apiKey == "" {
		return nil, fmt.Errorf("openai api key is missing")
	}

	targetURL := buildAdminOpenAIModelsURL(account.GetOpenAIBaseURL())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create openai apikey models request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", coalesceOpenAIModelListUserAgent(account.GetOpenAIUserAgent()))

	return s.doOpenAIAvailableModelsRequest(req, account)
}

func (s *adminServiceImpl) doOpenAIAvailableModelsRequest(req *http.Request, account *Account) ([]openaipkg.Model, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	proxyURL := ""
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL:              proxyURL,
		Timeout:               adminOpenAIAvailableModelsTimeout,
		ResponseHeaderTimeout: adminOpenAIAvailableModelsHeaderTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("build openai models client: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai models request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, adminOpenAIAvailableModelsMaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read openai models response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai models request returned status %d", resp.StatusCode)
	}

	models, err := parseOpenAIAvailableModels(body)
	if err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("openai models response did not include supported models")
	}
	return models, nil
}

func buildAdminOpenAIModelsURL(baseURL string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		return "https://api.openai.com/v1/models"
	}
	if strings.HasSuffix(normalized, "/v1/models") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/models"
	}
	return normalized + "/v1/models"
}

func coalesceOpenAIModelListUserAgent(custom string) string {
	custom = strings.TrimSpace(custom)
	if custom != "" {
		return custom
	}
	return adminOpenAIAvailableModelsDefaultUA
}

func parseOpenAIAvailableModels(body []byte) ([]openaipkg.Model, error) {
	if !json.Valid(body) {
		return nil, fmt.Errorf("invalid openai models response")
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode openai models response: %w", err)
	}

	seen := make(map[string]openaipkg.Model)
	order := make([]string, 0, 32)
	collectOpenAIAvailableModels(payload, seen, &order)

	models := make([]openaipkg.Model, 0, len(order))
	for _, id := range order {
		if model, ok := seen[id]; ok {
			models = append(models, model)
		}
	}

	sort.SliceStable(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})
	return models, nil
}

func collectOpenAIAvailableModels(node any, seen map[string]openaipkg.Model, order *[]string) {
	switch value := node.(type) {
	case map[string]any:
		id := extractOpenAIAvailableModelID(value)
		if isLikelyOpenAIModelID(id) {
			id = strings.TrimSpace(id)
			if _, exists := seen[id]; !exists {
				model := openaipkg.Model{
					ID:          id,
					Object:      "model",
					Type:        "model",
					DisplayName: extractOpenAIAvailableModelDisplayName(value, id),
				}
				if created := extractOpenAIAvailableModelCreated(value); created > 0 {
					model.Created = created
				}
				if ownedBy := extractOpenAIAvailableModelOwnedBy(value); ownedBy != "" {
					model.OwnedBy = ownedBy
				} else {
					model.OwnedBy = "openai"
				}
				seen[id] = model
				*order = append(*order, id)
			}
		}
		for _, child := range value {
			collectOpenAIAvailableModels(child, seen, order)
		}
	case []any:
		for _, child := range value {
			collectOpenAIAvailableModels(child, seen, order)
		}
	}
}

func extractOpenAIAvailableModelID(node map[string]any) string {
	return firstNonEmptyModelString(
		stringValue(node["id"]),
		stringValue(node["slug"]),
		stringValue(node["model_slug"]),
	)
}

func extractOpenAIAvailableModelDisplayName(node map[string]any, fallback string) string {
	return firstNonEmptyModelString(
		stringValue(node["display_name"]),
		stringValue(node["title"]),
		stringValue(node["name"]),
		strings.TrimSpace(fallback),
	)
}

func extractOpenAIAvailableModelOwnedBy(node map[string]any) string {
	return firstNonEmptyModelString(
		stringValue(node["owned_by"]),
		stringValue(node["owner"]),
	)
}

func extractOpenAIAvailableModelCreated(node map[string]any) int64 {
	for _, value := range []any{node["created"], node["created_at"]} {
		switch v := value.(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case int:
			return int64(v)
		}
	}
	return 0
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func firstNonEmptyModelString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isLikelyOpenAIModelID(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return false
	}
	return strings.HasPrefix(id, "gpt-") ||
		strings.HasPrefix(id, "chatgpt-") ||
		strings.HasPrefix(id, "o1") ||
		strings.HasPrefix(id, "o3") ||
		strings.HasPrefix(id, "o4")
}
