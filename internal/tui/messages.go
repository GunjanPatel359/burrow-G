package tui

// Messages the app sends to itself. Components keep their own (e.g. agentlist.SelectedMsg).

// refreshedMsg arrives when an agent's sessions were re-read into the database.
type refreshedMsg struct {
	agent string
	err   error
}

// openedMsg arrives when a session was opened in a new terminal tab.
type openedMsg struct {
	title string
	err   error
}
