package cpa

import (
	"bytes"
	"context"
	"cpa-quota/internal/config"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	authFilesPath = "/v0/management/auth-files"
	apiCallPath   = "/v0/management/api-call"

	codexUsageURL        = "https://chatgpt.com/backend-api/wham/usage"
	codexResetCreditsURL = "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits"
	claudeUsageURL       = "https://api.anthropic.com/api/oauth/usage"
	googleLoadAssistURL  = "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist"
)

var antigravityQuotaURLs = []string{
	"https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels",
	"https://daily-cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels",
	"https://daily-cloudcode-pa.sandbox.googleapis.com/v1internal:fetchAvailableModels",
}

type Client struct {
	config config.Config
	http   *http.Client
}

type authEntry struct {
	raw map[string]any
}

type queryTask struct {
	provider Provider
	entry    authEntry
}

type apiCallRequest struct {
	AuthIndex string            `json:"auth_index"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Header    map[string]string `json:"header,omitempty"`
	Data      string            `json:"data,omitempty"`
}

type apiCallEnvelope struct {
	StatusCode      int             `json:"status_code"`
	StatusCodeCamel int             `json:"statusCode"`
	Body            json.RawMessage `json:"body"`
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		config: cfg.Normalized(),
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) FetchSnapshot(ctx context.Context) (Snapshot, error) {
	entries, currentVersion, err := c.fetchAuthFiles(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	tasks := make([]queryTask, 0, len(entries))
	for _, entry := range entries {
		provider, ok := classifyProvider(entry)
		if ok {
			tasks = append(tasks, queryTask{provider: provider, entry: entry})
		}
	}

	accounts := make([]AccountQuota, len(tasks))
	sem := make(chan struct{}, 8)
	var accountWG sync.WaitGroup
	for i := range tasks {
		i := i
		accountWG.Add(1)
		go func() {
			defer accountWG.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				accounts[i] = baseAccount(tasks[i], ctx.Err().Error())
				return
			}
			accounts[i] = c.queryAccount(ctx, tasks[i])
		}()
	}
	accountWG.Wait()

	groups := make([]ProviderQuota, 0, 3)
	for _, provider := range []Provider{ProviderCodex, ProviderAntigravity, ProviderClaude} {
		group := ProviderQuota{Provider: provider, Title: providerTitle(provider)}
		for _, account := range accounts {
			if account.Provider == provider {
				group.Accounts = append(group.Accounts, account)
			}
		}
		if len(group.Accounts) == 0 {
			continue
		}
		sort.SliceStable(group.Accounts, func(i, j int) bool {
			left, right := group.Accounts[i], group.Accounts[j]
			if (left.Error != "") != (right.Error != "") {
				return left.Error == ""
			}
			leftRemaining, rightRemaining := minimumRemaining(left), minimumRemaining(right)
			if leftRemaining != rightRemaining {
				return leftRemaining < rightRemaining
			}
			return strings.ToLower(left.Name) < strings.ToLower(right.Name)
		})
		groups = append(groups, group)
	}

	return Snapshot{Groups: groups, FetchedAt: time.Now(), CurrentVersion: currentVersion}, nil
}

func (c *Client) fetchAuthFiles(ctx context.Context) ([]authEntry, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(authFilesPath), nil)
	if err != nil {
		return nil, "", err
	}
	c.applyManagementAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("GET %s: %w", authFilesPath, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", authFilesPath, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("GET %s: HTTP %d: %s", authFilesPath, resp.StatusCode, compactError(raw))
	}
	var payload struct {
		Files []map[string]any `json:"files"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, "", fmt.Errorf("decode %s: %w", authFilesPath, err)
	}
	entries := make([]authEntry, len(payload.Files))
	for i, file := range payload.Files {
		entries[i] = authEntry{raw: file}
	}
	return entries, resp.Header.Get("X-Cpa-Version"), nil
}

func (c *Client) callUpstream(ctx context.Context, call apiCallRequest) (map[string]any, error) {
	rawRequest, err := json.Marshal(call)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(apiCallPath), bytes.NewReader(rawRequest))
	if err != nil {
		return nil, err
	}
	c.applyManagementAuth(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", apiCallPath, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", apiCallPath, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("POST %s: HTTP %d: %s", apiCallPath, resp.StatusCode, compactError(raw))
	}
	var envelope apiCallEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode %s: %w", apiCallPath, err)
	}
	status := envelope.StatusCode
	if status == 0 {
		status = envelope.StatusCodeCamel
	}
	body, bodyText, decodeErr := decodeUpstreamBody(envelope.Body)
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("upstream HTTP %d: %s", status, compactError([]byte(bodyText)))
	}
	if decodeErr != nil {
		return nil, decodeErr
	}
	return body, nil
}

