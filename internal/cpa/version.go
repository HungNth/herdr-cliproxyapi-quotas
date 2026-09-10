package cpa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const latestVersionPath = "/v0/management/latest-version"

func parseVersionTriplet(raw string) (major int, minor int, patch int, ok bool) {
	trimmed := raw
	if strings.HasPrefix(raw, "v") || strings.HasPrefix(raw, "V") {
		trimmed = raw[1:]
		if strings.HasPrefix(trimmed, "v") || strings.HasPrefix(trimmed, "V") {
			return 0, 0, 0, false
		}
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	values := make([]int, 3)
	for index, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return 0, 0, 0, false
		}
		values[index] = value
	}
	return values[0], values[1], values[2], true
}

func CompareVersions(current string, latest string) (int, bool) {
	currentMajor, currentMinor, currentPatch, currentOK := parseVersionTriplet(current)
	latestMajor, latestMinor, latestPatch, latestOK := parseVersionTriplet(latest)
	if !currentOK || !latestOK {
		return 0, false
	}
	return compareTriplets(currentMajor, currentMinor, currentPatch, latestMajor, latestMinor, latestPatch), true
}

func compareTriplets(aMajor int, aMinor int, aPatch int, bMajor int, bMinor int, bPatch int) int {
	current := []int{aMajor, aMinor, aPatch}
	latest := []int{bMajor, bMinor, bPatch}
	for index := range current {
		if current[index] != latest[index] {
			if current[index] < latest[index] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func (c *Client) FetchLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(latestVersionPath), nil)
	if err != nil {
		return "", err
	}
	c.applyManagementAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", latestVersionPath, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", latestVersionPath, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GET %s: HTTP %d: %s", latestVersionPath, resp.StatusCode, compactError(raw))
	}
	var payload struct {
		LatestVersion string `json:"latest-version"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("decode %s: %w", latestVersionPath, err)
	}
	if payload.LatestVersion == "" {
		return "", fmt.Errorf("GET %s: empty latest-version", latestVersionPath)
	}
	return payload.LatestVersion, nil
}
