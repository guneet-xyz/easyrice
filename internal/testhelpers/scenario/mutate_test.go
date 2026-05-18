package scenario

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyMutate_AllOps(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	tests := []struct {
		name string
		op   MutateOp
		prep func(t *testing.T)
		verify func(t *testing.T)
	}{
		{
			name: "remove_file",
			op: MutateOp{
				Op:   "remove",
				Path: filepath.Join(home, "test.txt"),
			},
			prep: func(t *testing.T) {
				require.NoError(t, os.WriteFile(filepath.Join(home, "test.txt"), []byte("content"), 0o644))
			},
			verify: func(t *testing.T) {
				_, err := os.Stat(filepath.Join(home, "test.txt"))
				require.True(t, os.IsNotExist(err), "file should be removed")
			},
		},
		{
			name: "remove_directory",
			op: MutateOp{
				Op:   "remove",
				Path: filepath.Join(home, "testdir"),
			},
			prep: func(t *testing.T) {
				require.NoError(t, os.MkdirAll(filepath.Join(home, "testdir", "subdir"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(home, "testdir", "subdir", "file.txt"), []byte("content"), 0o644))
			},
			verify: func(t *testing.T) {
				_, err := os.Stat(filepath.Join(home, "testdir"))
				require.True(t, os.IsNotExist(err), "directory should be removed")
			},
		},
		{
			name: "write_file_new",
			op: MutateOp{
				Op:      "write_file",
				Path:    filepath.Join(home, "subdir", "newfile.txt"),
				Content: "hello world",
			},
			prep: func(t *testing.T) {
				// No prep needed; directory will be created
			},
			verify: func(t *testing.T) {
				content, err := os.ReadFile(filepath.Join(home, "subdir", "newfile.txt"))
				require.NoError(t, err)
				require.Equal(t, "hello world", string(content))
			},
		},
		{
			name: "write_file_overwrite",
			op: MutateOp{
				Op:      "write_file",
				Path:    filepath.Join(home, "existing.txt"),
				Content: "new content",
			},
			prep: func(t *testing.T) {
				require.NoError(t, os.WriteFile(filepath.Join(home, "existing.txt"), []byte("old content"), 0o644))
			},
			verify: func(t *testing.T) {
				content, err := os.ReadFile(filepath.Join(home, "existing.txt"))
				require.NoError(t, err)
				require.Equal(t, "new content", string(content))
			},
		},
		{
			name: "replace_symlink",
			op: MutateOp{
				Op:     "replace_symlink",
				Path:   filepath.Join(home, "link"),
				Target: filepath.Join(repo, "newtarget"),
			},
			prep: func(t *testing.T) {
				require.NoError(t, os.WriteFile(filepath.Join(repo, "oldtarget"), []byte("old"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(repo, "newtarget"), []byte("new"), 0o644))
				require.NoError(t, os.Symlink(filepath.Join(repo, "oldtarget"), filepath.Join(home, "link")))
			},
			verify: func(t *testing.T) {
				target, err := os.Readlink(filepath.Join(home, "link"))
				require.NoError(t, err)
				require.Equal(t, filepath.Join(repo, "newtarget"), target)
			},
		},
		{
			name: "mkdir_default_mode",
			op: MutateOp{
				Op:   "mkdir",
				Path: filepath.Join(home, "newdir"),
				Mode: 0,
			},
			prep: func(t *testing.T) {
				// No prep needed
			},
			verify: func(t *testing.T) {
				info, err := os.Stat(filepath.Join(home, "newdir"))
				require.NoError(t, err)
				require.True(t, info.IsDir(), "should be a directory")
			},
		},
		{
			name: "mkdir_custom_mode",
			op: MutateOp{
				Op:   "mkdir",
				Path: filepath.Join(home, "customdir"),
				Mode: 0o700,
			},
			prep: func(t *testing.T) {
				// No prep needed
			},
			verify: func(t *testing.T) {
				info, err := os.Stat(filepath.Join(home, "customdir"))
				require.NoError(t, err)
				require.True(t, info.IsDir(), "should be a directory")
				require.Equal(t, os.FileMode(0o700), info.Mode().Perm())
			},
		},
		{
			name: "chmod",
			op: MutateOp{
				Op:   "chmod",
				Path: filepath.Join(home, "tochange.txt"),
				Mode: 0o600,
			},
			prep: func(t *testing.T) {
				require.NoError(t, os.WriteFile(filepath.Join(home, "tochange.txt"), []byte("content"), 0o644))
			},
			verify: func(t *testing.T) {
				info, err := os.Stat(filepath.Join(home, "tochange.txt"))
				require.NoError(t, err)
				require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.prep(t)
			err := applyMutate(home, repo, tc.op)
			require.NoError(t, err, "applyMutate should not error")
			tc.verify(t)
		})
	}
}

func TestApplyMutate_PlaceholderExpansion(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	t.Run("expand_HOME_in_path", func(t *testing.T) {
		op := MutateOp{
			Op:      "write_file",
			Path:    filepath.Join("<HOME>", "test.txt"),
			Content: "content",
		}
		err := applyMutate(home, repo, op)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(home, "test.txt"))
		require.NoError(t, err)
		require.Equal(t, "content", string(content))
	})

	t.Run("expand_REPO_in_path", func(t *testing.T) {
		op := MutateOp{
			Op:      "write_file",
			Path:    filepath.Join("<REPO>", "test.txt"),
			Content: "content",
		}
		err := applyMutate(home, repo, op)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(repo, "test.txt"))
		require.NoError(t, err)
		require.Equal(t, "content", string(content))
	})

	t.Run("expand_HOME_in_target", func(t *testing.T) {
		// Create a file to symlink to
		require.NoError(t, os.WriteFile(filepath.Join(home, "target.txt"), []byte("target"), 0o644))

		op := MutateOp{
			Op:     "replace_symlink",
			Path:   filepath.Join(repo, "link"),
			Target: filepath.Join("<HOME>", "target.txt"),
		}

		// First create a symlink to replace
		require.NoError(t, os.Symlink(filepath.Join(home, "dummy"), filepath.Join(repo, "link")))

		err := applyMutate(home, repo, op)
		require.NoError(t, err)

		target, err := os.Readlink(filepath.Join(repo, "link"))
		require.NoError(t, err)
		require.Equal(t, filepath.Join(home, "target.txt"), target)
	})
}

func TestApplyMutate_RejectExternalPath(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	op := MutateOp{
		Op:   "write_file",
		Path: "/etc/passwd",
		Content: "malicious",
	}

	err := applyMutate(home, repo, op)
	require.Error(t, err)
	require.Contains(t, err.Error(), "outside sandbox")
}

func TestApplyMutate_UnknownOp(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	op := MutateOp{
		Op:   "explode",
		Path: filepath.Join(home, "test.txt"),
	}

	err := applyMutate(home, repo, op)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInvalidYAML), "should wrap ErrInvalidYAML")
	require.Contains(t, err.Error(), "unknown mutate op")
}
