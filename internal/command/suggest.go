package command

import "strings"

// Suggest lists what the last word of line can become.
func (s *Service) Suggest(line string) []Suggestion {
	ws, open := split(line)
	if !open && (line == "" || strings.HasSuffix(line, " ")) {
		ws = append(ws, word{})
	}

	// still typing the command name
	if len(ws) <= 1 {
		typed := ""
		if len(ws) == 1 {
			typed = ws[0].text
		}
		rows := make([]Suggestion, 0, len(s.commands))
		for _, c := range s.commands {
			rows = append(rows, Suggestion{Text: c.Name, Hint: c.Help, Key: c.Key})
		}
		return Match(rows, typed)
	}

	c, ok := s.find(ws[0].text)
	if !ok {
		return nil
	}
	last := ws[len(ws)-1]

	// the bare value
	if !last.isTag() {
		return suggest(c.arg(""), s.env, last.text)
	}

	// a tag's value: "@name:typed"
	name, typed, ok := strings.Cut(last.text[1:], ":")
	if ok {
		if name == "" {
			return nil
		}
		return suggest(c.arg(name), s.env, typed)
	}

	// a tag's name: "@na" → the tags not used yet
	used := map[string]bool{}
	for _, w := range ws[1 : len(ws)-1] {
		if w.isTag() {
			n, _, _ := strings.Cut(w.text[1:], ":")
			used[n] = true
		}
	}
	var rows []Suggestion
	for _, a := range c.Args {
		if a.Name != "" && !used[a.Name] {
			rows = append(rows, Suggestion{Text: a.Name, Hint: a.Help})
		}
	}
	rows = Match(rows, name)
	for i := range rows {
		rows[i].Text = "@" + rows[i].Text + ":"
	}
	return rows
}

func suggest(a *Arg, env Env, typed string) []Suggestion {
	if a == nil || a.Suggest == nil {
		return nil
	}
	return a.Suggest(env, typed)
}

// Complete puts v in place of the last word of line.
func (s *Service) Complete(line string, v Suggestion) string {
	ws, open := split(line)
	if line == "" || (!open && strings.HasSuffix(line, " ")) {
		ws = append(ws, word{start: len(line)})
	}
	last := ws[len(ws)-1]
	text := v.Text
	head := ""
	if last.isTag() {
		if name, _, ok := strings.Cut(last.text[1:], ":"); ok {
			head = "@" + name + ":"
			last.quoted = strings.HasPrefix(line[last.start+len(head):], `"`)
		}
	}
	if !strings.HasSuffix(text, ":") && (last.quoted || strings.Contains(text, " ")) {
		text = `"` + text + `"`
	}
	out := line[:last.start] + head + text
	if !strings.HasSuffix(out, ":") {
		out += " "
	}
	return out
}
