package statetest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModeTruncated(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	Corrupt(t, statePath, ModeTruncated)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	assert.Equal(t, 5, len(data), "truncated state should be exactly 5 bytes")
	assert.True(t, len(data) == 5, "file should be truncated to 5 bytes")
}

func TestModeInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	Corrupt(t, statePath, ModeInvalidJSON)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	assert.Equal(t, "{{{not json", string(data))

	var result interface{}
	err = json.Unmarshal(data, &result)
	assert.Error(t, err, "invalid JSON should fail to unmarshal")
}

func TestModeNotAMap(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	Corrupt(t, statePath, ModeNotAMap)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	assert.Equal(t, `["this", "is", "an", "array"]`, string(data))

	var result interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	_, isMap := result.(map[string]interface{})
	assert.False(t, isMap, "result should be array, not map")
	_, isArray := result.([]interface{})
	assert.True(t, isArray, "result should be array")
}

func TestModeMissingFields(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	Corrupt(t, statePath, ModeMissingFields)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Contains(t, result, "nvim")
	assert.Contains(t, result, "ghostty")

	nvim := result["nvim"].(map[string]interface{})
	assert.Equal(t, "", nvim["profile"], "nvim profile should be empty string")
	assert.Equal(t, []interface{}{}, nvim["installed_links"], "nvim installed_links should be empty array")
}

func TestModePartialWrite(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	Corrupt(t, statePath, ModePartialWrite)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, `"installed_links": [`, "should contain installed_links opening")
	assert.NotContains(t, content, `}`, "should not contain closing brace (truncated)")

	var result interface{}
	err = json.Unmarshal(data, &result)
	assert.Error(t, err, "partial write should be invalid JSON")
}

func TestModeNullJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	Corrupt(t, statePath, ModeNullJSON)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	assert.Equal(t, "null", string(data))

	var result interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Nil(t, result, "null JSON should unmarshal to nil")
}

func TestCorruptCreatesParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "subdir", "state.json")

	Corrupt(t, statePath, ModeNullJSON)

	assert.FileExists(t, statePath)
}