func (c *Client) endpoint(path string) string {
	return strings.TrimRight(c.config.BaseURL, "/") + path
}

func (c *Client) applyManagementAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.config.ManagementKey)
	req.Header.Set("User-Agent", "cpa-quota/0.1")
}

func (c *Client) queryAccount(ctx context.Context, task queryTask) AccountQuota {
	switch task.provider {
	case ProviderCodex:
		return c.queryCodex(ctx, task)
	case ProviderAntigravity:
		return c.queryAntigravity(ctx, task)
	case ProviderClaude:
		return c.queryClaude(ctx, task)
	default:
		return baseAccount(task, "unsupported provider")
	}
}

func (c *Client) queryCodex(ctx context.Context, task queryTask) AccountQuota {
	account := baseAccount(task, "")
	account.Windows = []QuotaWindow{{Label: "5-hour"}, {Label: "Weekly"}}
	authIndex := entryString(task.entry, "auth_index", "authIndex")
	if authIndex == "" {
		account.Error = "missing auth_index"
		return account
	}
	headers := map[string]string{
		"Authorization": "Bearer $TOKEN$",
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"OpenAI-Beta":   "codex-1",
		"OAI-Language":  "en",
		"User-Agent":    "codex_cli_rs/0.76.0 (cpa-quota)",
		"originator":    "cpa-quota",
	}
	if accountID := codexAccountID(task.entry); accountID != "" {
		headers["Chatgpt-Account-Id"] = accountID
	}

	usage, err := c.callUpstream(ctx, apiCallRequest{
		AuthIndex: authIndex,
		Method:    http.MethodGet,
		URL:       codexUsageURL,
		Header:    headers,
	})
	if err != nil {
		account.Error = err.Error()
		return account
	}
	account.Windows = parseCodexWindows(usage, time.Now())

	credits, err := c.callUpstream(ctx, apiCallRequest{
		AuthIndex: authIndex,
		Method:    http.MethodGet,
		URL:       codexResetCreditsURL,
		Header:    headers,
	})
	if err == nil {
		account.ManualResets = parseManualResetCount(credits)
	}
	return account
}

func (c *Client) queryClaude(ctx context.Context, task queryTask) AccountQuota {
	account := baseAccount(task, "")
	account.Windows = []QuotaWindow{{Label: "5-hour"}, {Label: "Weekly"}}
	authIndex := entryString(task.entry, "auth_index", "authIndex")
	if authIndex == "" {
		account.Error = "missing auth_index"
		return account
	}
	usage, err := c.callUpstream(ctx, apiCallRequest{
		AuthIndex: authIndex,
		Method:    http.MethodGet,
		URL:       claudeUsageURL,
		Header: map[string]string{
			"Authorization":  "Bearer $TOKEN$",
			"Accept":         "application/json",
			"Content-Type":   "application/json",
			"anthropic-beta": "oauth-2025-04-20",
			"User-Agent":     "cpa-quota/0.1",
		},
	})
	if err != nil {
		account.Error = err.Error()
		return account
	}
	account.Windows = []QuotaWindow{
		parseClaudeWindow("5-hour", firstValue(usage["five_hour"], usage["fiveHour"])),
		parseClaudeWindow("Weekly", firstValue(usage["seven_day"], usage["sevenDay"])),
	}
	return account
}

func (c *Client) queryAntigravity(ctx context.Context, task queryTask) AccountQuota {
	account := baseAccount(task, "")
	account.Windows = []QuotaWindow{{Label: "Claude & GPT models"}, {Label: "Gemini models"}}
	authIndex := entryString(task.entry, "auth_index", "authIndex")
	if authIndex == "" {
		account.Error = "missing auth_index"
		return account
	}

	projectID := antigravityProjectID(task.entry)
	if projectID == "" {
		if payload, err := c.loadAntigravityProject(ctx, authIndex); err == nil {
			projectID = projectIDFromAssist(payload)
		}
	}
	requestData := map[string]any{}
	if projectID != "" {
		requestData["project"] = projectID
	}
	data, _ := json.Marshal(requestData)
	headers := map[string]string{
		"Authorization": "Bearer $TOKEN$",
		"Content-Type":  "application/json",
		"User-Agent":    "antigravity/1.11.5 cpa-quota",
	}
	var payload map[string]any
	var lastErr error
	for _, upstreamURL := range antigravityQuotaURLs {
		payload, lastErr = c.callUpstream(ctx, apiCallRequest{
			AuthIndex: authIndex,
			Method:    http.MethodPost,
			URL:       upstreamURL,
			Header:    headers,
			Data:      string(data),
		})
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		account.Error = lastErr.Error()
		return account
	}
	account.Windows = parseAntigravityFamilies(payload)
	if account.Windows[0].Remaining == nil && account.Windows[1].Remaining == nil {
		account.Error = "no supported model quota returned"
	}
	return account
}

