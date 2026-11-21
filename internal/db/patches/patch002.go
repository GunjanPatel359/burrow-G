package patches

import (
	"context"
)

type patch002 struct{}

var _ Patch = (*patch002)(nil)

func (patch002) Version() string {
	return "0.0.2"
}

func (p patch002) Invoke(ctx context.Context, pct PatchContext) error {
	return updateSessions(ctx, pct)
}

func updateSessions(ctx context.Context, pct PatchContext) error {
	// 0.0.1 kept bookmarks in this column; nothing reads it any more.
	_, err := pct.Tx.ExecContext(ctx, `ALTER TABLE sessions DROP COLUMN bookmarked`)
	return err
}
