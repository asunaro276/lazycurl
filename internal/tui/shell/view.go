package shell

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/asunaro276/lazycurl/internal/curlexec"
	"github.com/asunaro276/lazycurl/internal/tui/styles"
)

var (
	panelBorderFocused   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	panelBorderUnfocused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	panelBorderDimmed    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238"))
	panelTitle           = lipgloss.NewStyle().Bold(true)
	panelTitleDimmed     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))
	listSelected         = lipgloss.NewStyle().Reverse(true)
	overlayBox           = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	statusBarStyle       = lipgloss.NewStyle().Faint(true)
	errorStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// View renders the full shell: the always-visible 2x2 panel grid and a
// status bar, with any active overlay composited on top (centered, over a
// dimmed background) via compositeOverlay. If the overlay's rendered size
// doesn't fit the terminal, it falls back to a fullscreen render of the
// overlay alone.
func (s *Shell) View() string {
	if s.width == 0 {
		return "loading..."
	}

	if s.overlay == overlayNone {
		return lipgloss.JoinVertical(lipgloss.Left, s.viewGrid(false), s.viewStatusBar())
	}

	background := lipgloss.JoinVertical(lipgloss.Left, s.viewGrid(true), s.viewStatusBar())
	overlayContent := overlayBox.Render(s.currentOverlayBody())
	return compositeOverlay(background, overlayContent, s.width, s.height)
}

// currentOverlayBody returns the raw (unboxed) content for whichever overlay
// is currently active.
func (s *Shell) currentOverlayBody() string {
	switch s.overlay {
	case overlayHelp:
		return s.viewHelp()
	case overlayEnvSelect:
		return s.viewEnvSelect()
	case overlayNewCollection:
		return "新規コレクション名:\n\n> " + s.input + "_"
	case overlaySaveTo:
		return s.viewSaveTo()
	case overlayRequestName:
		return s.viewRequestName()
	case overlayConfirmDelete:
		return fmt.Sprintf("リクエスト %q を削除しますか? (y/n)", s.currentPreviewRequestName())
	}
	return ""
}

// compositeOverlay centers overlayContent over background. If overlayContent
// doesn't fit within termWidth/termHeight, background is discarded and
// overlayContent is returned as-is (a fullscreen fallback for narrow
// terminals). Splicing uses ansi.Cut, which is display-width-aware and
// preserves ANSI escape sequences, so already-colored panel content isn't
// corrupted by the cut.
func compositeOverlay(background, overlayContent string, termWidth, termHeight int) string {
	ovLines := strings.Split(overlayContent, "\n")
	ovWidth := 0
	for _, l := range ovLines {
		if w := ansi.StringWidth(l); w > ovWidth {
			ovWidth = w
		}
	}
	ovHeight := len(ovLines)
	if ovWidth > termWidth || ovHeight > termHeight {
		return overlayContent
	}

	bgLines := strings.Split(background, "\n")
	offsetX := (termWidth - ovWidth) / 2
	offsetY := (termHeight - ovHeight) / 2

	for i, ol := range ovLines {
		bgIdx := offsetY + i
		if bgIdx < 0 || bgIdx >= len(bgLines) {
			continue
		}
		bg := bgLines[bgIdx]
		left := ansi.Cut(bg, 0, offsetX)
		right := ansi.Cut(bg, offsetX+ansi.StringWidth(ol), len(bg))
		bgLines[bgIdx] = left + ol + right
	}
	return strings.Join(bgLines, "\n")
}

