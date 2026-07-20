package form

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

func key(s string) tea.KeyMsg {
	switch s {
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestKVGridAddEditToggleDelete(t *testing.T) {
	g := NewKVGrid()
	g.Focus()

	g, _ = g.Update(key("a"))
	if len(g.Rows) != 1 {
		t.Fatalf("expected 1 row after 'a', got %d", len(g.Rows))
	}
	if !g.editing {
		t.Fatalf("expected to be editing key after 'a'")
	}

	g, _ = g.Update(key("X"))
	g, _ = g.Update(key("enter"))
	if g.Rows[0].Key != "X" {
		t.Fatalf("expected key 'X', got %q", g.Rows[0].Key)
	}
	if !g.editing {
		t.Fatalf("expected to move into editing value after enter on key")
	}

	g, _ = g.Update(key("Y"))
	g, _ = g.Update(key("enter"))
	if g.Rows[0].Value != "Y" {
		t.Fatalf("expected value 'Y', got %q", g.Rows[0].Value)
	}
	if g.editing {
		t.Fatalf("expected editing to stop after committing value")
	}
	if !g.Rows[0].Enabled {
		t.Fatalf("expected new row to default enabled")
	}

	g.cursorCol = colEnabled
	g, _ = g.Update(key(" "))
	if g.Rows[0].Enabled {
		t.Fatalf("expected row disabled after space toggle")
	}

	g, _ = g.Update(key("d"))
	if len(g.Rows) != 0 {
		t.Fatalf("expected 0 rows after delete, got %d", len(g.Rows))
	}
}

// TestKVGridEnterOnEmptyGridAddsRow confirms enter, not just 'a', starts a
// new row when the grid has none -- so a user reaching Params/Headers with
// nothing in it can start typing without knowing 'a' exists.
func TestKVGridEnterOnEmptyGridAddsRow(t *testing.T) {
	g := NewKVGrid()
	g.Focus()

	g, _ = g.Update(key("enter"))
	if len(g.Rows) != 1 {
		t.Fatalf("expected enter on an empty grid to add a row, got %d rows", len(g.Rows))
	}
	if !g.editing {
		t.Fatal("expected enter on an empty grid to start editing the new row's key")
	}
	if g.cursorCol != colKey {
		t.Fatalf("expected cursor on colKey after enter-created row, got %v", g.cursorCol)
	}
}

func TestKVGridEditingExistingKeyDoesNotAdvanceToValue(t *testing.T) {
	g := NewKVGrid()
	g.Focus()
	g.Rows = []httpfile.KV{{Enabled: true, Key: "X", Value: "Y"}}
	g.cursorCol = colKey

	g, _ = g.Update(key("enter")) // start editing existing key
	if !g.editing {
		t.Fatalf("expected to be editing key")
	}

	g, _ = g.Update(key("Z"))
	g, _ = g.Update(key("enter")) // commit
	if g.Rows[0].Key != "XZ" {
		t.Fatalf("expected key 'XZ', got %q", g.Rows[0].Key)
	}
	if g.editing {
		t.Fatalf("expected editing to stop after committing an existing row's key, not advance to value")
	}
	if g.cursorCol != colKey {
		t.Fatalf("expected cursor to stay on colKey, got %v", g.cursorCol)
	}
	if g.Rows[0].Value != "Y" {
		t.Fatalf("expected value unchanged, got %q", g.Rows[0].Value)
	}
}

func TestKVGridViewHighlightsSelectedCellOnly(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	g := NewKVGrid()
	g.Focus()
	g.Rows = []httpfile.KV{{Enabled: true, Key: "X-Key", Value: "some-value"}}

	cursoredKeyCell := renderKVCell("X-Key", true, false)
	plainKeyCell := renderKVCell("X-Key", false, false)
	cursoredValueCell := renderKVCell("some-value", true, false)
	plainValueCell := renderKVCell("some-value", false, false)

	g.cursorCol = colKey
	row := strings.Split(g.View("Key", "Value"), "\n")[1]
	if !strings.Contains(row, cursoredKeyCell) {
		t.Fatalf("expected key cell cursor-highlighted when colKey selected, got %q", row)
	}
	if !strings.Contains(row, plainValueCell) {
		t.Fatalf("expected value cell plain when colKey selected, got %q", row)
	}

	g.cursorCol = colValue
	row = strings.Split(g.View("Key", "Value"), "\n")[1]
	if !strings.Contains(row, plainKeyCell) {
		t.Fatalf("expected key cell plain when colValue selected, got %q", row)
	}
	if !strings.Contains(row, cursoredValueCell) {
		t.Fatalf("expected value cell cursor-highlighted when colValue selected, got %q", row)
	}
}

func TestKVGridViewHeaderAlignsWithBoxContent(t *testing.T) {
	g := NewKVGrid()
	g.Rows = []httpfile.KV{{Enabled: true, Key: "ZZZZ", Value: "WWWW"}}

	lines := strings.Split(g.View("Param", "Value"), "\n")
	header, row := lines[0], lines[1]

	keyIdx := strings.Index(row, "ZZZZ")
	if keyIdx == -1 || !strings.HasPrefix(header[keyIdx:], "Param") {
		t.Fatalf("expected header %q to align \"Param\" with key content in row %q", header, row)
	}

	valueIdx := strings.Index(row, "WWWW")
	if valueIdx == -1 || !strings.HasPrefix(header[valueIdx:], "Value") {
		t.Fatalf("expected header %q to align \"Value\" with value content in row %q", header, row)
	}
}

func TestKVGridUnfocusedIgnoresInput(t *testing.T) {
	g := NewKVGrid()
	g, _ = g.Update(key("a"))
	if len(g.Rows) != 0 {
		t.Fatalf("expected unfocused grid to ignore input, got %d rows", len(g.Rows))
	}
}
