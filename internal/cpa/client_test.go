package cpa

import (
	"context"
	"cpa-quota/internal/config"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
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
			case antigravityQuotaURL:
				writeEnvelope(t, w, map[string]any{"groups": []any{
					map[string]any{
						"displayName": "Claude and GPT models",
						"buckets": []any{
							map[string]any{"bucketId": "3p-5h", "window": "5h", "remainingFraction": 0.4, "resetTime": "2030-02-01T10:00:00Z"},
							map[string]any{"bucketId": "3p-weekly", "window": "weekly", "remainingFraction": 0.7, "resetTime": "2030-02-01T11:00:00Z"},
						},
					},
					map[string]any{
						"displayName": "Gemini Models",
						"buckets": []any{
							map[string]any{"bucketId": "gemini-5h", "window": "5h", "remainingFraction": 0.8, "resetTime": "2030-02-01T12:00:00Z"},
							map[string]any{"bucketId": "gemini-weekly", "window": "weekly", "remainingFraction": 0.9, "resetTime": "2030-02-01T13:00:00Z"},
						},
					},
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
	if len(antigravity.Accounts[0].Windows) != 4 {
		t.Fatalf("Antigravity windows count = %d, want 4", len(antigravity.Accounts[0].Windows))
	}
	assertRemaining(t, antigravity.Accounts[0].Windows[0], 40)
	assertRemaining(t, antigravity.Accounts[0].Windows[1], 70)
	assertRemaining(t, antigravity.Accounts[0].Windows[2], 80)
	assertRemaining(t, antigravity.Accounts[0].Windows[3], 90)

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

func TestParseAntigravityQuotaSummary(t *testing.T) {
	t.Parallel()

	raw := `{
  "groups": [
    {
      "buckets": [
        {
          "bucketId": "gemini-weekly",
          "displayName": "Weekly Limit Remaining",
          "window": "weekly",
          "resetTime": "2026-09-23T02:33:39Z",
          "description": "You have used some of your weekly limit, it will fully refresh in 6 days, 17 hours.",
          "remainingFraction": 0.8856122
        },
        {
          "bucketId": "gemini-5h",
          "displayName": "Five Hour Limit Remaining",
          "window": "5h",
          "resetTime": "2026-09-16T12:33:39Z",
          "description": "You have used some of your 5-hour limit, it will fully refresh in 3 hours.",
          "remainingFraction": 0.92559963
        }
      ],
      "displayName": "Gemini Models",
      "description": "Models within this group: Gemini Flash, Gemini Pro"
    },
    {
      "buckets": [
        {
          "bucketId": "3p-weekly",
          "displayName": "Weekly Limit Remaining",
          "window": "weekly",
          "resetTime": "2026-09-22T09:05:25Z",
          "description": "You have hit your 5-hour limit, so the weekly limit does not currently apply. Your 5-hour limit will refresh in 2 hours, 5 minutes.",
          "remainingFraction": 0.2524704
        },
        {
          "bucketId": "3p-5h",
          "displayName": "Five Hour Limit Remaining",
          "window": "5h",
          "resetTime": "2026-09-16T11:38:38Z",
          "description": "You have hit your 5-hour limit, it will refresh in 2 hours, 5 minutes. If on a supported paid plan, you can use AI credits in the interim.",
          "remainingFraction": 0
        }
      ],
      "displayName": "Claude and GPT models",
      "description": "Models within this group: Claude Opus, Claude Sonnet, GPT-OSS"
    }
  ]
}`

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal test payload: %v", err)
	}

	windows := parseAntigravityQuotaSummary(payload)
	if len(windows) != 4 {
		t.Fatalf("got %d windows, want 4", len(windows))
	}

	// Claude 5-hour
	if windows[0].Label != "Claude 5-hour" || windows[0].Remaining == nil || *windows[0].Remaining != 0 {
		t.Fatalf("Claude 5-hour = %#v", windows[0])
	}
	if windows[0].ResetAt == nil || windows[0].ResetAt.Format(time.RFC3339) != "2026-09-16T11:38:38Z" {
		t.Fatalf("Claude 5-hour reset = %v", windows[0].ResetAt)
	}

	// Claude Weekly
	if windows[1].Label != "Claude Weekly" || windows[1].Remaining == nil || math.Abs(*windows[1].Remaining-25.24704) > 0.001 {
		t.Fatalf("Claude Weekly = %#v", windows[1])
	}

	// Gemini 5-hour
	if windows[2].Label != "Gemini 5-hour" || windows[2].Remaining == nil || math.Abs(*windows[2].Remaining-92.559963) > 0.001 {
		t.Fatalf("Gemini 5-hour = %#v", windows[2])
	}

	// Gemini Weekly
	if windows[3].Label != "Gemini Weekly" || windows[3].Remaining == nil || math.Abs(*windows[3].Remaining-88.56122) > 0.001 {
		t.Fatalf("Gemini Weekly = %#v", windows[3])
	}
}
