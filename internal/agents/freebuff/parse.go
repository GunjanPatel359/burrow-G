package freebuff

import (
	"encoding/json"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/GunjanPatel359/burrow-G/internal/helpers"
	"github.com/GunjanPatel359/burrow-G/internal/session"
)

// readChat reads the chat folder dir, under projects/<project>/chats/, the way
// List shows it. ok is false for a folder List skips: a chat opened and
// never used, a run state it can't read, or no folder that leads
// --continue back to this chat.
func readChat(dir, project string) (s session.Session, ok bool) {
	// A chat that was opened and never used has no messages file.
	messages, err := os.Stat(filepath.Join(dir, "chat-messages.json"))
	if err != nil {
		return session.Session{}, false
	}
	root, cwd, err := folders(filepath.Join(dir, "run-state.json"))
	if err != nil {
		return session.Session{}, false
	}
	// --continue looks for the chat under projects/<base name of the project
	// root> and silently opens the latest chat when it isn't there, so only
	// offer a folder that leads back to this chat.
	var folder string
	for _, d := range []string{root, cwd} {
		if d != "" && filepath.Base(d) == project {
			folder = d
			break
		}
	}
	if folder == "" {
		return session.Session{}, false
	}

	title := ownTitle(filepath.Join(dir, "chat-title.txt"))
	if title == "" {
		title = cachedTitle(filepath.Join(dir, "chat-meta.json"), messages)
	}
	if title == "" {
		title = "(untitled)"
	}
	return session.Session{
		Agent:    "freebuff",
		ID:       filepath.Base(dir),
		Title:    title,
		Dir:      folder,
		LastUsed: messages.ModTime(),
	}, true
}

// folders reads sessionState.fileContext.projectRoot and .cwd from
// run-state.json. The file is close to 1 MB but both values come first, so it
// stops reading as soon as it has them.
func folders(path string) (root, cwd string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	if err := walk(dec, "sessionState", "fileContext"); err != nil {
		return "", "", err
	}
	for dec.More() && (root == "" || cwd == "") {
		key, err := dec.Token()
		if err != nil {
			return "", "", err
		}
		switch key {
		case "projectRoot":
			err = dec.Decode(&root)
		case "cwd":
			err = dec.Decode(&cwd)
		default:
			err = dec.Decode(&json.RawMessage{})
		}
		if err != nil {
			return "", "", err
		}
	}
	return root, cwd, nil
}

// errNoKey means a key walk looked for isn't in the file.
var errNoKey = errors.New("unsupported run state: key not found")

// walk moves dec, at the start of a JSON object, inside the value stored at
// the path of keys. Each value on the way but the last must be an object;
// the last may be an object or an array.
func walk(dec *json.Decoder, keys ...string) error {
	if err := expect(dec, '{'); err != nil {
		return err
	}
	for i, key := range keys {
		if err := seek(dec, key); err != nil {
			return err
		}
		if i < len(keys)-1 {
			if err := expect(dec, '{'); err != nil {
				return err
			}
			continue
		}
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if tok != json.Delim('{') && tok != json.Delim('[') {
			return errors.New("unsupported run state: " + key + " is not an object or array")
		}
	}
	return nil
}

// seek moves dec, which is inside an object, to the value stored under key,
// skipping the values before it.
func seek(dec *json.Decoder, key string) error {
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return err
		}
		if k == key {
			return nil
		}
		if err := dec.Decode(&json.RawMessage{}); err != nil {
			return err
		}
	}
	return errNoKey
}

// expect reads the brace or bracket that starts an object or an array.
func expect(dec *json.Decoder, delim json.Delim) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if tok != delim {
		return errors.New("unsupported freebuff file: expected " + delim.String())
	}
	return nil
}

// ownTitle returns the title Freebuff asked its server for and saved in
// chat-title.txt, or "" when there is none. Older versions never wrote it.
func ownTitle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	title := strings.TrimSpace(string(data))
	if title == "" {
		return ""
	}
	return helpers.Truncate(title, 60, "…")
}

// cachedTitle returns firstPrompt from chat-meta.json, or "" when the cache is
// missing or no longer matches chat-messages.json. Freebuff makes the same
// size and mtime check; the mtime is a float from JavaScript, so it is
// compared to within a millisecond.
func cachedTitle(path string, messages fs.FileInfo) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var m struct {
		FirstPrompt     string  `json:"firstPrompt"`
		MessagesSize    int64   `json:"messagesSize"`
		MessagesMtimeMs float64 `json:"messagesMtimeMs"`
	}
	if json.Unmarshal(data, &m) != nil {
		return ""
	}
	mtime := float64(messages.ModTime().UnixNano()) / 1e6
	if m.MessagesSize != messages.Size() || math.Abs(m.MessagesMtimeMs-mtime) >= 1 {
		return ""
	}
	return helpers.Truncate(m.FirstPrompt, 60, "…")
}
