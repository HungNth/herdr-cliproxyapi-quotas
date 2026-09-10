// Command fakeherdr is a test double for the Herdr CLI. It records each
// invocation's arguments to the file named by FAKE_HERDR_LOG and simulates
// pane existence using FAKE_HERDR_EXISTING_PANES (comma-separated IDs).
// Setting FAKE_HERDR_FAIL=1 makes every command exit non-zero.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	log := os.Getenv("FAKE_HERDR_LOG")
	if log != "" {
		f, err := os.OpenFile(log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err == nil {
			fmt.Fprintln(f, strings.Join(os.Args[1:], " "))
			f.Close()
		}
	}
	if os.Getenv("FAKE_HERDR_FAIL") == "1" {
		os.Exit(1)
	}
	args := os.Args[1:]
	if len(args) >= 3 && args[0] == "plugin" && args[1] == "pane" && args[2] == "open" {
		fmt.Println(`{"id":"cli:plugin","result":{"plugin_pane":{"pane":{"pane_id":"w1:p99"},"type":"ok"}}}`)
		os.Exit(0)
	}
	if len(args) >= 2 && args[0] == "pane" && args[1] == "get" {
		existing := os.Getenv("FAKE_HERDR_EXISTING_PANES")
		for _, id := range strings.Split(existing, ",") {
			if len(args) >= 3 && id == args[2] {
				fmt.Printf(`{"id":"cli:pane","result":{"pane":{"pane_id":"%s"}}}`, id)
				os.Exit(0)
			}
		}
		os.Exit(1)
	}
	os.Exit(0)
}
