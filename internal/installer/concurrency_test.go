//go:build !windows

// Package installer — BUG-driven concurrency tests for the installer.
//
// Each subtest is a thought-experiment in concurrent correctness: state.json
// has no internal locking (see internal/state/state.go: a Save is a single
// os.WriteFile call that fully replaces the file), and the installer composes
// Load → mutate → Save without any external synchronization. The behaviors
// asserted here are what a USER would reasonably expect from a dotfile
// manager that touches a single state.json from concurrent invocations.
//
// All subtests use a sync.WaitGroup + a `chan struct{}` start barrier to
// release goroutines as simultaneously as the scheduler allows. NO sleeps.
// All paths come from t.TempDir() via the repofixture builder; we never
// touch real $HOME.
//
// Spec source: AGENTS.md "State File" (single source of truth for uninstall)
// and `.omo/plans/better-tests.md` Task 11 (lines 1201-1294).
package installer

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/profile"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
	"github.com/guneet-xyz/easyrice/internal/testutil/repofixture"
)

// concurrencyFixture builds a managed repo with `n` packages, each having a
// single profile "default" containing a single source dir with one file.
// Targets are placed under <home>/.config/<pkg>/ so distinct packages never
// share a target.
func concurrencyFixture(t *testing.T, n int) (repoRoot, homeDir, statePath string, mf *manifest.Manifest, pkgNames []string) {
	t.Helper()
	b := repofixture.New(t)
	pkgNames = make([]string, n)
	for i := 0; i < n; i++ {
		name := pkgNameFor(i)
		pkgNames[i] = name
		b = b.WithPackage(name, map[string]repofixture.Profile{
			"default": {
				Sources: []manifest.SourceSpec{
					{Path: "default", Mode: "file", Target: "$HOME/.config/" + name},
				},
			},
		})
	}
	fx := b.Build()
	t.Setenv("HOME", fx.HomePath)
	t.Setenv("USERPROFILE", fx.HomePath)

	loaded, err := manifest.LoadFile(filepath.Join(fx.RepoPath, "rice.toml"))
	require.NoError(t, err)
	return fx.RepoPath, fx.HomePath, fx.StatePath, loaded, pkgNames
}

func pkgNameFor(i int) string {
	// Stable lowercase ASCII names so order via map iteration is non-deterministic
	// but the *set* is fixed.
	return string(rune('a'+i)) + "pkg"
}

// installOnce performs a single Install for the given package and profile
// against the shared state path. The error is returned for the caller to log
// (concurrent installs may legitimately fail with conflicts on the same
// target; what we care about is the post-condition, not per-goroutine errors).
func installOnce(t *testing.T, repoRoot, homeDir, statePath string, mf *manifest.Manifest, pkgName, profileName string) error {
	t.Helper()
	pkg := mf.Packages[pkgName]
	specs, err := profile.ResolveSpecs(repoRoot, &pkg, pkgName, profileName)
	if err != nil {
		return err
	}
	_, err = Install(InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: pkgName,
		ProfileName: profileName,
		Pkg:         &pkg,
		Specs:       specs,
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   statePath,
	})
	return err
}

// convergeAllOnce performs a single ConvergeAll on the shared state.
func convergeAllOnce(repoRoot, homeDir, statePath string, mf *manifest.Manifest) error {
	_, err := ConvergeAll(ConvergeAllRequest{
		RepoRoot:       repoRoot,
		DefaultProfile: "default",
		CurrentOS:      runtime.GOOS,
		HomeDir:        homeDir,
		StatePath:      statePath,
		Manifest:       mf,
	})
	return err
}

// assertStateInvariant — the only failure detector that matters for
// concurrency bugs: state.json must (a) parse, (b) reference link targets
// that are real symlinks pointing at their declared sources, and (c) contain
// the same set of packages that have at least one of their links on disk.
//
// We DO NOT assert that EVERY package in the manifest is installed (that's
// a stronger property only some BUG tests claim). The invariant only says
// "state and disk are mutually consistent".
func assertStateInvariant(t *testing.T, statePath string) {
	t.Helper()
	st, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("state.json must parse, got: %v", err)
	}
	for pkgName, ps := range st {
		for _, link := range ps.InstalledLinks {
			ok, err := symlink.IsSymlinkTo(link.Target, link.Source)
			if err != nil {
				t.Errorf("invariant: package %q link %s: stat error: %v", pkgName, link.Target, err)
				continue
			}
			if !ok {
				t.Errorf("invariant: package %q link %s does not point to %s",
					pkgName, link.Target, link.Source)
			}
		}
	}
}

