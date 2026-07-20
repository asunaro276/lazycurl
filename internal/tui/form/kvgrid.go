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

	// advanceToValue is set only by the "a" (add row) flow so that
	// committing a brand-new row's key advances straight into editing its
	// value. Re-editing an existing row's key must not trigger this.
	advanceToValue bool
}

// NewKVGrid returns an empty, unfocused grid.
func NewKVGrid() KVGrid {
	ti := textinput.New()
	ti.Prompt = ""
	return KVGrid{input: ti, cursorCol: colKey}
}

func (g *KVGrid) Focus() { g.focused = true }
func (g *KVGrid) Blur() {
	g.focused = false
	g.cancelEdit()
}

func (g KVGrid) Focused() bool { return g.focused }

// Editing reports whether a cell is currently mid-edit (as opposed to mere
// row/column navigation). Used by Editor to decide whether an esc should be
// forwarded to the grid (cancel the cell edit) or should instead pop the
// grid itself out of focus.
func (g KVGrid) Editing() bool { return g.editing }

func (g *KVGrid) cancelEdit() {
	g.editing = false
	g.advanceToValue = false
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

// addRow appends a new enabled row and immediately starts editing its Key
// cell. Used by both the "a" key and by "enter" pressed on an empty grid,
// so a user can start typing a Param/Header without knowing "a" exists.
func (g *KVGrid) addRow() {
	g.Rows = append(g.Rows, httpfile.KV{Enabled: true})
	g.cursorRow = len(g.Rows) - 1
	g.cursorCol = colKey
	g.startEdit()
	g.advanceToValue = true
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
				advance := g.advanceToValue && g.cursorCol == colKey
				g.commitEdit()
				g.advanceToValue = false
				if advance {
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
		if len(g.Rows) == 0 {
			g.addRow()
		} else if g.cursorCol == colEnabled {
			g.Rows[g.cursorRow].Enabled = !g.Rows[g.cursorRow].Enabled
		} else {
			g.advanceToValue = false
			g.startEdit()
		}
	case "a":
		g.addRow()
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
	styleHeader     = lipgloss.NewStyle().Bold(true).Faint(true)
	styleSelected   = lipgloss.NewStyle().Reverse(true)
	styleDisabled   = lipgloss.NewStyle().Faint(true)
	styleBoxPlain   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleBoxCursor  = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	styleBoxEditing = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
)

// kvCellWidth is the inner text width of a Key/Value input box in View.
const kvCellWidth = 20

// kvBoxPrefix/kvBoxSuffix are the visible widths of a cell's "[ " / " ]"
// decoration; the header row's label offsets must match these exactly so
// header text lines up with the box's inner content, not its brackets.
const (
	kvBoxPrefix = 2
	kvBoxSuffix = 2
	kvColGap    = 3
)

// renderKVCell renders one Key or Value cell as an always-visible input box
// ("[ text ]"), so it reads as a form field rather than plain table text.
// The box brackets highlight to indicate cursor/edit state; cursor and
// editing never overlap since editing implies the cell is under the cursor.
func renderKVCell(content string, cursor, editing bool) string {
	style := styleBoxPlain
	switch {
	case editing:
		style = styleBoxEditing
	case cursor:
		style = styleBoxCursor
	}
	return style.Render("[") + " " + pad(content, kvCellWidth) + " " + style.Render("]")
}

// View renders the grid. keyLabel/valueLabel customize the header row
// (e.g. "Key"/"Value" for headers, "Param"/"Value" for query params).
func (g KVGrid) View(keyLabel, valueLabel string) string {
	var b []string

	// Label offsets mirror renderKVCell's "[ " prefix so header text lines
	// up with each box's inner content, fixing the header/row misalignment.
	headerKeyOffset := 3 + 1 + kvBoxPrefix // checkbox + separator + "[ "
	headerValueOffset := kvBoxSuffix + kvColGap + kvBoxPrefix
	header := strings.Repeat(" ", headerKeyOffset) + pad(keyLabel, kvCellWidth) + strings.Repeat(" ", headerValueOffset) + valueLabel
	b = append(b, styleHeader.Render(header))

	if len(g.Rows) == 0 {
		hint := "(no rows - press 'a' to add)"
		if g.focused {
			b = append(b, styleBoxCursor.Render(hint))
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
		editingKey := g.focused && g.editing && i == g.cursorRow && g.cursorCol == colKey
		editingValue := g.focused && g.editing && i == g.cursorRow && g.cursorCol == colValue
		if editingKey {
			key = g.input.View()
		} else if editingValue {
			value = g.input.View()
		}

		onRow := g.focused && i == g.cursorRow && !g.editing
		keyCell := renderKVCell(key, onRow && g.cursorCol == colKey, editingKey)
		valueCell := renderKVCell(value, onRow && g.cursorCol == colValue, editingValue)

		line := checkbox + " " + keyCell + strings.Repeat(" ", kvColGap) + valueCell
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