func (c *Client) loadAntigravityProject(ctx context.Context, authIndex string) (map[string]any, error) {
	metadata := map[string]string{
		"ideType":    "ANTIGRAVITY",
		"platform":   "PLATFORM_UNSPECIFIED",
		"pluginType": "GEMINI",
	}
	metadataJSON, _ := json.Marshal(metadata)
	requestBody, _ := json.Marshal(map[string]any{"metadata": metadata})
	return c.callUpstream(ctx, apiCallRequest{
		AuthIndex: authIndex,
		Method:    http.MethodPost,
		URL:       googleLoadAssistURL,
		Header: map[string]string{
			"Authorization":     "Bearer $TOKEN$",
			"Content-Type":      "application/json",
			"User-Agent":        "google-api-nodejs-client/9.15.1",
			"X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
			"Client-Metadata":   string(metadataJSON),
		},
		Data: string(requestBody),
	})
}

func classifyProvider(entry authEntry) (Provider, bool) {
	provider := normalizeIdentifier(firstString(entry.raw["provider"], entry.raw["type"]))
	switch provider {
	case "codex", "openai-codex":
		return ProviderCodex, true
	case "antigravity":
		return ProviderAntigravity, true
	case "claude", "anthropic":
		if isAPIKeyCredential(entry) {
			return "", false
		}
		return ProviderClaude, true
	default:
		return "", false
	}
}

func isAPIKeyCredential(entry authEntry) bool {
	for _, value := range []any{
		entry.raw["auth_type"], entry.raw["authType"], entry.raw["account_type"], entry.raw["accountType"],
		nested(entry.raw, "metadata", "auth_type"), nested(entry.raw, "metadata", "account_type"),
	} {
		normalized := normalizeIdentifier(cleanString(value))
		if strings.Contains(normalized, "api-key") || normalized == "apikey" {
			return true
		}
	}
	return cleanString(entry.raw["api_key"]) != ""
}

func baseAccount(task queryTask, queryError string) AccountQuota {
	return AccountQuota{
		Provider:    task.provider,
		Name:        accountDisplayName(task.entry),
		Status:      entryString(task.entry, "status"),
		Disabled:    boolValue(task.entry.raw["disabled"]),
		Unavailable: boolValue(task.entry.raw["unavailable"]),
		Error:       queryError,
	}
}

func providerTitle(provider Provider) string {
	switch provider {
	case ProviderCodex:
		return "Codex"
	case ProviderAntigravity:
		return "Antigravity"
	case ProviderClaude:
		return "Claude"
	default:
		return string(provider)
	}
}

func minimumRemaining(account AccountQuota) float64 {
	minimum := 101.0
	for _, window := range account.Windows {
		if window.Remaining != nil && *window.Remaining < minimum {
			minimum = *window.Remaining
		}
	}
	return minimum
}

func parseCodexWindows(payload map[string]any, now time.Time) []QuotaWindow {
	primary, secondary := any(nil), any(nil)
	if rateLimit := asMap(firstValue(payload["rate_limit"], payload["rateLimit"])); rateLimit != nil {
		primary = firstValue(rateLimit["primary_window"], rateLimit["primaryWindow"])
		secondary = firstValue(rateLimit["secondary_window"], rateLimit["secondaryWindow"])
	} else {
		primary = firstValue(payload["5_hour_window"], payload["fiveHourWindow"])
		secondary = firstValue(payload["weekly_window"], payload["weeklyWindow"])
	}
	return []QuotaWindow{
		parseCodexWindow("5-hour", primary, now),
		parseCodexWindow("Weekly", secondary, now),
	}
}