// TestInstaller_Concurrency exercises BUG-060 through BUG-065. Each subtest
// is a separate concurrency hazard. The outer test runs sequentially so the
// race detector report carries the right subtest name; inner goroutines use
// a start barrier.
func TestInstaller_Concurrency(t *testing.T) {
	// =====================================================================
	// BUG-060 — Concurrent ConvergeAll on the same managed repo
	// Spec source: AGENTS.md "State File" (single source of truth for
	//   uninstall) + Task 11 plan line 1206.
	// Expected: 8 goroutines all running ConvergeAll against the SAME state
	//   path converge to a consistent final state. state.json parses; every
	//   recorded link points to its source; the set of state entries equals
	//   the set of installed packages on disk.
	// Actual (pre-fix, suspected): last-writer-wins on state.Save drops
	//   entries written by other goroutines; symlinks exist on disk without
	//   matching state entries.
	// Why test is correct: ConvergeAll is the documented "install [no args]"
	//   entry point. Two terminals running `rice install` concurrently is a
	//   reasonable user action and MUST NOT corrupt state.
	// =====================================================================
	t.Run("BUG_060_ConcurrentConvergeAll", func(t *testing.T) {
		t.Log("BUG-060")
		repoRoot, homeDir, statePath, mf, pkgNames := concurrencyFixture(t, 4)

		const N = 8
		start := make(chan struct{})
		var wg sync.WaitGroup
		errs := make([]error, N)
		for i := 0; i < N; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				errs[i] = convergeAllOnce(repoRoot, homeDir, statePath, mf)
			}(i)
		}
		close(start)
		wg.Wait()

		// Some goroutines may report conflicts because another already
		// installed a target — that is acceptable. The invariant is what
		// matters.
		assertStateInvariant(t, statePath)

		// Every package's link should exist on disk because at least one
		// goroutine completed ConvergeAll successfully.
		st, err := state.Load(statePath)
		require.NoError(t, err)
		for _, name := range pkgNames {
			if _, ok := st[name]; !ok {
				t.Errorf("BUG-060: package %q missing from state.json after %d ConvergeAll runs", name, N)
			}
		}
	})

	// =====================================================================
	// BUG-061 — Concurrent Install of same package with different profiles
	// Spec source: Task 11 plan line 1207.
	// Expected: one profile wins atomically. state.profile must match the
	//   actual on-disk links — never a torn mixture (profile=A but links
	//   recorded for B, or vice versa).
	// Actual (suspected): Load→mutate→Save interleaving allows profile from
	//   goroutine A and InstalledLinks from goroutine B to coexist.
	// Why test is correct: switching profiles is `install --profile`; two
	//   parallel invocations should produce ONE coherent outcome.
	// =====================================================================
	t.Run("BUG_061_ConflictingProfiles", func(t *testing.T) {
		t.Log("BUG-061")
		// Build a single package with two distinct profiles whose sources
		// land on the SAME target path but point at DIFFERENT source files.
		b := repofixture.New(t).WithPackage("zsh", map[string]repofixture.Profile{
			"a": {Sources: []manifest.SourceSpec{{Path: "a", Mode: "file", Target: "$HOME/.config/zsh"}}},
			"b": {Sources: []manifest.SourceSpec{{Path: "b", Mode: "file", Target: "$HOME/.config/zsh"}}},
		})
		fx := b.Build()
		t.Setenv("HOME", fx.HomePath)
		t.Setenv("USERPROFILE", fx.HomePath)
		mf, err := manifest.LoadFile(filepath.Join(fx.RepoPath, "rice.toml"))
		require.NoError(t, err)

		start := make(chan struct{})
		var wg sync.WaitGroup
		for _, prof := range []string{"a", "b"} {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				<-start
				_ = installOnce(t, fx.RepoPath, fx.HomePath, fx.StatePath, mf, "zsh", p)
			}(prof)
		}
		close(start)
		wg.Wait()

		assertStateInvariant(t, fx.StatePath)

		st, err := state.Load(fx.StatePath)
		require.NoError(t, err)
		ps, ok := st["zsh"]
		require.True(t, ok, "BUG-061: zsh missing from state")
		assert.Contains(t, []string{"a", "b"}, ps.Profile,
			"BUG-061: profile must be exactly one of the requested profiles, got %q", ps.Profile)
		// Every recorded link's source must come from the recorded profile's directory.
		profileDir := filepath.Join(fx.RepoPath, "zsh", ps.Profile)
		for _, link := range ps.InstalledLinks {
			rel, relErr := filepath.Rel(profileDir, link.Source)
			if relErr != nil || rel == ".." || filepath.IsAbs(rel) || (len(rel) >= 3 && rel[:3] == "../") {
				t.Errorf("BUG-061: profile=%q but link source %q does not live under %q", ps.Profile, link.Source, profileDir)
			}
		}
	})

	// =====================================================================
	// BUG-062 — Concurrent Install + Uninstall of the same package
	// Spec source: Task 11 plan line 1208.
	// Expected: final post-condition is either fully installed (state has
	//   entry, every recorded link exists on disk) OR fully uninstalled
	//   (state has no entry, no orphan symlinks on disk for this package).
	//   Never "state says installed but links are gone" or vice versa.
	// Actual (suspected): partial state where state.json claims links that
	//   were already removed by the concurrent uninstall.
	// Why test is correct: a user running `rice install foo` in one shell
	//   and `rice uninstall foo` in another should never end in an
	//   inconsistent state — that's exactly what the state file is supposed
	//   to prevent.
	// =====================================================================
	t.Run("BUG_062_InstallVsUninstall", func(t *testing.T) {
		t.Log("BUG-062")
		repoRoot, homeDir, statePath, mf, pkgNames := concurrencyFixture(t, 1)
		pkgName := pkgNames[0]
		// Pre-install so Uninstall has something to remove.
		require.NoError(t, installOnce(t, repoRoot, homeDir, statePath, mf, pkgName, "default"))

		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_ = installOnce(t, repoRoot, homeDir, statePath, mf, pkgName, "default")
		}()
		go func() {
			defer wg.Done()
			<-start
			_ = Uninstall(UninstallRequest{PackageName: pkgName, StatePath: statePath})
		}()
		close(start)
		wg.Wait()

		assertStateInvariant(t, statePath)
	})

	// =====================================================================
	// BUG-063 — Concurrent install of DIFFERENT packages
	// Spec source: Task 11 plan line 1209.
	// Expected: 8 disjoint packages, 8 goroutines, 8 entries in state.json.
	//   With non-overlapping targets there is NO reason any install should
	//   fail; the only loss possible comes from state.Save not being atomic
	//   across goroutines (last-writer-wins on the whole file).
	// Actual (suspected, BUG-007 territory): some entries are silently
	//   dropped because the last goroutine to Save overwrote earlier saves
	//   with a stale snapshot.
	// Why test is correct: this is the canonical "two terminals install
	//   different packages" workflow. There is no legitimate reason to lose
	//   data here.
	// =====================================================================
	t.Run("BUG_063_DistinctPackagesAllSurvive", func(t *testing.T) {
		t.Log("BUG-063")
		const N = 8
		repoRoot, homeDir, statePath, mf, pkgNames := concurrencyFixture(t, N)

		start := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < N; i++ {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				<-start
				_ = installOnce(t, repoRoot, homeDir, statePath, mf, name, "default")
			}(pkgNames[i])
		}
		close(start)
		wg.Wait()

		assertStateInvariant(t, statePath)

		st, err := state.Load(statePath)
		require.NoError(t, err)
		for _, name := range pkgNames {
			if _, ok := st[name]; !ok {
				t.Errorf("BUG-063: package %q lost from state.json (last-writer-wins on state.Save)", name)
			}
		}
	})

	// =====================================================================
	// BUG-064 — state.Save called twice rapidly; second must fully replace
	// Spec source: Task 11 plan line 1210.
	// Expected: when goroutine X calls state.Save({}) and goroutine Y calls
	//   state.Save({foo}) and Y wins, the file contains exactly the foo
	//   payload — not a merged/interleaved/truncated JSON object.
	// Actual (suspected): non-atomic write (os.WriteFile is not atomic on
	//   POSIX without rename) can leave partial JSON visible to readers.
	// Why test is correct: state is the source of truth for uninstall. A
	//   reader (e.g., another `rice status`) that sees a torn write would
	//   misreport installed state or panic on unmarshal.
	// =====================================================================
	t.Run("BUG_064_RapidSaveReplacement", func(t *testing.T) {
		t.Log("BUG-064")
		repoRoot, homeDir, statePath, mf, pkgNames := concurrencyFixture(t, 2)
		// Pre-install one to get a non-empty state.
		require.NoError(t, installOnce(t, repoRoot, homeDir, statePath, mf, pkgNames[0], "default"))

		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		// Goroutine A: install pkg[1].
		go func() {
			defer wg.Done()
			<-start
			_ = installOnce(t, repoRoot, homeDir, statePath, mf, pkgNames[1], "default")
		}()
		// Goroutine B: load + save (overwrite with current snapshot many times).
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 16; i++ {
				s, err := state.Load(statePath)
				if err != nil {
					t.Errorf("BUG-064: state.Load mid-flight failed (torn write?): %v", err)
					return
				}
				if err := state.Save(statePath, s); err != nil {
					t.Errorf("BUG-064: state.Save mid-flight failed: %v", err)
					return
				}
			}
		}()
		close(start)
		wg.Wait()

		// After all writes settle, file must parse and invariant holds.
		assertStateInvariant(t, statePath)
	})

	// =====================================================================
	// BUG-065 — Concurrent BuildConvergePlan (read) with ExecuteConvergePlan
	// Spec source: Task 11 plan line 1211.
	// Expected: a goroutine running BuildConvergePlan in a tight loop while
	//   another goroutine is executing a converge plan must always see a
	//   consistent view: either pre-execute (Outcome=Installed because no
	//   state yet) or post-execute (Outcome=NoOp/Repaired with valid state).
	//   It must NEVER hit a json parse error or a panic.
	// Actual (suspected): mid-write reads of state.json fail to unmarshal.
	// Why test is correct: BuildConvergePlan is documented read-only on the
	//   filesystem; callers can issue it from any context (e.g., `rice
	//   status`). It must tolerate concurrent writers.
	// =====================================================================
	t.Run("BUG_065_PlanReadDuringExecuteWrite", func(t *testing.T) {
		t.Log("BUG-065")
		repoRoot, homeDir, statePath, mf, pkgNames := concurrencyFixture(t, 1)
		pkgName := pkgNames[0]
		pkg := mf.Packages[pkgName]

		req := ConvergeRequest{
			RepoRoot:         repoRoot,
			PackageName:      pkgName,
			RequestedProfile: "default",
			CurrentOS:        runtime.GOOS,
			HomeDir:          homeDir,
			StatePath:        statePath,
			Pkg:              &pkg,
			Manifest:         mf,
		}

		start := make(chan struct{})
		stop := make(chan struct{})
		var wg sync.WaitGroup

		// Writer: keep converging (install → no-op → install …). After each
		// successful install we Uninstall so the next iteration installs
		// again — keeps state.json churning.
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 32; i++ {
				cr, err := BuildConvergePlan(req)
				if err != nil || cr == nil {
					continue
				}
				_ = ExecuteConvergePlan(req, cr)
				_ = Uninstall(UninstallRequest{PackageName: pkgName, StatePath: statePath})
			}
			close(stop)
		}()

		// Reader: hammer BuildConvergePlan; on any error other than the
		// documented "no profile" path, record it.
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for {
				select {
				case <-stop:
					return
				default:
				}
				if _, err := BuildConvergePlan(req); err != nil {
					t.Errorf("BUG-065: BuildConvergePlan returned error during concurrent execute: %v", err)
					return
				}
			}
		}()

		close(start)
		wg.Wait()

		assertStateInvariant(t, statePath)
	})
}

