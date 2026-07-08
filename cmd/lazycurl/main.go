// Command lazycurl is a keyboard-driven terminal UI for building, storing,
// and executing HTTP requests via curl.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asunaro276/lazycurl/internal/collection"
	"github.com/asunaro276/lazycurl/internal/config"
	"github.com/asunaro276/lazycurl/internal/curlexec"
	"github.com/asunaro276/lazycurl/internal/environment"
	"github.com/asunaro276/lazycurl/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazycurl: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	versionResult, err := curlexec.CheckVersion(ctx)
	if err != nil {
		return err
	}
	if !versionResult.MeetsMinVerion {
		fmt.Fprintf(os.Stderr, "警告: curl %s が検出されましたが、%s 以上を推奨します(`-w '%%{json}'` が使用できない可能性があります)\n",
			versionResult.Version, curlexec.MinVersion)
	}

	dirs, err := config.EnsureDirs()
	if err != nil {
		return fmt.Errorf("設定ディレクトリの初期化に失敗しました: %w", err)
	}

	colStore := collection.NewStore(dirs.Collections)
	envStore := environment.NewStore(
		filepath.Join(dirs.Collections, "env"),
		filepath.Join(dirs.Root, "state.json"),
	)
	executor := curlexec.NewExecutor()

	app, err := tui.New(colStore, envStore, executor)
	if err != nil {
		return fmt.Errorf("起動に失敗しました: %w", err)
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
