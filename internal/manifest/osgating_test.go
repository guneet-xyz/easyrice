package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckOS(t *testing.T) {
	tests := []struct {
		name        string
		manifest    *Manifest
		currentOS   string
		expectError bool
		errorMsg    string
	}{
		{
			name: "linux package on linux",
			manifest: &Manifest{
				Name:        "test-pkg",
				SupportedOS: []string{"linux"},
			},
			currentOS:   "linux",
			expectError: false,
		},
		{
			name: "darwin package on darwin",
			manifest: &Manifest{
				Name:        "test-pkg",
				SupportedOS: []string{"darwin"},
			},
			currentOS:   "darwin",
			expectError: false,
		},
		{
			name: "linux-only package on windows",
			manifest: &Manifest{
				Name:        "test-pkg",
				SupportedOS: []string{"linux"},
			},
			currentOS:   "windows",
			expectError: true,
			errorMsg:    "does not support windows",
		},
		{
			name: "linux-only package on windows contains supported OS",
			manifest: &Manifest{
				Name:        "test-pkg",
				SupportedOS: []string{"linux"},
			},
			currentOS:   "windows",
			expectError: true,
			errorMsg:    "linux",
		},
		{
			name: "multi-OS package (linux+darwin) on darwin",
			manifest: &Manifest{
				Name:        "test-pkg",
				SupportedOS: []string{"linux", "darwin"},
			},
			currentOS:   "darwin",
			expectError: false,
		},
		{
			name: "multi-OS package (linux+darwin) on windows",
			manifest: &Manifest{
				Name:        "test-pkg",
				SupportedOS: []string{"linux", "darwin"},
			},
			currentOS:   "windows",
			expectError: true,
			errorMsg:    "does not support windows",
		},
		{
			name: "multi-OS package (linux+darwin) on windows contains supported list",
			manifest: &Manifest{
				Name:        "test-pkg",
				SupportedOS: []string{"linux", "darwin"},
			},
			currentOS:   "windows",
			expectError: true,
			errorMsg:    "linux, darwin",
		},
		{
			name: "empty SupportedOS",
			manifest: &Manifest{
				Name:        "test-pkg",
				SupportedOS: []string{},
			},
			currentOS:   "linux",
			expectError: true,
			errorMsg:    "does not support linux",
		},
		{
			name: "windows package on windows",
			manifest: &Manifest{
				Name:        "test-pkg",
				SupportedOS: []string{"windows"},
			},
			currentOS:   "windows",
			expectError: false,
		},
		{
			name: "all three OSes supported on linux",
			manifest: &Manifest{
				Name:        "test-pkg",
				SupportedOS: []string{"linux", "darwin", "windows"},
			},
			currentOS:   "linux",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckOS(tt.manifest, tt.currentOS)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
