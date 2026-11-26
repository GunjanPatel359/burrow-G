package opencode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
)

var _ session.Transcriber = Source{}

// lockDir removes write access from the database's folder, so a passing
// read proves the reader never writes (no journal, no WAL file).
func lockDir(t *testing.T, path string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
}

func TestTranscript(t *testing.T) {
	v2 := `CREATE TABLE session_v2 ` + columns + `;
CREATE TABLE session_message (id TEXT, session_id TEXT, type TEXT, seq INTEGER, time_created INTEGER, data TEXT);
INSERT INTO session_message VALUES
 ('m1','s1','system',1,1,'{"text":"SYSTEM PROMPT"}'),
 ('m2','s1','user',2,2,'{"text":"fix the auth bug","time":{"created":1794402177000}}'),
 ('m3','s1','assistant',3,3,'{"content":[{"type":"reasoning","text":"HIDDEN"},{"type":"text","text":"Looking."},{"type":"tool","name":"read","state":{"input":{"filePath":"auth.go"},"content":"TOOL OUTPUT"}}],"time":{"created":1794402178000}}'),
 ('m4','s1','synthetic',4,4,'{"text":"SYNTHETIC"}'),
 ('m5','s1','assistant',5,5,'not json'),
 ('m6','s1','assistant',6,6,'{"content":[{"type":"text","text":"Fixed."}]}'),
 ('m7','s2','user',1,1,'{"text":"OTHER SESSION"}');`

	v1 := `CREATE TABLE session ` + columns + `;
CREATE TABLE message (id TEXT, session_id TEXT, time_created INTEGER, data TEXT);
CREATE TABLE part (id TEXT, message_id TEXT, session_id TEXT, time_created INTEGER, data TEXT);
INSERT INTO message VALUES
 ('a','s1',1,'{"role":"user","time":{"created":1794402177000}}'),
 ('b','s1',2,'{"role":"assistant","time":{"created":1794402178000}}'),
 ('c','s1',3,'{"role":"assistant"}'),
 ('d','s2',1,'{"role":"user"}');
INSERT INTO part VALUES
 ('p1','a','s1',1,'{"type":"text","text":"fix the auth bug"}'),
 ('p2','a','s1',2,'{"type":"text","text":"SYNTHETIC","synthetic":true}'),
 ('p3','b','s1',3,'{"type":"reasoning","text":"HIDDEN"}'),
 ('p4','b','s1',4,'{"type":"text","text":"Looking."}'),
 ('p5','b','s1',5,'{"type":"tool","tool":"read","state":{"input":{"filePath":"auth.go"},"output":"TOOL OUTPUT"}}'),
 ('p6','b','s1',6,'broken'),
 ('p7','c','s1',7,'{"type":"text","text":"Fixed."}'),
 ('p8','d','s2',1,'{"type":"text","text":"OTHER SESSION"}');`

	for name, script := range map[string]string{"v2": v2, "v1": v1} {
		t.Run(name, func(t *testing.T) {
			path := newDB(t, script)
			lockDir(t, path)

			got, err := Source{DBPath: path}.Transcript(session.Session{Agent: "opencode", ID: "s1"})
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Messages) != 2 || got.Skipped != 1 {
				t.Fatalf("got %d messages, %d skipped: %+v", len(got.Messages), got.Skipped, got.Messages)
			}
			user, reply := got.Messages[0], got.Messages[1]
			if user.Role != "user" || user.Text != "fix the auth bug" || !user.Time.Equal(time.UnixMilli(1794402177000)) {
				t.Errorf("user = %+v", user)
			}
			if reply.Text != "Looking.\n\nFixed." || strings.Join(reply.Tools, "|") != "read auth.go" {
				t.Errorf("reply = %+v", reply)
			}
		})
	}
}

func TestStartCommand(t *testing.T) {
	if got := (Source{}).StartCommand("/tmp", "hi"); strings.Join(got, " ") != "opencode --prompt hi" {
		t.Errorf("got %q", got)
	}
}
