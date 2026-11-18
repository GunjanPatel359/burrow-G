package patches

import "database/sql"

// patch0_0_2: bookmarks moved to their own table.
var patch0_0_2 = Patch{
	Version: "0.0.2",
	Apply: func(tx *sql.Tx) error {
		// 0.0.1 kept bookmarks in this column; nothing reads it any more.
		if err := dropColumn(tx, "sessions", "bookmarked"); err != nil {
			return err
		}
		// left by a test build of the sudo work.
		return dropColumn(tx, "sessions", "owner")
	},
}