// viewGrid renders the shell's fixed 2x2 panel layout: [0] Request and
// [1] Response on top, [2] Collections and [3] History on the bottom. No
// mode or layout variant exists -- all four panels are always shown. When
// dimmed is true (an overlay is active above it), panels render with a muted
// style instead of their normal focused/unfocused colors.
func (s *Shell) viewGrid(dimmed bool) string {
	leftWidth := s.width / 2
	rightWidth := s.width - leftWidth - 1
	topHeight := s.height * 2 / 3
	bottomHeight := s.height - topHeight - 3 // leave room for status bar

	s.editor.SetSize(leftWidth-4, topHeight-4)

	requestPanel := s.renderPanel(PanelRequest, leftWidth, topHeight, s.editor.View(), dimmed)
	responsePanel := s.renderPanel(PanelResponse, rightWidth, topHeight, s.viewResponse(), dimmed)
	top := lipgloss.JoinHorizontal(lipgloss.Top, requestPanel, responsePanel)

	collectionsPanel := s.renderPanel(PanelCollections, leftWidth, bottomHeight, s.viewCollectionsAccordion(), dimmed)
	historyPanel := s.renderPanel(PanelHistory, rightWidth, bottomHeight, s.viewHistoryAccordion(), dimmed)
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, collectionsPanel, historyPanel)

	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

func (s *Shell) viewSaveTo() string {
	var b strings.Builder
	b.WriteString(panelTitle.Render("保存先コレクションを選択") + "\n\n")
	for i, c := range s.collections {
		line := c.Name
		if i == s.saveIdx {
			line = listSelected.Render(line)
		}
		b.WriteString(line + "\n")
	}
	newLine := "+ 新規コレクションを作成"
	if s.saveIdx == len(s.collections) {
		newLine = listSelected.Render(newLine)
	}
	b.WriteString(newLine)
	return b.String()
}

// viewRequestName renders the save-time name prompt shown when the target
// request (the loaded collection request, or the scratch request) has no
// name yet.
func (s *Shell) viewRequestName() string {
	return panelTitle.Render("リクエスト名を入力") + "\n\n> " + s.input + "_"
}

func (s *Shell) currentPreviewRequestName() string {
	if s.collectionReqIdx >= 0 && s.collectionReqIdx < len(s.previewRequests) {
		return s.previewRequests[s.collectionReqIdx].Name
	}
	return ""
}

func (s *Shell) renderPanel(p Panel, w, h int, content string, dimmed bool) string {
	style := panelBorderUnfocused
	title := panelTitle
	switch {
	case dimmed:
		style = panelBorderDimmed
		title = panelTitleDimmed
	case s.focus == p:
		style = panelBorderFocused
	}
	body := title.Render(panelLabels[p]) + "\n" + content
	return style.Width(w - 2).Height(h - 2).Render(body)
}

