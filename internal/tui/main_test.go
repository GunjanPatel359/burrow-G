package tui

import (
	"os"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/tui/components/statusbar"
)

// TestMain makes done messages fade at once, so tests that run every
// command don't each wait two seconds.
func TestMain(m *testing.M) {
	statusbar.FadeAfter = time.Millisecond
	os.Exit(m.Run())
}
