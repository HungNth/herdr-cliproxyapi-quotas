package cpa

import (
	"cpa-quota/internal/config"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestFetchSnapshotUsesOnlyReadOnlyManagementEndpoints(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var paths []string
	var handlerErrors []string
	recordError := func(message string) {
		mu.Lock()
		defer mu.Unlock()
		handlerErrors = append(handlerErrors, message)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer secret" {
			recordError("missing management authorization")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case authFilesPath:
			if r.Method != http.MethodGet {
				recordError("auth-files did not use GET")
			}
			writeJSON(t, w, map[string]any{"files": []any{
				map[string]any{
					"provider":   "codex",
					"auth_index": "codex-1",
					"email":      "codex@example.com",
					"id_token": map[string]any{
						"chatgpt_account_id": "account-1",
					},
				},
				map[string]any{
					"provider":    "antigravity",
					"auth_index":  "ag-1",
					"email":       "ag@example.com",
					"project_id":  "project-1",
					"unavailable": true,
				},
				map[string]any{
					"provider":     "claude",
					"auth_index":   "claude-1",
					"email":        "claude@example.com",
					"account_type": "oauth",
				},
				map[string]any{
					"provider":     "claude",
					"auth_index":   "claude-key",
					"email":        "api-key@example.com",
					"account_type": "api_key",
				},
			}})
		case apiCallPath:
			if r.Method != http.MethodPost {
				recordError("api-call did not use POST")
			}
			var call apiCallRequest
			if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
				recordError("invalid api-call payload: " + err.Error())
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			switch call.URL {
			case codexUsageURL:
				writeEnvelope(t, w, map[string]any{
					"plan_type": "plus",
					"rate_limit": map[string]any{
						"primary_window":   map[string]any{"used_percent": 25, "reset_at": 2_000_000_000},
						"secondary_window": map[string]any{"used_percent": 60, "reset_at": 2_000_100_000},
					},
				})
			case codexResetCreditsURL:
				writeEnvelope(t, w, map[string]any{"available_count": 2})
			case claudeUsageURL:
				writeEnvelope(t, w, map[string]any{
					"five_hour": map[string]any{"utilization": 10, "resets_at": "2030-01-01T10:00:00Z"},
					"seven_day": map[string]any{"utilization": 30, "resets_at": "2030-01-07T10:00:00Z"},
				})
			case antigravityQuotaURLs[0]:
				writeEnvelope(t, w, map[string]any{"models": map[string]any{
					"claude-opus-4-6-thinking": map[string]any{"quotaInfo": map[string]any{"remainingFraction": 0.4, "resetTime": "2030-02-01T10:00:00Z"}},
					"gpt-oss-120b-medium":      map[string]any{"quotaInfo": map[string]any{"remainingFraction": 0.7, "resetTime": "2030-02-01T11:00:00Z"}},
					"gemini-3.1-pro-high":      map[string]any{"quotaInfo": map[string]any{"remainingFraction": 0.8, "resetTime": "2030-02-01T12:00:00Z"}},
					"gemini-2.5-flash":         map[string]any{"quotaInfo": map[string]any{"remainingFraction": 0.01, "resetTime": "2030-02-01T13:00:00Z"}},
				}})
			default:
				recordError("unexpected upstream URL: " + call.URL)
				writeEnvelopeStatus(t, w, http.StatusBadRequest, map[string]any{"error": "unexpected URL"})
			}
		case latestVersionPath:
			if r.Method != http.MethodGet {
				recordError("latest-version did not use GET")
			}
			writeJSON(t, w, map[string]any{"latest-version": "v7.2.155"})
		default:
			recordError("unexpected management path: " + r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snapshot, err := NewClient(config.Config{BaseURL: server.URL, ManagementKey: "secret"}).FetchSnapshot(context.Background())
	if err != nil {
		t.Fatalf("FetchSnapshot() error = %v", err)
	}

	mu.Lock()
	if len(paths) == 0 {
		t.Fatal("no management calls recorded")
	}
	for _, path := range paths {
		if path != authFilesPath && path != apiCallPath && path != latestVersionPath {
			t.Fatalf("called endpoint outside the read-only boundary: %s", path)
		}
	}
	if len(snapshot.Groups) != 3 {
		t.Fatalf("groups = %d, want 3", len(snapshot.Groups))
	}
	codex := findGroup(t, snapshot, ProviderCodex)
	if len(codex.Accounts) != 1 || codex.Accounts[0].ManualResets == nil || *codex.Accounts[0].ManualResets != 2 {
		t.Fatalf("Codex account/manual resets = %#v", codex.Accounts)
	}
	assertRemaining(t, codex.Accounts[0].Windows[0], 75)
	assertRemaining(t, codex.Accounts[0].Windows[1], 40)

	antigravity := findGroup(t, snapshot, ProviderAntigravity)
	if len(antigravity.Accounts) != 1 || !antigravity.Accounts[0].Unavailable {
		t.Fatalf("Antigravity account = %#v", antigravity.Accounts)
	}
	assertRemaining(t, antigravity.Accounts[0].Windows[0], 40)
	assertRemaining(t, antigravity.Accounts[0].Windows[1], 80)

	claude := findGroup(t, snapshot, ProviderClaude)
	if len(claude.Accounts) != 1 || claude.Accounts[0].Name != "claude@example.com" {
		t.Fatalf("Claude accounts = %#v", claude.Accounts)
	}
	assertRemaining(t, claude.Accounts[0].Windows[0], 90)
	assertRemaining(t, claude.Accounts[0].Windows[1], 70)
}

func findGroup(t *testing.T, snapshot Snapshot, provider Provider) ProviderQuota {
	t.Helper()
	for _, group := range snapshot.Groups {
		if group.Provider == provider {
			return group
		}
	}
	t.Fatalf("provider group %q not found", provider)
	return ProviderQuota{}
}

func assertRemaining(t *testing.T, window QuotaWindow, want float64) {
	t.Helper()
	if window.Remaining == nil || math.Abs(*window.Remaining-want) > 0.001 {
		t.Fatalf("%s remaining = %v, want %.1f", window.Label, window.Remaining, want)
	}
}

func writeEnvelope(t *testing.T, w http.ResponseWriter, body map[string]any) {
	t.Helper()
	writeEnvelopeStatus(t, w, http.StatusOK, body)
}

func writeEnvelopeStatus(t *testing.T, w http.ResponseWriter, status int, body map[string]any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal envelope body: %v", err)
	}
	writeJSON(t, w, map[string]any{"status_code": status, "body": string(raw)})
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
