package repo

import (
	"errors"
	"fmt"
)

var ErrRepoNotInitialized = errors.New("easyrice repo is not initialized; run `rice init <url>` first")

var ErrRepoDirty = errors.New("rice repo has uncommitted changes; commit or stash them before this operation")

var ErrRemoteAlreadyExists = errors.New("a remote already exists at that path")

var ErrRemoteNotFound = errors.New("remote not found")

var ErrRemoteInUse = errors.New("remote is referenced by an import; remove the import from rice.toml first")

var ErrSubmoduleNotInitialized = errors.New("remote is not initialized; run `rice remote update <name>`")

func ErrPackageNotDeclared(name string) error {
	return fmt.Errorf("package %q not declared in rice.toml", name)
}