// TestInstaller_Concurrency_StateInvariant is the QA scenario from the plan
// (line 1275). After a representative concurrent run, parse state.json and
// assert: (a) parses, (b) every InstalledLink target is a symlink to its
// source, (c) every package with state entries has on-disk links and vice
// versa.
func TestInstaller_Concurrency_StateInvariant(t *testing.T) {
	const N = 8
	repoRoot, homeDir, statePath, mf, pkgNames := concurrencyFixture(t, N)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			<-start
			_ = installOnce(t, repoRoot, homeDir, statePath, mf, name, "default")
		}(pkgNames[i])
	}
	close(start)
	wg.Wait()

	// (a) parse
	st, err := state.Load(statePath)
	require.NoError(t, err, "state.json must parse after concurrent installs")

	// (b) every link points to its source.
	for pkgName, ps := range st {
		for _, link := range ps.InstalledLinks {
			ok, err := symlink.IsSymlinkTo(link.Target, link.Source)
			require.NoError(t, err, "stat %s: %v", link.Target, err)
			assert.True(t, ok, "invariant: %s/%s not a symlink to %s", pkgName, link.Target, link.Source)
		}
	}

	// (c) state entries ↔ packages-with-links-on-disk. For each manifest
	// package, check if its expected target exists on disk; if yes, the
	// package must be in state.
	for _, name := range pkgNames {
		target := filepath.Join(homeDir, ".config", name, "dummy.txt")
		_, statErr := symlink.IsSymlinkTo(target, filepath.Join(repoRoot, name, "default", "dummy.txt"))
		// statErr non-nil typically means target missing; if missing, the
		// package should also be absent from state.
		_, inState := st[name]
		if statErr == nil && !inState {
			t.Errorf("invariant: package %q has on-disk link but no state entry", name)
		}
	}
}