func parseCodexWindow(label string, raw any, now time.Time) QuotaWindow {
	window := QuotaWindow{Label: label}
	value := asMap(raw)
	if value == nil {
		return window
	}
	if used, ok := numberValue(firstValue(value["used_percent"], value["usedPercent"])); ok {
		remaining := clamp(100-used, 0, 100)
		window.Remaining = &remaining
	} else if remainingCount, ok := numberValue(firstValue(value["remaining_count"], value["remainingCount"])); ok {
		if totalCount, ok := numberValue(firstValue(value["total_count"], value["totalCount"])); ok && totalCount > 0 {
			remaining := clamp(remainingCount/totalCount*100, 0, 100)
			window.Remaining = &remaining
		}
	}
	window.ResetAt = parseResetTime(value, now)
	return window
}

func parseClaudeWindow(label string, raw any) QuotaWindow {
	window := QuotaWindow{Label: label}
	value := asMap(raw)
	if value == nil {
		return window
	}
	if utilization, ok := numberValue(value["utilization"]); ok {
		remaining := clamp(100-utilization, 0, 100)
		window.Remaining = &remaining
	}
	window.ResetAt = parseTimeValue(firstValue(value["resets_at"], value["resetsAt"]))
	return window
}

func parseManualResetCount(payload map[string]any) *int {
	if nestedPayload := asMap(payload["rate_limit_reset_credits"]); nestedPayload != nil {
		payload = nestedPayload
	}
	value, ok := numberValue(firstValue(payload["available_count"], payload["availableCount"]))
	if !ok {
		return nil
	}
	count := max(0, int(value))
	return &count
}

func parseAntigravityFamilies(payload map[string]any) []QuotaWindow {
	families := []QuotaWindow{{Label: "Claude & GPT models"}, {Label: "Gemini models"}}
	models := asMap(payload["models"])
	for modelID, raw := range models {
		family := -1
		normalized := strings.ToLower(strings.ReplaceAll(modelID, "_", "-"))
		isClaude46 := strings.HasPrefix(normalized, "claude-") && (strings.Contains(normalized, "4-6") || strings.Contains(normalized, "4.6"))
		switch {
		case isClaude46 || strings.HasPrefix(normalized, "gpt-"):
			family = 0
		case strings.HasPrefix(normalized, "gemini-3.") || strings.HasPrefix(normalized, "gemini-3-"):
			family = 1
		default:
			continue
		}
		model := asMap(raw)
		quota := asMap(firstValue(model["quotaInfo"], model["quota_info"]))
		if quota == nil {
			quota = model
		}
		remainingFraction, ok := numberValue(firstValue(quota["remainingFraction"], quota["remaining_fraction"], quota["remaining"]))
		if !ok {
			if parseTimeValue(firstValue(quota["resetTime"], quota["reset_time"])) == nil {
				continue
			}
			remainingFraction = 0
		}
		remaining := remainingFraction
		if remaining <= 1 {
			remaining *= 100
		}
		remaining = clamp(remaining, 0, 100)
		reset := parseTimeValue(firstValue(quota["resetTime"], quota["reset_time"]))
		mergeFamilyQuota(&families[family], remaining, reset)
	}
	return families
}

func mergeFamilyQuota(window *QuotaWindow, remaining float64, reset *time.Time) {
	if window.Remaining == nil || remaining < *window.Remaining {
		window.Remaining = &remaining
		window.ResetAt = reset
		return
	}
	if remaining == *window.Remaining && reset != nil && (window.ResetAt == nil || reset.Before(*window.ResetAt)) {
		window.ResetAt = reset
	}
}

func parseResetTime(value map[string]any, now time.Time) *time.Time {
	if reset := parseTimeValue(firstValue(value["reset_at"], value["resetAt"])); reset != nil {
		return reset
	}
	if seconds, ok := numberValue(firstValue(value["reset_after_seconds"], value["resetAfterSeconds"])); ok {
		reset := now.Add(time.Duration(seconds * float64(time.Second)))
		return &reset
	}
	return nil
}

func parseTimeValue(raw any) *time.Time {
	if raw == nil {
		return nil
	}
	if number, ok := numberValue(raw); ok {
		seconds := number
		if seconds > 10_000_000_000 {
			seconds /= 1000
		}
		sec, fraction := math.Modf(seconds)
		parsed := time.Unix(int64(sec), int64(fraction*float64(time.Second)))
		return &parsed
	}
	text := strings.TrimSpace(cleanString(raw))
	if text == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, text); err == nil {
			return &parsed
		}
	}
	return nil
}

