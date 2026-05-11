package updater

import (
	"context"

	"github.com/creativeprojects/go-selfupdate"
)

// swapper abstracts the binary swap behind an interface so callers can
// substitute alternative implementations in tests (default: go-selfupdate).
type swapper interface {
	Swap(ctx context.Context, assetURL, assetFileName, realExe string) error
}

// goSelfupdateSwapper is the production swapper that delegates to
// go-selfupdate's UpdateTo for the actual atomic binary swap.
type goSelfupdateSwapper struct{}

// Swap performs the atomic binary swap by downloading assetURL and replacing
// realExe with it. SECURITY: HTTPS only — assetURL originates from
// go-selfupdate's GitHub source which uses HTTPS exclusively (see fetch.go).
func (g *goSelfupdateSwapper) Swap(ctx context.Context, assetURL, assetFileName, realExe string) error {
	return selfupdate.UpdateTo(ctx, assetURL, assetFileName, realExe)
}
