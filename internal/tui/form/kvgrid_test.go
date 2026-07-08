package form

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestKVGridUnfocusedIgnoresInput(t *testing.T) {
	g := NewKVGrid()
	g, _ = g.Update(key("a"))
	if len(g.Rows) != 0 {
		t.Fatalf("expected unfocused grid to ignore input, got %d rows", len(g.Rows))
	}
}
