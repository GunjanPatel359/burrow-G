# Bubble Tea cheat sheet — from zero to our TUI

Read top to bottom once. After that, use the tables as a quick reference.
Part 1 is theory, Part 2 is a tiny complete example, Part 3 is the building
blocks, Part 4 walks through **our** code, and Part 5 lists traps we hit.

---

## Part 1 — The theory

### 1.1 What a TUI is

A terminal is just a **grid of characters** (e.g. 110 columns × 30 rows).
A TUI ("terminal user interface") is a program that:

1. clears the grid,
2. prints a picture made of text and colour codes,
3. waits for a key press,
4. prints a new picture,
5. repeats until you quit.

That's all. There are no real buttons or windows — "a list with a highlighted
row" is just text, where one line has a blue background colour code.

### 1.2 The one idea behind Bubble Tea: Model → Update → View

Bubble Tea uses the **Elm Architecture**. Your whole app is three things:

| Piece | Question it answers | In Go |
|---|---|---|
| **Model** | *What is the state right now?* | a struct |
| **Update** | *Something happened — how does the state change?* | `func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd)` |
| **View** | *What does this state look like?* | `func (m model) View() string` |

Bubble Tea runs this loop for you:

```
          ┌───────────────────────────────┐
          │            MODEL              │  ← all state lives here
          └───────────────────────────────┘
             ▲                        │
  new model  │                        │  View(model) → string
             │                        ▼
   ┌─────────┴─────────┐     ┌──────────────────┐
   │  Update(msg)      │     │ terminal shows   │
   └─────────▲─────────┘     │ that string      │
             │               └──────────────────┘
             │  msg = "something happened"
   key press · window resized · background job finished
```

Rules that make this easy to reason about:

- **Only `Update` changes state.** Nothing else touches the model.
- **`View` only reads.** It turns state into text, nothing more. Bubble Tea
  calls it after every `Update`, so the screen always matches the state.
- **Everything is a message.** Key press, resize, timer, "refresh done" — all
  arrive in `Update` the same way, one at a time. No two run at once, so you
  never need locks.

### 1.3 Messages (`tea.Msg`)

A message is *any* Go value. `Update` uses a **type switch** to see what it got:

```go
switch msg := msg.(type) {
case tea.KeyMsg:          // a key was pressed; msg.String() is "q", "enter", "ctrl+c", "down"…
case tea.WindowSizeMsg:   // terminal resized; msg.Width, msg.Height
case refreshedMsg:        // OUR OWN message type (any struct works)
}
```

Built-in messages you'll meet:

| Message | When |
|---|---|
| `tea.KeyMsg` | every key press |
| `tea.WindowSizeMsg` | at start, and whenever the terminal is resized |
| `tea.QuitMsg` | sent by `tea.Quit`; ends the program |

### 1.4 Commands (`tea.Cmd`) — how slow work happens

`Update` must be **fast** (it blocks the screen). So slow work — reading files,
the network, a database — goes into a **command**:

```go
type Cmd func() Msg     // that's literally the definition
```

A command is *a function that does work and returns a message*. You **return**
it from `Update` (or `Init`); Bubble Tea runs it **in the background**, and when
it finishes, its returned message comes back into `Update`.

```
Update returns cmd ──► Bubble Tea runs cmd() in background ──► msg ──► Update(msg)
     (instant)              (takes as long as it takes)
```

Useful ones:

| Thing | What it is |
|---|---|
| `nil` | "no command" — nothing to do |
| `tea.Quit` | a ready-made command that ends the program |
| `tea.Batch(a, b)` | run several commands at once |

### 1.5 `Init` and starting the program

```go
func (m model) Init() tea.Cmd   // runs once at start; return a first command or nil

p := tea.NewProgram(startModel, tea.WithAltScreen())  // AltScreen: full-screen, restores terminal on exit
final, err := p.Run()                                  // blocks until quit; returns the LAST model
```

`Run` handing back the final model is how a TUI returns a result to the rest
of the program (we use it to know which session you picked).

### 1.6 Why `Update` returns the model

`Update` has a **value receiver** (`m model`, not `m *model`). It gets a
*copy*, changes the copy, and **returns** it. Bubble Tea keeps whatever you
return. If you change `m` but forget to return it, the change is lost.

