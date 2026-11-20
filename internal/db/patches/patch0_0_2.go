package patches

// patch0_0_2: bookmarks moved to their own table.
type patch0_0_2 struct{}

func (patch0_0_2) Version() string { return "0.0.2" }

func (patch0_0_2) Invoke() error {
	// 0.0.1 kept bookmarks in this column; nothing reads it any more.
	return exec(`ALTER TABLE sessions DROP COLUMN bookmarked`)
}
