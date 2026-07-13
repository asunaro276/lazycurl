package shell

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/asunaro276/lazycurl/internal/curlexec"
	"github.com/asunaro276/lazycurl/internal/tui/styles"
)

var (
	panelBorderFocused   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	panelBorderUnfocused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	panelTitle           = lipgloss.NewStyle().Bold(true)
	listSelected         = lipgloss.NewStyle().Reverse(true)
	overlayBox           = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	statusBarStyle       = lipgloss.NewStyle().Faint(true)
	errorStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	modeTabActive        = lipgloss.NewStyle().Bold(true).Reverse(true).Padding(0, 1)
	modeTabInactive      = lipgloss.NewStyle().Faint(true).Padding(0, 1)
)

// View renders the full shell: a mode-tab header, that mode's panel
// layout, and a status bar, with any active overlay drawn on top.
func (s *Shell) View() string {
	if s.width == 0 {
		return "loading..."
	}

	header := s.viewModeTabs()

	var body string
	if s.mode == ModeAdhoc {
		body = s.viewAdhocBody()
	} else {
		body = s.viewCollectionsBody()
	}

	main := lipgloss.JoinVertical(lipgloss.Left, header, body, s.viewStatusBar())

	switch s.overlay {
	case overlayHelp:
		return overlayBox.Render(s.viewHelp())
	case overlayEnvSelect:
		return overlayBox.Render(s.viewEnvSelect())
	case overlayNewCollection:
		return overlayBox.Render("新規コレクション名:\n\n> " + s.input + "_")
	case overlayConfirmDelete:
		return overlayBox.Render(fmt.Sprintf("リクエスト %q を削除しますか? (y/n)", s.currentRequestName()))
	case overlaySaveTarget:
		return overlayBox.Render(s.viewSaveTarget())
	}
	return main
}

// viewModeTabs renders the Adhoc/Collections mode indicator, highlighting
// whichever mode is currently active.
func (s *Shell) viewModeTabs() string {
	adhoc, collections := "Adhoc", "Collections"
	if s.mode == ModeAdhoc {
		adhoc = modeTabActive.Render(adhoc)
		collections = modeTabInactive.Render(collections)
	} else {
		adhoc = modeTabInactive.Render(adhoc)
		collections = modeTabActive.Render(collections)
	}
	return adhoc + " " + collections
}

// layoutDims computes the shared panel dimensions used by both the Adhoc
// and Collections body layouts.
func (s *Shell) layoutDims() (leftWidth, rightWidth, topHeight, bottomHeight int) {
	leftWidth = s.width / 4
	rightWidth = s.width - leftWidth - 1
	bodyHeight := s.height - 1 // mode-tab header line
	topHeight = bodyHeight * 2 / 3
	bottomHeight = bodyHeight - topHeight - 3 // leave room for status bar
	return leftWidth, rightWidth, topHeight, bottomHeight
}

