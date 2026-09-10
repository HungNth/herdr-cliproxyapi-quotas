package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseVersionTriplet(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in    string
		major int
		minor int
		patch int
		ok    bool
	}{
		{"vV7.2.154", 0, 0, 0, false},
		{"vv7.2.154", 0, 0, 0, false},
		{"7.2.154", 7, 2, 154, true},
		{"v7.2.154", 7, 2, 154, true},
		{"V7.2.154", 7, 2, 154, true},
		{"7.2.10", 7, 2, 10, true},
		{"7.2", 0, 0, 0, false},
		{"7.2.154-beta.1", 0, 0, 0, false},
		{"", 0, 0, 0, false},
		{"abc", 0, 0, 0, false},
		{"7.x.154", 0, 0, 0, false},
	}
	for _, testCase := range cases {
		major, minor, patch, ok := parseVersionTriplet(testCase.in)
		if ok != testCase.ok || major != testCase.major || minor != testCase.minor || patch != testCase.patch {
			t.Fatalf("parseVersionTriplet(%q) = %d,%d,%d,%v", testCase.in, major, minor, patch, ok)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		a    string
		b    string
		want int
	}{
		{"7.2.154", "7.2.154", 0},
		{"v7.2.154", "7.2.154", 0},
		{"7.2.9", "7.2.10", -1},
		{"7.2.10", "7.2.9", 1},
		{"7.3.0", "7.2.99", 1},
		{"6.9.9", "7.0.0", -1},
	}
	for _, testCase := range cases {
		got, ok := compareVersions(testCase.a, testCase.b)
		if !ok || got != testCase.want {
			t.Fatalf("compareVersions(%q, %q) = %d,%v, want %d", testCase.a, testCase.b, got, ok, testCase.want)
		}
	}
}

func TestCompareVersionsMalformed(t *testing.T) {
	t.Parallel()

	if _, ok := compareVersions("7.2", "7.2.1"); ok {
		t.Fatal("malformed current must not compare")
	}
	if _, ok := compareVersions("7.2.1", "next"); ok {
		t.Fatal("malformed latest must not compare")
	}
}

func TestMissingVersionHeaderLeavesCurrentUnknown(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == authFilesPath {
			writeJSON(t, w, map[string]any{"files": []any{}})
			return
		}
		writeJSON(t, w, map[string]any{"latest-version": "v7.2.155"})
	}))
	defer server.Close()

	snapshot, err := newClient(Config{BaseURL: server.URL, ManagementKey: "secret"}).FetchSnapshot(context.Background())
	if err != nil {
		t.Fatalf("FetchSnapshot() error = %v", err)
	}
	if snapshot.CurrentVersion != "" {
		t.Fatalf("CurrentVersion = %q, want empty", snapshot.CurrentVersion)
	}
}

func TestFetchLatestVersionRejectsBadAuthAndBadPayload(t *testing.T) {
	t.Parallel()

	t.Run("malformed payload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer secret" {
				t.Errorf("missing management authorization")
			}
			w.Write([]byte("not json"))
		}))
		defer server.Close()

		_, err := newClient(Config{BaseURL: server.URL, ManagementKey: "secret"}).fetchLatestVersion(context.Background())
		if err == nil {
			t.Fatal("malformed payload must error")
		}
	})

	t.Run("http failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "upstream broken", http.StatusBadGateway)
		}))
		defer server.Close()

		_, err := newClient(Config{BaseURL: server.URL, ManagementKey: "secret"}).fetchLatestVersion(context.Background())
		if err == nil || !strings.Contains(err.Error(), "502") {
			t.Fatalf("expected 502 error, got %v", err)
		}
	})
}
