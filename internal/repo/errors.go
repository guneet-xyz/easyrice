package repo

import (
	"errors"
	"fmt"
)

var ErrRepoNotInitialized = errors.New("easyrice repo not initialized; run: rice init <url>")

func ErrPackageNotDeclared(name string) error {
	return fmt.Errorf("package %q not declared in rice.toml", name)
}
