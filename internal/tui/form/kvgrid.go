// Package form implements the request editor's hybrid form UI: key-value
// grid editing for Params/Headers, a type selector for Auth, and a
// textarea (with $EDITOR escape hatch) for Body.
package form

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

// column identifies which cell of a KVGrid row is under the cursor.
type column int

const (
	colEnabled column = iota
	colKey
	colValue
)

// KVGrid is a key-value grid editor for headers and query params: each row
// has an enabled checkbox, a key, and a value. Rows can be added, deleted,
// toggled, and edited in place.
type KVGrid struct {
	Rows []httpfile.KV

	cursorRow int
	cursorCol column
	editing   bool
	input     textinput.Model
	focused   bool
}

// NewKVGrid returns an empty, unfocused grid.
func NewKVGrid() KVGrid {
	ti := textinput.New()
	ti.Prompt = ""
	return KVGrid{input: ti}
}

func (g *KVGrid) Focus() { g.focused = true }
func (g *KVGrid) Blur() {
	g.focused = false
	g.cancelEdit()
}

func (g KVGrid) Focused() bool { return g.focused }

func (g *KVGrid) cancelEdit() {
	g.editing = false
	g.input.Blur()
}

func (g *KVGrid) startEdit() {
	if len(g.Rows) == 0 || g.cursorCol == colEnabled {
		return
	}
	row := g.Rows[g.cursorRow]
	if g.cursorCol == colKey {
		g.input.SetValue(row.Key)
	} else {
		g.input.SetValue(row.Value)
	}
	g.input.CursorEnd()
	g.input.Focus()
	g.editing = true
}

func (g *KVGrid) commitEdit() {
	if !g.editing || len(g.Rows) == 0 {
		return
	}
	val := g.input.Value()
	if g.cursorCol == colKey {
		g.Rows[g.cursorRow].Key = val
	} else if g.cursorCol == colValue {
		g.Rows[g.cursorRow].Value = val
	}
	g.editing = false
	g.input.Blur()
}

// Update handles a key message when the grid is focused, returning any
// command produced (e.g. textinput cursor blink).
func (g KVGrid) Update(msg tea.Msg) (KVGrid, tea.Cmd) {
	if !g.focused {
		return g, nil
	}

	if g.editing {
		switch m := msg.(type) {
		case tea.KeyMsg:
			switch m.String() {
			case "enter":
				g.commitEdit()
				if g.cursorCol == colKey {
					g.cursorCol = colValue
					g.startEdit()
				}
				return g, nil
			case "esc":
				g.cancelEdit()
				return g, nil
			}
		}
		var cmd tea.Cmd
		g.input, cmd = g.input.Update(msg)
		return g, cmd
	}

	m, ok := msg.(tea.KeyMsg)
	if !ok {
		return g, nil
	}

	switch m.String() {
	case "j", "down":
		if g.cursorRow < len(g.Rows)-1 {
			g.cursorRow++
		}
	case "k", "up":
		if g.cursorRow > 0 {
			g.cursorRow--
		}
	case "l", "right", "tab":
		if g.cursorCol < colValue {
			g.cursorCol++
		}
	case "h", "left", "shift+tab":
		if g.cursorCol > colEnabled {
			g.cursorCol--
		}
	case " ":
		if len(g.Rows) > 0 {
			g.Rows[g.cursorRow].Enabled = !g.Rows[g.cursorRow].Enabled
		}
	case "enter":
		if g.cursorCol == colEnabled {
			if len(g.Rows) > 0 {
				g.Rows[g.cursorRow].Enabled = !g.Rows[g.cursorRow].Enabled
			}
		} else {
			g.startEdit()
		}
	case "a":
		g.Rows = append(g.Rows, httpfile.KV{Enabled: true})
		g.cursorRow = len(g.Rows) - 1
		g.cursorCol = colKey
		g.startEdit()
	case "d", "x":
		if len(g.Rows) > 0 {
			g.Rows = append(g.Rows[:g.cursorRow], g.Rows[g.cursorRow+1:]...)
			if g.cursorRow >= len(g.Rows) && g.cursorRow > 0 {
				g.cursorRow--
			}
		}
	}
	return g, nil
}

var (
	styleHeader   = lipgloss.NewStyle().Bold(true).Faint(true)
	styleSelected = lipgloss.NewStyle().Reverse(true)
	styleDisabled = lipgloss.NewStyle().Faint(true)
)

// View renders the grid. keyLabel/valueLabel customize the header row
// (e.g. "Key"/"Value" for headers, "Param"/"Value" for query params).
func (g KVGrid) View(keyLabel, valueLabel string) string {
	var b []string
	b = append(b, styleHeader.Render("   "+pad(keyLabel, 20)+" "+valueLabel))

	if len(g.Rows) == 0 {
		hint := "(no rows - press 'a' to add)"
		if g.focused {
			b = append(b, styleSelected.Render(hint))
		} else {
			b = append(b, styleDisabled.Render(hint))
		}
		return lipgloss.JoinVertical(lipgloss.Left, b...)
	}

	for i, row := range g.Rows {
		checkbox := "[ ]"
		if row.Enabled {
			checkbox = "[x]"
		}

		key := row.Key
		value := row.Value
		if g.focused && g.editing && i == g.cursorRow {
			if g.cursorCol == colKey {
				key = g.input.View()
			} else if g.cursorCol == colValue {
				value = g.input.View()
			}
		}

		keyText := pad(key, 20)
		valueText := value

		selected := g.focused && i == g.cursorRow && !g.editing
		if selected && g.cursorCol == colKey {
			keyText = styleSelected.Render(keyText)
		} else if selected && g.cursorCol == colValue {
			valueText = styleSelected.Render(valueText)
		}

		line := checkbox + " " + keyText + " " + valueText
		if !row.Enabled {
			line = styleDisabled.Render(line)
		}
		b = append(b, line)
	}
	return lipgloss.JoinVertical(lipgloss.Left, b...)
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
