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

func TestKVGridViewHighlightsSelectedCellOnly(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	g := NewKVGrid()
	g.Focus()
	g.Rows = []httpfile.KV{{Enabled: true, Key: "X-Key", Value: "some-value"}}

	highlightedKey := styleSelected.Render(pad("X-Key", 20))
	highlightedValue := styleSelected.Render("some-value")

	g.cursorCol = colKey
	row := strings.Split(g.View("Key", "Value"), "\n")[1]
	if !strings.Contains(row, highlightedKey) {
		t.Fatalf("expected key cell highlighted when colKey selected, got %q", row)
	}
	if strings.Contains(row, highlightedValue) {
		t.Fatalf("expected value cell NOT highlighted when colKey selected, got %q", row)
	}

	g.cursorCol = colValue
	row = strings.Split(g.View("Key", "Value"), "\n")[1]
	if strings.Contains(row, highlightedKey) {
		t.Fatalf("expected key cell NOT highlighted when colValue selected, got %q", row)
	}
	if !strings.Contains(row, highlightedValue) {
		t.Fatalf("expected value cell highlighted when colValue selected, got %q", row)
	}
}

func TestKVGridUnfocusedIgnoresInput(t *testing.T) {
	g := NewKVGrid()
	g, _ = g.Update(key("a"))
	if len(g.Rows) != 0 {
		t.Fatalf("expected unfocused grid to ignore input, got %d rows", len(g.Rows))
	}
}
