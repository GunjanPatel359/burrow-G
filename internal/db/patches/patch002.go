package patches

import (
	"context"
)

type patch002 struct{}

var _ Patch = (*patch002)(nil)

func (patch002) Version() string {
	return "0.0.2"
}

func (patch002) Invoke(ctx context.Context, pct PatchContext) error {
	// 0.0.1 stored no version and nothing that can't be rebuilt, so start clean.
	_, err := pct.Tx.ExecContext(ctx, `DROP TABLE IF EXISTS sessions; DROP TABLE IF EXISTS bookmarks`)
	return err
}
