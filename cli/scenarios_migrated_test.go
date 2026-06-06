package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/testhelpers/scenario"
)

// runScenarioFromTestdata wires up the sandbox the way TestScenario_InstallProfileHappy
// does (T11 pilot): isolated HOME, fresh repo dir, fresh state file, copy of the
// scenario tree into a temp dir, placeholder expansion in steps.yaml, then
// scenario.Run.
func runScenarioFromTestdata(t *testing.T, scenarioName string) {
	t.Helper()
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", scenarioName))
	require.NoError(t, err)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AppData", filepath.Join(homeDir, "AppData"))

	repoRoot := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))
	copyTree(t, filepath.Join(srcDir, "repo"), repoRoot)

	stateFile := filepath.Join(t.TempDir(), "state.json")

	scenarioDir := t.TempDir()
	copyTree(t, srcDir, scenarioDir)

	stepsPath := filepath.Join(scenarioDir, "steps.yaml")
	raw, err := os.ReadFile(stepsPath)
	require.NoError(t, err)
	rendered := strings.NewReplacer(
		"__HOME__", homeDir,
		"__REPO__", repoRoot,
		"__STATE__", stateFile,
	).Replace(string(raw))
	require.NoError(t, os.WriteFile(stepsPath, []byte(rendered), 0o644))

	scenario.Run(t, scenarioDir, newScenarioConfig())
}

// Inventory of cli/cli_e2e_test.go and cli/uninstall_e2e_test.go (Task 15).
//
// Classification per the migration rubric:
//   MIGRATE: 3+ fixture writes OR multi-step (install+uninstall) OR complex assertions
//   INLINE : 1-2 fixture writes, single-step, already isolated -- leave as-is
//   SKIP   : uses withMockRunner / belongs to T18 deps work, or specialized git lifecycle
//
// | Test                                                  | Class   | Notes                              |
// |-------------------------------------------------------|---------|-------------------------------------|
// | TestE2E_InstallWithDeps                               | SKIP    | withMockRunner -> T18              |
// | TestE2E_InstallWithDeps_Linux                         | SKIP    | withMockRunner -> T18              |
// | TestE2E_VersionMismatchAbort                          | SKIP    | withMockRunner -> T18              |
// | TestE2E_ReservedSelfDepError                          | INLINE  | single-step error                  |
// | TestE2E_UninstallClearsState                          | SKIP    | InstalledDependencies (T18)        |
// | TestE2E_SkipDeps                                      | SKIP    | withMockRunner -> T18              |
// | TestE2E_FullManagedRiceFlow                           | INLINE  | git submodule + transcript: too specialized |
// | TestE2E_Uninstall_HappyPath                           | MIGRATE | uninstall_happy                    |
// | TestE2E_Uninstall_ManuallyDeletedSymlink_Skipped      | MIGRATE | uninstall_manually_deleted         |
// | TestE2E_Uninstall_SymlinkReplacedByRealFile           | MIGRATE | uninstall_replaced_by_file         |
// | TestE2E_Uninstall_SymlinkReplacedByDifferentSymlink   | INLINE  | inspects foreign Readlink target   |
// | TestE2E_Uninstall_SymlinkReplacedByDirectory          | MIGRATE | uninstall_replaced_by_dir          |
// | TestE2E_Uninstall_FolderMode_ReplacedByDirectory      | MIGRATE | uninstall_folder_mode_replaced     |
// | TestE2E_Uninstall_PackageNotInState_Error             | INLINE  | single-step error                  |
// | TestE2E_Uninstall_StateFileMissing_Error              | INLINE  | single-step error                  |
// | TestE2E_Uninstall_StaleStateEntry                     | INLINE  | needs pre-seeded state.json        |
// | TestE2E_Uninstall_PreservesOtherPackages              | MIGRATE | uninstall_preserves_others (3 step) |

