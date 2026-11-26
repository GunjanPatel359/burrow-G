package opencode

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/GunjanPatel359/burrow-G/internal/session"
	"github.com/GunjanPatel359/burrow-G/internal/transcript"
)

type v2Data struct {
	Text    string `json:"text"`
	Content []struct {
		Type  string `json:"type"`
		Text  string `json:"text"`
		Name  string `json:"name"`
		State struct {
			Input json.RawMessage `json:"input"`
		} `json:"state"`
	} `json:"content"`
	Time struct {
		Created int64 `json:"created"`
	} `json:"time"`
}

type v1Part struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	Synthetic bool   `json:"synthetic"`
	Tool      string `json:"tool"`
	State     struct {
		Input json.RawMessage `json:"input"`
	} `json:"state"`
}

func (o Source) Transcript(s session.Session) (session.Transcript, error) {
	t := session.Transcript{Session: s}
	db, err := sql.Open("sqlite", "file:"+o.DBPath+"?mode=ro")
	if err != nil {
		return t, err
	}
	defer db.Close()

	// Newer OpenCode keeps messages in session_message (with session_v2),
	// older versions in message plus part.
	var table string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name IN ('session_v2', 'message') ORDER BY name = 'session_v2' DESC LIMIT 1`).Scan(&table)
	if errors.Is(err, sql.ErrNoRows) {
		return t, errors.New("unsupported database schema: no message tables found")
	}
	if err != nil {
		return t, err
	}
	if table == "session_v2" {
		return t, readV2(db, &t)
	}
	return t, readV1(db, &t)
}

func readV2(db *sql.DB, t *session.Transcript) error {
	rows, err := db.Query(`SELECT type, data FROM session_message WHERE session_id = ? ORDER BY seq, time_created`, t.Session.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var kind, raw string
		if err := rows.Scan(&kind, &raw); err != nil {
			return err
		}
		if kind != "user" && kind != "assistant" {
			continue // system, synthetic, idle and switch markers
		}
		var d v2Data
		if json.Unmarshal([]byte(raw), &d) != nil {
			t.Skipped++
			continue
		}
		m := session.Message{Role: kind, Time: millis(d.Time.Created), Text: d.Text}
		for _, c := range d.Content {
			switch c.Type {
			case "text":
				m.Text = join(m.Text, c.Text)
			case "tool":
				m.Tools = append(m.Tools, transcript.ToolLine(c.Name, c.State.Input))
			}
		}
		t.Add(m)
	}
	return rows.Err()
}

func readV1(db *sql.DB, t *session.Transcript) error {
	rows, err := db.Query(`SELECT m.id, m.data, p.data FROM message m LEFT JOIN part p ON p.message_id = m.id
		WHERE m.session_id = ? ORDER BY m.time_created, m.id, p.time_created, p.id`, t.Session.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var cur session.Message
	var curID string
	flush := func() {
		if curID != "" {
			t.Add(cur)
		}
	}
	for rows.Next() {
		var id, msgRaw string
		var partRaw sql.NullString
		if err := rows.Scan(&id, &msgRaw, &partRaw); err != nil {
			return err
		}
		if id != curID {
			flush()
			curID, cur = id, session.Message{}
			var md struct {
				Role string `json:"role"`
				Time struct {
					Created int64 `json:"created"`
				} `json:"time"`
			}
			if json.Unmarshal([]byte(msgRaw), &md) != nil {
				t.Skipped++
				curID = ""
				continue
			}
			cur.Role, cur.Time = md.Role, millis(md.Time.Created)
		}
		if curID == "" || !partRaw.Valid || (cur.Role != "user" && cur.Role != "assistant") {
			continue
		}
		var p v1Part
		if json.Unmarshal([]byte(partRaw.String), &p) != nil {
			t.Skipped++
			continue
		}
		switch {
		case p.Type == "text" && !p.Synthetic:
			cur.Text = join(cur.Text, p.Text)
		case p.Type == "tool":
			cur.Tools = append(cur.Tools, transcript.ToolLine(p.Tool, p.State.Input))
		}
	}
	flush()
	return rows.Err()
}

func (Source) StartCommand(dir, prompt string) []string {
	return []string{"opencode", "--prompt", prompt}
}

func join(a, b string) string {
	if a == "" || b == "" {
		return a + b
	}
	return a + "\n\n" + b
}

func millis(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}
