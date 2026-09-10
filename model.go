package main

import "time"

const (
	pluginID       = "herdr-cliproxyapi-quota-plugin"
	paneEntrypoint = "quota"
)

type Config struct {
	BaseURL       string `json:"base_url"`
	ManagementKey string `json:"management_key"`
}

type Provider string

const (
	ProviderCodex       Provider = "codex"
	ProviderAntigravity Provider = "antigravity"
	ProviderClaude      Provider = "claude"
)

type QuotaWindow struct {
	Label     string
	Remaining *float64
	ResetAt   *time.Time
}

type AccountQuota struct {
	Provider     Provider
	Name         string
	Status       string
	Disabled     bool
	Unavailable  bool
	Windows      []QuotaWindow
	ManualResets *int
	Error        string
}

type ProviderQuota struct {
	Provider Provider
	Title    string
	Accounts []AccountQuota
}

type Snapshot struct {
	Groups         []ProviderQuota
	FetchedAt      time.Time
	CurrentVersion string
}
