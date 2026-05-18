package statetest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guneet-xyz/easyrice/internal/state"
)

type Mode int

const (
	ModeTruncated Mode = iota
	ModeInvalidJSON
	ModeNotAMap
	ModeMissingFields
	ModePartialWrite
	ModeNullJSON
)

func Corrupt(t *testing.T, statePath string, mode Mode) {
	t.Helper()

	var content []byte

	switch mode {
	case ModeTruncated:
		validState := state.State{
			"test": state.PackageState{
				Profile: "default",
				InstalledLinks: []state.InstalledLink{
					{
						Source: "/repo/test/file",
						Target: "/home/user/.config/test",
					},
				},
				InstalledAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		}
		fullData, err := json.MarshalIndent(validState, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal valid state: %v", err)
		}
		content = fullData[:5]

	case ModeInvalidJSON:
		content = []byte("{{{not json")

	case ModeNotAMap:
		content = []byte(`["this", "is", "an", "array"]`)

	case ModeMissingFields:
		content = []byte(`{
  "nvim": {
    "profile": "",
    "installed_links": [],
    "installed_at": ""
  },
  "ghostty": {
    "installed_links": null
  }
}`)

	case ModePartialWrite:
		validState := state.State{
			"test": state.PackageState{
				Profile: "default",
				InstalledLinks: []state.InstalledLink{
					{
						Source: "/repo/test/file",
						Target: "/home/user/.config/test",
					},
				},
				InstalledAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		}
		fullData, err := json.MarshalIndent(validState, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal valid state: %v", err)
		}
		fullStr := string(fullData)
		searchStr := `"installed_links": [`
		idx := -1
		for i := 0; i < len(fullStr)-len(searchStr); i++ {
			if fullStr[i:i+len(searchStr)] == searchStr {
				idx = i
				break
			}
		}
		if idx == -1 {
			t.Fatalf("could not find 'installed_links': [ in marshaled state")
		}
		truncatePos := idx + len(searchStr) + 1
		content = []byte(fullStr[:truncatePos])

	case ModeNullJSON:
		content = []byte("null")

	default:
		t.Fatalf("unknown corruption mode: %v", mode)
	}

	dir := filepath.Dir(statePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}

	if err := os.WriteFile(statePath, content, 0644); err != nil {
		t.Fatalf("failed to write corrupted state file: %v", err)
	}
}