// viewCollectionsAccordion renders the Collections panel: the collection
// under the cursor (collectionIdx) is always expanded to show its request
// list (previewRequests), with whichever row the cursor rests on
// highlighted (the header itself, when collectionReqIdx == -1). Other
// collections stay collapsed to a single name line.
func (s *Shell) viewCollectionsAccordion() string {
	if len(s.collections) == 0 {
		return "(コレクションがありません。'N' で新規作成)"
	}

	var lines []string
	for i, c := range s.collections {
		if i != s.collectionIdx {
			lines = append(lines, c.Name)
			continue
		}

		header := c.Name
		if s.collectionReqIdx == -1 {
			header = listSelected.Render(header)
		}
		if env := s.activeEnvName(); env != "" {
			header += "  " + statusBarStyle.Render("env: "+env)
		}
		lines = append(lines, header)

		if len(s.previewRequests) == 0 {
			lines = append(lines, statusBarStyle.Render("  (リクエストがありません。'n' で新規作成)"))
			continue
		}
		for j, r := range s.previewRequests {
			summary := padMethod(r.Method) + " " + r.Name
			if j == s.collectionReqIdx {
				lines = append(lines, "  "+listSelected.Render(summary))
			} else {
				lines = append(lines, "  "+styles.MethodBadge(padMethod(r.Method))+" "+r.Name)
			}
		}
	}
	return strings.Join(lines, "\n")
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

// viewHistoryAccordion renders the History panel: the entry under the
// cursor (historyIdx) is expanded to preview its method/URL and outcome; it
// only takes effect on [1] Response once enter is pressed (viewingIdx).
// Other entries stay collapsed to a single summary line.
func (s *Shell) viewHistoryAccordion() string {
	if len(s.history) == 0 {
		return "(履歴はありません)"
	}
	var lines []string
	for i, h := range s.history {
		status := "ERR"
		if h.Err == nil && h.Response != nil {
			status = styles.StatusBadge(h.Response.StatusCode)
		}
		summary := fmt.Sprintf("%s %s %s", h.At.Format("15:04:05"), status, h.Request.Name)

		if i != s.historyIdx {
			lines = append(lines, summary)
			continue
		}

		lines = append(lines, listSelected.Render(summary))
		lines = append(lines, "  "+h.Request.Method+" "+h.Request.URL)
		if h.Err != nil {
			lines = append(lines, "  "+errorStyle.Render(h.Err.Error()))
		} else if h.Response != nil {
			lines = append(lines, fmt.Sprintf("  %s", h.Response.TimeTotal))
		}
	}
	return strings.Join(lines, "\n")
}

func (s *Shell) viewStatusBar() string {
	if s.statusMsg != "" {
		return errorStyle.Render(s.statusMsg)
	}
	return statusBarStyle.Render(s.footerHint())
}

// footerHint derives the keybinding hint line from the currently focused
// panel and, for the Request panel, its normal/insert state and active
// tab -- so the footer always reflects exactly the keys usable right now.
func (s *Shell) footerHint() string {
	switch s.focus {
	case PanelRequest:
		if s.editor.Editing() {
			return "esc: 前の階層に戻る  ctrl+r: 送信  ctrl+s: 保存  ctrl+c: 終了"
		}
		return "j/k: 移動  h/l: Method変更  enter: 編集開始  [/]: タブ切替  ctrl+r: 送信  ctrl+s: 保存  0-3/tab: パネル移動  ?: ヘルプ  q: 終了"
	case PanelCollections:
		return "j/k: 移動  enter: 開く/ロード  n: 新規リクエスト  N: 新規コレクション  c: 複製  d/x: 削除  E: environment切替  0-3/tab: パネル移動  ?: ヘルプ  q: 終了"
	case PanelHistory:
		return "j/k: プレビュー  enter: Responseへ反映  0-3/tab: パネル移動  ?: ヘルプ  q: 終了"
	case PanelResponse:
		return "0-3/tab: パネル移動  ?: ヘルプ  q: 終了"
	}
	return ""
}

// viewHelp renders the keybinding help overlay.
func (s *Shell) viewHelp() string {
	var lines []string
	lines = append(lines, panelTitle.Render("キーバインド ("+panelLabels[s.focus]+")"))
	lines = append(lines, "")
	lines = append(lines, "tab / shift+tab   パネル間移動")
	lines = append(lines, "0-3               パネルへジャンプ (insert状態を除く)")
	lines = append(lines, "j/k               上下移動")
	lines = append(lines, "?                 このヘルプ")
	lines = append(lines, "q / ctrl-c        終了")
	lines = append(lines, "")
	switch s.focus {
	case PanelRequest:
		lines = append(lines, "j/k               Method/URL/タブ内容間を移動 (normal)")
		lines = append(lines, "h/l               Methodを変更 (normal)")
		lines = append(lines, "enter             フィールドへinsert (normal -> insert)")
		lines = append(lines, "esc               1階層戻る (insert -> normal)")
		lines = append(lines, "[ / ]             タブ切替 (normal, Params/Headers/Auth/Body)")
		lines = append(lines, "ctrl+r            送信")
		lines = append(lines, "ctrl+s            保存 (無名なら名前を確認)")
	case PanelCollections:
		lines = append(lines, "enter             ヘッダ: 展開  リクエスト行: [0] Requestへロード")
		lines = append(lines, "n                 選択中コレクションに新規リクエスト")
		lines = append(lines, "N                 新規コレクション作成")
		lines = append(lines, "c                 リクエストを複製")
		lines = append(lines, "d/x               リクエストを削除")
		lines = append(lines, "E                 environment切り替え")
	case PanelHistory:
		lines = append(lines, "j/k               プレビュー展開")
		lines = append(lines, "enter             選択した履歴を[1] Responseパネルへ反映")
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