func projectIDFromAssist(payload map[string]any) string {
	value := payload["cloudaicompanionProject"]
	if project := cleanString(value); project != "" {
		return project
	}
	return cleanString(nested(payload, "cloudaicompanionProject", "id"))
}

func antigravityProjectID(entry authEntry) string {
	return firstString(
		entry.raw["project_id"], entry.raw["projectId"],
		nested(entry.raw, "metadata", "project_id"), nested(entry.raw, "metadata", "projectId"),
		nested(entry.raw, "attributes", "project_id"), nested(entry.raw, "attributes", "projectId"),
	)
}

func codexAccountID(entry authEntry) string {
	for _, candidate := range []any{
		entry.raw["id_token"], nested(entry.raw, "metadata", "id_token"), nested(entry.raw, "attributes", "id_token"),
	} {
		payload := parseJWTLike(candidate)
		if payload == nil {
			continue
		}
		if id := firstString(payload["chatgpt_account_id"], payload["chatgptAccountId"]); id != "" {
			return id
		}
		if id := firstString(nested(payload, "https://api.openai.com/auth", "chatgpt_account_id")); id != "" {
			return id
		}
	}
	return ""
}

func accountDisplayName(entry authEntry) string {
	if name := firstString(
		entry.raw["email"], nested(entry.raw, "metadata", "email"), nested(entry.raw, "attributes", "email"),
	); name != "" {
		return name
	}
	for _, candidate := range []any{
		entry.raw["id_token"], nested(entry.raw, "metadata", "id_token"), nested(entry.raw, "attributes", "id_token"),
	} {
		payload := parseJWTLike(candidate)
		if email := firstString(payload["email"], nested(payload, "https://api.openai.com/profile", "email")); email != "" {
			return email
		}
	}
	if name := firstString(entry.raw["account"], entry.raw["name"], entry.raw["label"], entry.raw["auth_index"], entry.raw["authIndex"]); name != "" {
		return name
	}
	return "unknown"
}

func parseJWTLike(raw any) map[string]any {
	if object := asMap(raw); object != nil {
		return object
	}
	text := strings.TrimSpace(cleanString(raw))
	if text == "" {
		return nil
	}
	var object map[string]any
	if json.Unmarshal([]byte(text), &object) == nil {
		return object
	}
	parts := strings.Split(text, ".")
	if len(parts) < 2 {
		return nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(decoded, &object) != nil {
		return nil
	}
	return object
}

func decodeUpstreamBody(raw json.RawMessage) (map[string]any, string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, "", errors.New("empty upstream response")
	}
	bodyText := string(raw)
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &bodyText); err != nil {
			return nil, "", fmt.Errorf("decode upstream body: %w", err)
		}
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(bodyText), &body); err != nil {
		return nil, bodyText, fmt.Errorf("decode upstream JSON: %w", err)
	}
	return body, bodyText, nil
}

func compactError(raw []byte) string {
	text := strings.Join(strings.Fields(string(raw)), " ")
	if text == "" {
		return "empty response"
	}
	if len(text) > 180 {
		return text[:177] + "..."
	}
	return text
}

func entryString(entry authEntry, keys ...string) string {
	values := make([]any, 0, len(keys))
	for _, key := range keys {
		values = append(values, entry.raw[key])
	}
	return firstString(values...)
}

func firstString(values ...any) string {
	for _, value := range values {
		if text := cleanString(value); text != "" {
			return text
		}
	}
	return ""
}

func cleanString(value any) string {
	switch value := value.(type) {
	case string:
		return strings.TrimSpace(value)
	case json.Number:
		return value.String()
	case float64:
		if value == math.Trunc(value) {
			return strconv.FormatInt(int64(value), 10)
		}
		return strconv.FormatFloat(value, 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	default:
		return ""
	}
}

func normalizeIdentifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	return value
}

func firstValue(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func nested(root map[string]any, path ...string) any {
	var current any = root
	for _, part := range path {
		object := asMap(current)
		if object == nil {
			return nil
		}
		current = object[part]
	}
	return current
}

func asMap(value any) map[string]any {
	object, _ := value.(map[string]any)
	return object
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

func numberValue(value any) (float64, bool) {
	switch value := value.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case json.Number:
		parsed, err := value.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func clamp(value, minimum, maximum float64) float64 {
	return math.Min(maximum, math.Max(minimum, value))
}