Inside, small helper methods with a **pointer receiver** (`func (m *model) …`)
are fine for editing the copy in place before returning it.

---

## Part 2 — A complete tiny example (a counter)

```go
package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	count int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			m.count++
		case "down":
			m.count--
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return fmt.Sprintf("Count: %d\n\n↑/↓ change · q quit\n", m.count)
}

func main() {
	if _, err := tea.NewProgram(model{}).Run(); err != nil {
		fmt.Println(err)
	}
}
```

What happens when you press ↑:

1. Bubble Tea turns the key into `tea.KeyMsg` with `String() == "up"`.
2. It calls `Update(msg)` → we do `m.count++` and return the new model.
3. It calls `View()` on the new model → `"Count: 1 …"`.
4. It redraws the terminal with that string.

Everything in our real app is this same loop with more state.

---

## Part 3 — The building blocks we use

### 3.1 Bubbles: ready-made components

**Bubbles** is a library of components (list, text input, spinner, table…).
Each component is itself a little Model/Update/View. You keep it **inside your
model** and **forward messages to it**:

```go
type model struct {
	agents list.Model          // a component lives in our state
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// handle OUR keys first … then pass everything else on:
	var cmd tea.Cmd
	m.agents, cmd = m.agents.Update(msg)   // component updates itself
	return m, cmd
}

func (m model) View() string {
	return m.agents.View()                 // component draws itself
}
```

The list then handles ↑/↓, paging, and `/` search for free.

### 3.2 `list.Model` — the parts

| Part | What it is | Ours |
|---|---|---|
| **Item** | anything with `FilterValue() string` (the text `/` searches) | `agentItem`, `sessionItem` |
| **Delegate** | decides how ONE row is drawn | `agentDelegate`, `sessionDelegate` |
| `SetItems(items)` | replace the rows | `loadAgents`, `loadSessions` |
| `SelectedItem()` | the row under the cursor | `open` / details box |
| `FilterState()` | `Unfiltered`, `Filtering` (typing), `FilterApplied` | don't steal keys while typing |
| `SetSize(w, h)` | how much space it gets | `resize()` |

A **delegate** is an interface with 4 methods:

```go
Height() int                                   // lines per row (ours: 1)
Spacing() int                                  // blank lines between rows (ours: 0)
Update(msg tea.Msg, m *list.Model) tea.Cmd     // per-row key handling (ours: none)
Render(w io.Writer, m list.Model, index int, item list.Item)  // WRITE the row text
```

`Render` gets the list (for `m.Width()` and `m.Index()` = selected row) and
writes one line. That's where the `▌`, `★` and right-aligned time come from.

### 3.3 Key bindings

```go
keyQuit := key.NewBinding(key.WithKeys("q", "ctrl+c"))   // a named set of keys
if key.Matches(msg, keyQuit) { … }                       // is this KeyMsg one of them?
```

Nicer than comparing strings everywhere, and one place to change keys.

### 3.4 Lip Gloss: styling (CSS for the terminal)

```go
style := lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7")).Bold(true)
style.Render("Agents")          // → "Agents" wrapped in colour codes

lipgloss.NewStyle().Width(20).Height(10).Render(s)       // pad/clip s to a box
lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)  // put boxes side by side
lipgloss.Width(s)               // visible width of s (ignores colour codes)
```

**Colour codes are invisible characters.** `"\x1b[34mHi\x1b[0m"` shows as 2
characters but `len()` says 11. Always measure with `lipgloss.Width`, and cut
text to size **before** styling it.

---

## Part 4 — Our TUI, file by file

The UI is split so that each piece is a small Bubble Tea program of its own
(a **component**), and one root model wires them together.

```
internal/tui/
  app.go        root model: owns the components, routes messages, decides focus
  layout.go     View(): puts the components on screen
  keys.go       every key the app handles
  messages.go   every message the app sends itself
  commands.go   every background job (the only code touching db / files / terminals)
  theme/
    palettes.go   colours per theme — add a theme = add one entry
    theme.go      palette → styles; Get(name), Names(), Row()
  components/
    agentlist/    left pane   (Model, Update, View, row drawing)
    sessionlist/  right pane  (rows + search)
    details/      the 4-line details box
    statusbar/    "● status" + key help
```