func TestScenario_Uninstall_Happy(t *testing.T) {
	skipOnWindows(t)
	runScenarioFromTestdata(t, "uninstall_happy")
}

func TestScenario_Uninstall_ManuallyDeleted(t *testing.T) {
	skipOnWindows(t)
	runScenarioFromTestdata(t, "uninstall_manually_deleted")
}

func TestScenario_Uninstall_ReplacedByFile(t *testing.T) {
	skipOnWindows(t)
	runScenarioFromTestdata(t, "uninstall_replaced_by_file")
}

func TestScenario_Uninstall_ReplacedByDir(t *testing.T) {
	skipOnWindows(t)
	runScenarioFromTestdata(t, "uninstall_replaced_by_dir")
}

func TestScenario_Uninstall_FolderModeReplaced(t *testing.T) {
	skipOnWindows(t)
	runScenarioFromTestdata(t, "uninstall_folder_mode_replaced")
}

func TestScenario_Uninstall_PreservesOthers(t *testing.T) {
	skipOnWindows(t)
	runScenarioFromTestdata(t, "uninstall_preserves_others")
}

// Inventory of cli/install_e2e_test.go (Task 18).
//
// Classification per the migration rubric:
//   MIGRATE: 3+ fixture writes OR complex assertions OR multi-step
//   INLINE : 1-2 fixture writes, single-step, already isolated -- leave as-is
//   DONE   : already migrated in the T11 pilot (TestE2E_Install_FreshSmoke)
//
// | Test                                                  | Class   | Notes                                |
// |-------------------------------------------------------|---------|---------------------------------------|
// | TestE2E_Install_FreshSmoke                            | DONE    | T11 pilot                            |
// | TestE2E_Install_DeepNestedTarget_CreatesDirs          | MIGRATE | install_deep_nested_target           |
// | TestE2E_Install_HomeExpansion                         | MIGRATE | install_home_expansion (also snapshots repo) |
// | TestE2E_Install_OverlayLastWins                       | MIGRATE | install_overlay_last_wins            |
// | TestE2E_Install_NoArgs_ConvergesAll                   | MIGRATE | install_no_args_converges_all        |
// | TestE2E_Install_SourceWithSymlinkSkipped              | INLINE  | needs os.Symlink in seed (no mutate op) |
// | TestE2E_Install_EmptySourceDir_Success                | INLINE  | empty dir cannot be checked into testdata |
// | TestE2E_Install_UnsupportedOS_Error                   | INLINE  | error-string assertion               |
// | TestE2E_Install_PackageNotDeclared_Error              | INLINE  | error-string assertion               |
// | TestE2E_Install_ProfileNotDeclared_Error              | INLINE  | error-string assertion               |
// | TestE2E_Install_SourceDirMissing_Error                | INLINE  | error-string assertion               |
// | TestE2E_Install_TargetEscapesHome_Refused             | INLINE  | sandbox-escape assertion outside <HOME>/<REPO> |
// | TestE2E_Install_MalformedManifest_Error               | INLINE  | error-string assertion               |
// | TestE2E_Install_StateFileMissing_CreatedFresh         | INLINE  | --state path outside <HOME> sandbox  |
// | TestE2E_Install_StateFileCorrupted_Errors             | INLINE  | byte-equality on corrupted state     |

func TestScenario_Install_DeepNestedTarget(t *testing.T) {
	skipOnWindows(t)
	runScenarioFromTestdata(t, "install_deep_nested_target")
}

func TestScenario_Install_HomeExpansion(t *testing.T) {
	skipOnWindows(t)
	runScenarioFromTestdata(t, "install_home_expansion")
}

func TestScenario_Install_OverlayLastWins(t *testing.T) {
	skipOnWindows(t)
	runScenarioFromTestdata(t, "install_overlay_last_wins")
}

func TestScenario_Install_NoArgsConvergesAll(t *testing.T) {
	skipOnWindows(t)
	runScenarioFromTestdata(t, "install_no_args_converges_all")
}
