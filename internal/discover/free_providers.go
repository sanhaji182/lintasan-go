package discover

// FreeProvider represents a free AI provider that can be auto-connected.
type FreeProvider struct {
	Name      string   `json:"name"`
	Prefix    string   `json:"prefix"`
	BaseURL   string   `json:"base_url"`
	Models    []string `json:"models"`
	AuthType  string   `json:"auth_type"` // "none", "apikey", "oauth"
	QuotaInfo string   `json:"quota_info"`
	Enabled   bool     `json:"enabled"`
}

// GetFreeProviders returns the registry of known free AI providers.
func GetFreeProviders() []FreeProvider {
	return []FreeProvider{
		{
			Name:     "Kiro AI",
			Prefix:   "kr/",
			BaseURL:  "https://api.kiro.ai/v1",
			Models:   []string{"claude-sonnet-4.5", "claude-haiku-4.5", "glm-5"},
			AuthType: "apikey",
			QuotaInfo: "50 credits/month, unlimited free tier",
		},
		{
			Name:     "OpenCode Free",
			Prefix:   "oc/",
			BaseURL:  "https://api.opencode.ai/v1",
			Models:   []string{"auto"},
			AuthType: "none",
			QuotaInfo: "Unlimited, no auth needed",
		},
		{
			Name:     "Pollinations",
			Prefix:   "pol/",
			BaseURL:  "https://text.pollinations.ai/v1",
			Models:   []string{"gpt-5", "claude", "gemini", "deepseek", "llama-4"},
			AuthType: "none",
			QuotaInfo: "No key needed, unlimited",
		},
		{
			Name:     "Cloudflare AI",
			Prefix:   "cf/",
			BaseURL:  "https://api.cloudflare.com/client/v4/accounts/placeholder/ai/v1",
			Models:   []string{"llama-3.3-70b", "qwen-2.5-32b", "mistral-7b"},
			AuthType: "apikey",
			QuotaInfo: "10K neurons/day, 50+ models",
		},
		{
			Name:     "NVIDIA NIM",
			Prefix:   "nvidia/",
			BaseURL:  "https://integrate.api.nvidia.com/v1",
			Models:   []string{"llama-3.3-70b", "qwen-2.5-72b", "deepseek-r1"},
			AuthType: "apikey",
			QuotaInfo: "~40 RPM, 129 models",
		},
		{
			Name:     "Cerebras",
			Prefix:   "cerebras/",
			BaseURL:  "https://api.cerebras.ai/v1",
			Models:   []string{"qwen3-235b", "llama-3.3-70b"},
			AuthType: "apikey",
			QuotaInfo: "1M tokens/day, fast inference",
		},
		{
			Name:     "LongCat",
			Prefix:   "lc/",
			BaseURL:  "https://api.longcat.ai/v1",
			Models:   []string{"longcat-flash-lite"},
			AuthType: "none",
			QuotaInfo: "50M tokens/day",
		},
	}
}

// GetFreeProviderByPrefix looks up a free provider by its model prefix.
func GetFreeProviderByPrefix(prefix string) *FreeProvider {
	for _, p := range GetFreeProviders() {
		if p.Prefix == prefix {
			return &p
		}
	}
	return nil
}

// GetFreeProviderByName looks up a free provider by name.
func GetFreeProviderByName(name string) *FreeProvider {
	for _, p := range GetFreeProviders() {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

// GetAllFreeModels returns all models from all free providers.
func GetAllFreeModels() map[string][]string {
	result := make(map[string][]string)
	for _, p := range GetFreeProviders() {
		result[p.Prefix] = p.Models
	}
	return result
}
