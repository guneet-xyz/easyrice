package profile

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/repo"
)

// Resolver resolves profile specs, including recursive import resolution
// across remote rices. A Resolver tracks visited (repoRoot, remote, package,
// profile) tuples to detect import cycles.
type Resolver struct {
	RepoRoot string
	remote   string
	visited  map[string]bool
}

// NewResolver creates a Resolver scoped to the given repo root.
func NewResolver(repoRoot string) *Resolver {
	return &Resolver{RepoRoot: repoRoot, visited: make(map[string]bool)}
}

// ResolveSpecs returns the ordered list of SourceSpec entries for the given
// profile. If the profile has an Import field, the imported profile is
// resolved recursively from the remote rice and its specs (with absolute
// paths) are prepended before any local sources.
func (r *Resolver) ResolveSpecs(pkg *manifest.PackageDef, pkgName, profileName string) ([]manifest.SourceSpec, error) {
	key := r.RepoRoot + "|" + r.remote + "|" + pkgName + "|" + profileName
	if r.visited[key] {
		return nil, fmt.Errorf("import cycle detected: %s", key)
	}
	r.visited[key] = true

	prof, exists := pkg.Profiles[profileName]
	if !exists {
		available := make([]string, 0, len(pkg.Profiles))
		for name := range pkg.Profiles {
			available = append(available, name)
		}
		sort.Strings(available)
		return nil, fmt.Errorf("profile %q not defined in package %q; available: %s",
			profileName, pkgName, strings.Join(available, ", "))
	}

	var result []manifest.SourceSpec

	if prof.Import != "" {
		importedSpecs, err := r.resolveImport(prof.Import)
		if err != nil {
			return nil, fmt.Errorf("package %q profile %q import: %w", pkgName, profileName, err)
		}
		result = append(result, importedSpecs...)
	}

	result = append(result, prof.Sources...)
	return result, nil
}

// resolveImport resolves an import string and returns specs whose Path field
// has been rewritten to an absolute path under the remote rice tree.
func (r *Resolver) resolveImport(importStr string) ([]manifest.SourceSpec, error) {
	spec, err := manifest.ParseImportSpec(importStr)
	if err != nil {
		return nil, err
	}

	remoteRepoRoot := filepath.Join(repo.RemotesDir(r.RepoRoot), spec.Remote)
	remoteTomlPath := repo.RemoteTomlPath(r.RepoRoot, spec.Remote)

	remoteMf, err := manifest.LoadFile(remoteTomlPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", repo.ErrSubmoduleNotInitialized, spec.Remote)
	}

	remotePkg, ok := remoteMf.Packages[spec.Package]
	if !ok {
		return nil, fmt.Errorf("package %q not found in remote rice %q", spec.Package, spec.Remote)
	}

	// Sub-imports inside the remote are resolved against the SAME root so
	// that "remotes/<name>" always refers to a sibling under the original
	// repo's remotes/ dir. This is what makes A->B->A cycles detectable.
	remoteResolver := &Resolver{RepoRoot: r.RepoRoot, remote: spec.Remote, visited: r.visited}
	remoteSpecs, err := remoteResolver.ResolveSpecs(&remotePkg, spec.Package, spec.Profile)
	if err != nil {
		return nil, fmt.Errorf("remote %q: %w", spec.Remote, err)
	}

	packageRoot := remotePkg.Root
	if packageRoot == "" {
		packageRoot = spec.Package
	}

	absolutized := make([]manifest.SourceSpec, len(remoteSpecs))
	for i, s := range remoteSpecs {
		out := s
		if !filepath.IsAbs(s.Path) {
			out.Path = filepath.Join(remoteRepoRoot, packageRoot, s.Path)
		}
		absolutized[i] = out
	}
	return absolutized, nil
}

// ResolveSpecs is a package-level convenience wrapper that constructs a fresh
// Resolver for the given repo root and resolves the named profile.
func ResolveSpecs(repoRoot string, pkg *manifest.PackageDef, pkgName, profileName string) ([]manifest.SourceSpec, error) {
	return NewResolver(repoRoot).ResolveSpecs(pkg, pkgName, profileName)
}
