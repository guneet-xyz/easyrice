//go:build !windows

package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/testhelpers/scenario"
)

func TestScenario_IdempotentConverge(t *testing.T) {
	t.Log("BUG-180")
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "idempotent_converge"))
	require.NoError(t, err)

	sb := setupScenarioSandbox(t)
	copyTree(t, filepath.Join(srcDir, "repo"), sb.RepoRoot)

	scenarioDir := renderScenario(t, srcDir, sb)
	scenario.Run(t, scenarioDir, newScenarioConfig())
}

func TestScenario_ProfileFallbackError(t *testing.T) {
	t.Log("BUG-181")
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "profile_fallback_error"))
	require.NoError(t, err)

	sb := setupScenarioSandbox(t)
	copyTree(t, filepath.Join(srcDir, "repo"), sb.RepoRoot)

	scenarioDir := renderScenario(t, srcDir, sb)
	scenario.Run(t, scenarioDir, newScenarioConfig())
}

func TestScenario_PartialInstallRecovery(t *testing.T) {
	t.Log("BUG-182")
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "partial_install_recovery"))
	require.NoError(t, err)

	sb := setupScenarioSandbox(t)
	copyTree(t, filepath.Join(srcDir, "repo"), sb.RepoRoot)

	scenarioDir := renderScenario(t, srcDir, sb)
	scenario.Run(t, scenarioDir, newScenarioConfig())
}

func TestScenario_DeepOverlay3Remotes(t *testing.T) {
	t.Log("BUG-183")
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "deep_overlay_3_remotes"))
	require.NoError(t, err)

	sb := setupScenarioSandbox(t)
	copyTree(t, filepath.Join(srcDir, "repo"), sb.RepoRoot)

	scenarioDir := renderScenario(t, srcDir, sb)
	scenario.Run(t, scenarioDir, newScenarioConfig())
}

func TestScenario_RemoteInUseBlocksRemove(t *testing.T) {
	t.Log("BUG-184")
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "remote_in_use_blocks_remove"))
	require.NoError(t, err)

	sb := setupScenarioSandbox(t)

	bareManifest := `schema_version = 1
`
	writeManagedManifestAndInit(t, sb.RepoRoot, bareManifest)
	setupRemoteSubmodule(t, sb.RepoRoot, "base", baseUpstreamFiles())

	managedManifest := `schema_version = 1

[packages.demo]
description = "demo importing base.common to keep remote in use"
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
import = "remotes/base#base.common"
`
	rewriteManagedManifest(t, sb.RepoRoot, managedManifest)

	scenarioDir := renderScenario(t, srcDir, sb)
	scenario.Run(t, scenarioDir, newScenarioConfig())
}

func TestScenario_DirtyTreeBlocksRemoteAdd(t *testing.T) {
	t.Log("BUG-185")
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "dirty_tree_blocks_remote_add"))
	require.NoError(t, err)

	sb := setupScenarioSandbox(t)

	bareManifest := `schema_version = 1
`
	writeManagedManifestAndInit(t, sb.RepoRoot, bareManifest)

	scenarioDir := renderScenario(t, srcDir, sb)
	scenario.Run(t, scenarioDir, newScenarioConfig())
}
