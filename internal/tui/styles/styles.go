// Package styles centralizes lazycurl's color-coding rules: HTTP method
// badges and status-code badges, shared by the request list and response
// panels.
package styles

import "github.com/charmbracelet/lipgloss"

var methodColors = map[string]lipgloss.Color{
	"GET":     lipgloss.Color("39"),  // blue
	"POST":    lipgloss.Color("34"),  // green
	"PUT":     lipgloss.Color("214"), // orange
	"PATCH":   lipgloss.Color("214"), // orange
	"DELETE":  lipgloss.Color("196"), // red
	"HEAD":    lipgloss.Color("135"), // purple
	"OPTIONS": lipgloss.Color("244"), // gray
}

// MethodBadge renders an HTTP method as a colored badge.
func MethodBadge(method string) string {
	color, ok := methodColors[method]
	if !ok {
		color = lipgloss.Color("244")
	}
	return lipgloss.NewStyle().Bold(true).Foreground(color).Render(method)
}

// StatusColor returns the color associated with an HTTP status code's
// class (2xx/3xx/4xx/5xx), or a neutral color outside that range.
func StatusColor(code int) lipgloss.Color {
	switch {
	case code >= 200 && code < 300:
		return lipgloss.Color("34") // green
	case code >= 300 && code < 400:
		return lipgloss.Color("39") // blue
	case code >= 400 && code < 500:
		return lipgloss.Color("214") // orange
	case code >= 500:
		return lipgloss.Color("196") // red
	default:
		return lipgloss.Color("244") // gray
	}
}

// StatusBadge renders an HTTP status code as a colored badge.
func StatusBadge(code int) string {
	return lipgloss.NewStyle().Bold(true).Foreground(StatusColor(code)).Render(statusText(code))
}

func statusText(code int) string {
	if code <= 0 {
		return "---"
	}
	return itoa(code)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
