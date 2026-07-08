package curlexec

import "fmt"

// exitMessages maps common curl exit codes to human-readable Japanese
// messages. See `man curl` EXIT CODES.
var exitMessages = map[int]string{
	3:  "URLの形式が不正です",
	5:  "指定されたプロキシに解決できませんでした",
	6:  "ホスト名を解決できませんでした",
	7:  "接続できませんでした",
	18: "レスポンスの転送が不完全なまま終了しました",
	22: "サーバーがHTTPエラーを返しました",
	23: "ローカルでの書き込みに失敗しました",
	26: "リクエストbodyの読み込みに失敗しました",
	28: "タイムアウトしました",
	35: "TLS/SSLハンドシェイクに失敗しました",
	47: "リダイレクトが多すぎます",
	52: "サーバーから空のレスポンスが返されました",
	56: "レスポンスの受信中にエラーが発生しました",
	60: "TLS証明書の検証に失敗しました(自己署名証明書の場合は@insecureプラグマを検討してください)",
}

// ExitError represents a non-zero curl exit code, with a human-readable
// message alongside the raw exit code.
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("curl exited with code %d: %s", e.Code, e.Message)
}

func newExitError(code int) *ExitError {
	msg, ok := exitMessages[code]
	if !ok {
		msg = fmt.Sprintf("curlがエラー終了しました(終了コード %d)", code)
	}
	return &ExitError{Code: code, Message: msg}
}