Three rules keep it manageable as it grows:

1. **A component never calls another component.** `agentlist` doesn't know
   `sessionlist` exists. It just sends `agentlist.SelectedMsg{Name}` and `app.go`
   decides what that means.
2. **Components never touch the database, files or terminal.** Only
   `commands.go` does slow or outside work, as `tea.Cmd`s.
3. **No colour is written outside `theme/palettes.go`.** Components receive a
   `theme.Theme` in `New(...)` and have `SetTheme(...)`, so switching themes later
   is "give every component the new theme".

### 4.1 The root model (`app.go:27`)

```go
type model struct {
	store   *db.DB            // our database
	sources []session.Source  // one per agent (claude, qoder, …)
	theme   theme.Theme

	agents   agentlist.Model   // left pane
	sessions sessionlist.Model // right pane
	details  details.Model     // box under the sessions
	status   statusbar.Model   // bottom line

	focus    pane              // agentPane or sessionPane
	agent    session.Source    // which agent the right pane shows
	chosen   *session.Session  // set when you resume here (o)
	terminal launch.Terminal   // nil when new tabs aren't supported

	width, height int
}
```

Notice the root model holds **other models**. That nesting is the whole trick.

### 4.2 Start-up

- `newModel` (`app.go:45`) creates each component with the theme, loads agent
  counts and the first agent's sessions **from the database**, focuses the agent pane.
- `Init` (`app.go:79`) returns `refresh(first agent)` — after the first draw, that
  agent's files are re-read in the background.
- `Run` (`app.go:71`) starts the program full-screen and returns `chosen`.

### 4.3 `Update` — what each event does (`app.go:86`)

| Event | Result |
|---|---|
| `tea.WindowSizeMsg` | remember size, `resize()` every component |
| `refreshedMsg` | reload agent counts; if it's the shown agent, reload its sessions; status ● refreshed |
| `agentlist.SelectedMsg` | switch the right pane to that agent + background refresh (`selectAgent`, `app.go:159`) |
| `openedMsg` | status "● opened … in a new tab" or the error |
| `q` / `ctrl+c` | `tea.Quit` |
| `r` | status refreshing…, return `refresh(agent)` |
| `enter` / `→` / `tab` on agents | focus sessions pane |
| `enter` on a session (new tabs supported) | return `openTab(...)` — burrow-G stays open |
| `enter` (no tab support) or `o` | `chosen = selected`, `tea.Quit` → main resumes here |
| `←` / `esc` on sessions | focus agents pane |
| anything else | forwarded to the **focused component** |

While you type a search, every key goes to `sessionlist` — otherwise typing
"q" in the search box would quit.

### 4.4 A full trip: pressing ↓ on the agent pane

```
1. you press ↓
2. app.Update gets tea.KeyMsg{"down"} — not one of the app's keys → forwarded:
       m.agents, cmd = m.agents.Update(msg)
3. inside agentlist.Update (components/agentlist/model.go:88):
       the list moves its cursor to "Qoder"
       the selected name changed → it returns a Cmd that produces SelectedMsg{"qoder"}
4. Bubble Tea runs that Cmd → SelectedMsg{"qoder"} arrives in app.Update
5. app.selectAgent("qoder"):
       m.agent = qoder source
       m.loadSessions()                    // Qoder rows from the DB (instant)
       m.status.Busy("refreshing…")
       return refresh(db, qoder)           // background job
6. View draws Qoder's sessions with a yellow "● refreshing…"
7. in the background: store.Refresh(qoder) re-reads ~/.qoder → refreshedMsg{"qoder"}
8. app.Update(refreshedMsg) → loadSessions() again (fresh) → status.OK("refreshed")
```

Step 3→4 is the "components talk through messages" rule in action.

### 4.5 Inside a component (`components/sessionlist/model.go`)

A component looks exactly like the counter from Part 2, plus setters the root
model calls:

| Part | What it does |
|---|---|
| `New(theme, home)` | build the inner `list.Model` with our delegate |
| `SetSessions`, `SetSize`, `SetFocused`, `SetTheme` | the root model pushes data/state in |
| `Selected()`, `Len()`, `Searching()` | the root model reads state out |
| `Update(msg)` | forward to the list; show the search box only while searching (`:81`) |
| `View()` | the list's text |
| `delegate.Render` (`:97`) | draws `★ title ……… 2 hours ago` using `theme.Row` |

`theme.Row` (`theme/theme.go:62`) draws any row: left text, right text at the
far end, `▌` marker and highlight when selected. Both lists use it, so rows look
the same everywhere and follow the theme.

### 4.6 The layout (`layout.go:33`)

```
 Agents (20 wide) │ "Claude Code · 60"
 agentlist        │ sessionlist          ← listHeight()
                  │ ──────────────
                  │ details              ← details.Height lines
──────────────────────────────────────
 statusbar
```

`View()` asks each component for its string, boxes them to size with Lip Gloss,
and joins left and right with `lipgloss.JoinHorizontal` (`layout.go:48`).

### 4.7 How we test without a terminal

- **Each component** has its own `model_test.go`: call `Update` with a fake key,
  check `Selected()` or `View()`.
- **The whole app** (`app_test.go`) uses three helpers:

```go
m, cmd := press(m, enter)   // call Update with a fake key
m = deliver(m, cmd)         // run the returned Cmd, feed its messages back in
m.View()                    // check the text contains what we expect
```

A fake agent (`fakeSource`) and a fake terminal (`fakeTerminal`) stand in for the
real ones, so tests never read your real sessions or open real tabs.

## Part 5 — Traps we hit (and the fix)

| Trap | What happened | Fix |
|---|---|---|
| Component's own keys | `list` has built-in quit keys (`q`, `esc`) and turns them back on when items change → `esc` quit the app | `l.DisableQuitKeybindings()` in each list component's `New` and handle `q`/`ctrl+c` ourselves |
| Shadowing a package | a variable named `list` hid the `list` package → "list.Item is not a type" | name it `rows` |
| Colour codes have length | cutting an already-coloured string counted invisible codes | cut plain text first, then style |
| Hidden search row | list reserved a blank line for the search box | show it only while searching (`components/sessionlist/model.go:81`) |
| Forgetting to return `m` | changes in `Update` vanish | always `return m, cmd` |
| Slow work in `Update` | would freeze the screen | put it in a `tea.Cmd` |

---

## Quick reference

```go
// the three methods every model needs
func (m model) Init() tea.Cmd
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m model) View() string

// start
tea.NewProgram(m, tea.WithAltScreen()).Run()

// messages
tea.KeyMsg            msg.String() == "enter" | "q" | "ctrl+c" | "up" | "down" | "esc" | "tab"
tea.WindowSizeMsg     msg.Width, msg.Height
type myMsg struct{…}  // your own

// commands
nil · tea.Quit · tea.Batch(c1, c2) · func() tea.Msg { …slow work…; return myMsg{} }

// list
list.New(items, delegate, w, h) · SetItems · SelectedItem · Index · SetSize · FilterState
l.SetShowTitle/SetShowHelp/SetShowStatusBar/SetShowFilter/SetFilteringEnabled(bool)

// keys
key.NewBinding(key.WithKeys("a", "b")) · key.Matches(msg, binding)

// lipgloss
lipgloss.NewStyle().Foreground(c).Background(c).Bold(true).Width(n).Height(n).Render(s)
lipgloss.JoinHorizontal(lipgloss.Top, a, b) · lipgloss.JoinVertical(lipgloss.Left, a, b)
lipgloss.Width(s)
```

## Try it yourself (5-minute exercises)

1. In `theme/palettes.go`, change `Warning` to `"#ff0000"` — the ★ turns red. (Change it back.)
2. In `layout.go`, change `leftWidth` from 20 to 26 — the agent pane gets wider.
3. In `keys.go` → `agentHelp`, change the help text — the bottom line changes.
   That's `View` at work: it's only text built from the model.
4. Run the counter from Part 2: save it as `/tmp/counter/main.go` with a
   `go mod init counter && go get github.com/charmbracelet/bubbletea@v1.3.10`.