// viewCollectionsBody renders the Collections mode's four-panel layout:
// Collections/Requests on the left, Response/History on the right.
func (s *Shell) viewCollectionsBody() string {
	leftWidth, rightWidth, topHeight, bottomHeight := s.layoutDims()

	collectionsPanel := s.renderPanel(PanelCollections, leftWidth, topHeight, s.viewCollections())
	requestsPanel := s.renderPanel(PanelRequests, leftWidth, bottomHeight, s.viewRequests())
	left := lipgloss.JoinVertical(lipgloss.Left, collectionsPanel, requestsPanel)

	responsePanel := s.renderPanel(PanelResponse, rightWidth, topHeight, s.viewResponse())
	historyPanel := s.renderPanel(PanelHistory, rightWidth, bottomHeight, s.viewHistory())
	right := lipgloss.JoinVertical(lipgloss.Left, responsePanel, historyPanel)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// viewAdhocBody renders the Adhoc mode's three-panel layout: the scratch
// request edit pane on the left, Response/History on the right.
func (s *Shell) viewAdhocBody() string {
	leftWidth, rightWidth, topHeight, bottomHeight := s.layoutDims()

	editPanel := s.renderPanel(PanelEdit, leftWidth, topHeight+bottomHeight, s.viewAdhocEdit())

	responsePanel := s.renderPanel(PanelResponse, rightWidth, topHeight, s.viewResponse())
	historyPanel := s.renderPanel(PanelHistory, rightWidth, bottomHeight, s.viewHistory())
	right := lipgloss.JoinVertical(lipgloss.Left, responsePanel, historyPanel)

	return lipgloss.JoinHorizontal(lipgloss.Top, editPanel, right)
}

func (s *Shell) currentRequestName() string {
	if s.requestIdx < len(s.requests) {
		return s.requests[s.requestIdx].Name
	}
	return ""
}

func (s *Shell) renderPanel(p Panel, w, h int, content string) string {
	style := panelBorderUnfocused
	if s.focus == p {
		style = panelBorderFocused
	}
	title := panelTitle.Render(panelLabels[p])
	body := title + "\n" + content
	return style.Width(w - 2).Height(h - 2).Render(body)
}

func (s *Shell) viewCollections() string {
	if len(s.collections) == 0 {
		return "(コレクションがありません。'n' で新規作成)"
	}
	var lines []string
	for i, c := range s.collections {
		line := c.Name
		if i == s.collectionIdx {
			line = listSelected.Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (s *Shell) viewRequests() string {
	if len(s.requests) == 0 {
		return "(リクエストがありません。'n' で新規作成)"
	}
	var lines []string
	for i, r := range s.requests {
		var line string
		if i == s.requestIdx {
			line = listSelected.Render(padMethod(r.Method) + " " + r.Name)
		} else {
			line = styles.MethodBadge(padMethod(r.Method)) + " " + r.Name
		}
		lines = append(lines, line)
	}
	env := s.activeEnvName()
	header := ""
	if env != "" {
		header = statusBarStyle.Render("env: "+env) + "\n"
	}
	return header + strings.Join(lines, "\n")
}

func padMethod(m string) string {
	if len(m) >= 7 {
		return m
	}
	return m + strings.Repeat(" ", 7-len(m))
}

func (s *Shell) viewResponse() string {
	if s.sending {
		return "送信中... (ctrl-c で中断)"
	}

	var entry *HistoryEntry
	if s.viewingIdx >= 0 && s.viewingIdx < len(s.history) {
		entry = &s.history[s.viewingIdx]
	} else if len(s.history) > 0 {
		entry = &s.history[len(s.history)-1]
	}
	if entry == nil {
		return "(まだリクエストが送信されていません)"
	}

	if entry.Err != nil {
		return errorStyle.Render(entry.Err.Error())
	}
	return renderResponse(entry.Response)
}

func renderResponse(resp *curlexec.Response) string {
	if resp == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(styles.StatusBadge(resp.StatusCode))
	b.WriteString(fmt.Sprintf("  %s\n\n", resp.TimeTotal))
	b.WriteString(panelTitle.Render("Headers") + "\n")
	for _, h := range resp.Headers {
		b.WriteString(h.Key + ": " + h.Value + "\n")
	}
	b.WriteString("\n" + panelTitle.Render("Body") + "\n")
	b.WriteString(string(resp.Body))
	return b.String()
}

func (s *Shell) viewHistory() string {
	if len(s.history) == 0 {
		return "(履歴はありません)"
	}
	var lines []string
	for i, h := range s.history {
		status := "ERR"
		if h.Err == nil && h.Response != nil {
			status = styles.StatusBadge(h.Response.StatusCode)
		}
		line := fmt.Sprintf("%s %s %s", h.At.Format("15:04:05"), status, h.Request.Name)
		if i == s.historyIdx {
			line = listSelected.Render(fmt.Sprintf("%s %s", h.At.Format("15:04:05"), h.Request.Name))
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (s *Shell) viewAdhocEdit() string {
	r := s.adhocRequest
	var b strings.Builder
	b.WriteString(styles.MethodBadge(padMethod(r.Method)) + " " + r.URL + "\n")
	if r.Name != "" {
		b.WriteString("\nName: " + r.Name)
	}
	b.WriteString(fmt.Sprintf("\nHeaders: %d", len(r.Headers)))
	if r.Body != "" {
		b.WriteString(fmt.Sprintf("\nBody: %d bytes", len(r.Body)))
	}
	b.WriteString("\n\n" + statusBarStyle.Render("e: 編集  enter: 送信  s: コレクションへ保存"))
	return b.String()
}

func (s *Shell) viewSaveTarget() string {
	var b strings.Builder
	b.WriteString(panelTitle.Render("保存先を選択") + "\n\n")
	newLabel := "+ 新規コレクションを作成"
	if s.saveOverlayIdx == 0 {
		newLabel = listSelected.Render(newLabel)
	}
	b.WriteString(newLabel + "\n")
	for i, c := range s.collections {
		line := c.Name
		if s.saveOverlayIdx == i+1 {
			line = listSelected.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (s *Shell) viewStatusBar() string {
	if s.statusMsg != "" {
		return errorStyle.Render(s.statusMsg)
	}
	if s.mode == ModeAdhoc {
		return statusBarStyle.Render("[/]: モード切替  tab: 切替  j/k: 移動  enter: 送信  e: 編集  s: 保存  ?: ヘルプ  q: 終了")
	}
	return statusBarStyle.Render("[/]: モード切替  tab: 切替  j/k: 移動  enter: 送信/選択  n: 新規  e: 編集  ?: ヘルプ  q: 終了")
}

func (s *Shell) viewHelp() string {
	var lines []string
	lines = append(lines, panelTitle.Render("キーバインド ("+panelLabels[s.focus]+")"))
	lines = append(lines, "")
	lines = append(lines, "[ / ]             モード切替 (Adhoc ⇄ Collections)")
	lines = append(lines, "tab / shift+tab   パネル間移動")
	lines = append(lines, "j/k               上下移動")
	lines = append(lines, "?                 このヘルプ")
	lines = append(lines, "q / ctrl-c        終了")
	lines = append(lines, "")
	switch s.focus {
	case PanelEdit:
		lines = append(lines, "e / n             リクエスト編集")
		lines = append(lines, "enter             リクエスト送信")
		lines = append(lines, "s                 コレクションへ保存")
	case PanelCollections:
		lines = append(lines, "enter             リクエスト一覧へ")
		lines = append(lines, "n                 新規コレクション作成")
		lines = append(lines, "E                 environment切り替え")
	case PanelRequests:
		lines = append(lines, "enter             リクエスト送信")
		lines = append(lines, "e                 リクエスト編集")
		lines = append(lines, "n                 新規リクエスト作成")
		lines = append(lines, "c                 複製")
		lines = append(lines, "d/x               削除")
		lines = append(lines, "E                 environment切り替え")
	case PanelHistory:
		lines = append(lines, "enter             選択した履歴をResponseパネルに表示")
	case PanelResponse:
		lines = append(lines, "(表示専用)")
	}
	lines = append(lines, "")
	lines = append(lines, "(閉じる: esc / ? / enter)")
	return strings.Join(lines, "\n")
}

func (s *Shell) viewEnvSelect() string {
	var b strings.Builder
	b.WriteString(panelTitle.Render("Environment を選択") + "\n\n")
	if len(s.envNames) == 0 {
		b.WriteString("(このコレクションにはenvironmentがありません)")
		return b.String()
	}
	for i, n := range s.envNames {
		line := n
		if i == s.envIdx {
			line = listSelected.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}
