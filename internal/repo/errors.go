package repo

import (
	"errors"
	"fmt"
)

var ErrRepoNotInitialized = errors.New("easyrice repo not initialized; run: rice init <url>")

var ErrRepoDirty = errors.New("rice repo has uncommitted changes; commit or stash before this operation")

var ErrRemoteAlreadyExists = errors.New("remote rice already exists at that path")

var ErrRemoteNotFound = errors.New("remote rice not found")

var ErrRemoteInUse = errors.New("remote rice is referenced by an import; remove the import from rice.toml first")

var ErrSubmoduleNotInitialized = errors.New("submodule not initialized; run: rice remote update <name>")

func ErrPackageNotDeclared(name string) error {
	return fmt.Errorf("package %q not declared in rice.toml", name)
}
