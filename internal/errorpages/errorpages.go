// Copyright 2020-2024 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errorpages

import (
	"fmt"
	"html"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// TemplateData holds all the data that can be used in error page templates
type TemplateData struct {
	Code         int    `token:"code"`
	Message      string `token:"message"`
	Description  string `token:"description"`
	ShowDetails  bool   `token:"show_details"`
	Host         string `token:"host"`
	OriginalURI  string `token:"original_uri"`
	ForwardedFor string `token:"forwarded_for"`
	RequestID    string `token:"request_id"`
	NowUnix      int64  // registered as builtin function
	L10nEnabled  bool   // registered as custom function
	L10nScript   string // registered as custom function
}

// Values converts TemplateData fields into a map keyed by their token tags,
// suitable for registering as template functions.
func (d *TemplateData) Values() map[string]any {
	result := make(map[string]any)
	v := reflect.ValueOf(*d)
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if token, ok := t.Field(i).Tag.Lookup("token"); ok {
			result[token] = v.Field(i).Interface()
		}
	}
	return result
}

// Handler manages error page templates and detection
type Handler struct {
	templateText string // preprocessed template content
	version      string
}

// NewWithTemplate creates a handler that uses a Go template for error pages
func NewWithTemplate(templateBytes []byte, version string) (*Handler, error) {
	preprocessed := preprocessTemplate(string(templateBytes))
	return &Handler{
		templateText: preprocessed,
		version:      version,
	}, nil
}

// IsErrorStatus checks if a status code is in the 4xx or 5xx range
func IsErrorStatus(status string) bool {
	if len(status) != 3 {
		return false
	}
	return status[0] == '4' || status[0] == '5'
}

// RenderErrorPage renders the template with the provided data
func (h *Handler) RenderErrorPage(data *TemplateData) ([]byte, error) {
	if data.NowUnix == 0 {
		data.NowUnix = time.Now().Unix()
	}
	if data.Message == "" {
		data.Message = getStatusMessage(data.Code)
	}
	if data.Description == "" {
		data.Description = getStatusDescription(data.Code)
	}

	fns := template.FuncMap{
		"escape":       html.EscapeString,
		"nowUnix":      func() string { return strconv.FormatInt(data.NowUnix, 10) },
		"l10n_enabled": func() bool { return data.L10nEnabled },
		"l10nScript":   func() string { return data.L10nScript },
		"namespace":    func() string { return "" },
	}

	for k, v := range data.Values() {
		val := v
		fns[k] = func() any { return val }
	}

	tmpl, err := template.New("errorpage").Funcs(fns).Parse(h.templateText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return []byte(buf.String()), nil
}

// preprocessTemplate strips HTML/CSS/JS comment wrappers around Go template
// directives so that text/template can parse them natively. Value expressions
// like // {{ l10nScript }} are left untouched.
func preprocessTemplate(raw string) string {
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// HTML comment wrapper: <!-- {{...}} -->
		if strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->") {
			inner := strings.TrimPrefix(trimmed, "<!--")
			inner = strings.TrimSuffix(inner, "-->")
			inner = strings.TrimSpace(inner)
			if containsOnlyDirectives(inner) {
				lines[i] = ensureOuterTrimMarkers(inner)
				continue
			}
		}

		// CSS comment wrapper: /* {{...}} */
		if strings.HasPrefix(trimmed, "/*") && strings.HasSuffix(trimmed, "*/") {
			inner := strings.TrimPrefix(trimmed, "/*")
			inner = strings.TrimSuffix(inner, "*/")
			inner = strings.TrimSpace(inner)
			if containsOnlyDirectives(inner) {
				lines[i] = ensureOuterTrimMarkers(inner)
				continue
			}
		}

		// JS line comment wrapper: // {{...}}
		if strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "///") {
			inner := strings.TrimPrefix(trimmed, "//")
			inner = strings.TrimSpace(inner)
			if containsOnlyDirectives(inner) {
				lines[i] = ensureOuterTrimMarkers(inner)
				continue
			}
		}
	}
	return strings.Join(lines, "\n")
}

// containsOnlyDirectives checks whether s consists entirely of Go template
// actions ({{ ... }}) containing control-flow keywords, with only whitespace
// between them.
func containsOnlyDirectives(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	remaining := s
	found := false
	for len(remaining) > 0 {
		idx := strings.Index(remaining, "{{")
		if idx == -1 {
			return found && strings.TrimSpace(remaining) == ""
		}
		if strings.TrimSpace(remaining[:idx]) != "" {
			return false
		}
		endIdx := strings.Index(remaining[idx:], "}}")
		if endIdx == -1 {
			return false
		}

		action := remaining[idx+2 : idx+endIdx]
		action = strings.Trim(action, "- ")
		action = strings.TrimSpace(action)

		if !isControlKeyword(action) {
			return false
		}

		remaining = remaining[idx+endIdx+2:]
		found = true
	}
	return found
}

var controlKeywords = []string{"if ", "else if ", "else", "end", "range ", "with ", "block ", "define ", "template "}

func isControlKeyword(action string) bool {
	for _, kw := range controlKeywords {
		if action == strings.TrimSpace(kw) || strings.HasPrefix(action, kw) {
			return true
		}
	}
	return false
}

// ensureOuterTrimMarkers ensures the first {{ and last }} in the string have
// whitespace-trimming markers ({{- and -}}).
func ensureOuterTrimMarkers(s string) string {
	if strings.HasPrefix(s, "{{") && !strings.HasPrefix(s, "{{-") {
		s = "{{- " + strings.TrimLeft(s[2:], " ")
	}
	if strings.HasSuffix(s, "}}") && !strings.HasSuffix(s, "-}}") {
		s = strings.TrimRight(s[:len(s)-2], " ") + " -}}"
	}
	return s
}

// getStatusMessage returns the standard HTTP status message for a code
func getStatusMessage(code int) string {
	messages := map[int]string{
		// 4xx Client Errors
		400: "Bad Request",
		401: "Unauthorized",
		402: "Payment Required",
		403: "Forbidden",
		404: "Not Found",
		405: "Method Not Allowed",
		406: "Not Acceptable",
		407: "Proxy Authentication Required",
		408: "Request Timeout",
		409: "Conflict",
		410: "Gone",
		411: "Length Required",
		412: "Precondition Failed",
		413: "Payload Too Large",
		414: "URI Too Long",
		415: "Unsupported Media Type",
		416: "Range Not Satisfiable",
		417: "Expectation Failed",
		418: "I'm a teapot",
		421: "Misdirected Request",
		422: "Unprocessable Entity",
		423: "Locked",
		424: "Failed Dependency",
		425: "Too Early",
		426: "Upgrade Required",
		428: "Precondition Required",
		429: "Too Many Requests",
		431: "Request Header Fields Too Large",
		451: "Unavailable For Legal Reasons",

		// 5xx Server Errors
		500: "Internal Server Error",
		501: "Not Implemented",
		502: "Bad Gateway",
		503: "Service Unavailable",
		504: "Gateway Timeout",
		505: "HTTP Version Not Supported",
		506: "Variant Also Negotiates",
		507: "Insufficient Storage",
		508: "Loop Detected",
		510: "Not Extended",
		511: "Network Authentication Required",
	}

	if msg, ok := messages[code]; ok {
		return msg
	}

	if code >= 400 && code < 500 {
		return "Client Error"
	}
	return "Server Error"
}

// getStatusDescription returns a description for common HTTP status codes
func getStatusDescription(code int) string {
	descriptions := map[int]string{
		400: "The request could not be understood by the server due to malformed syntax.",
		401: "The request requires user authentication.",
		403: "The server understood the request, but is refusing to fulfill it.",
		404: "The requested resource could not be found.",
		405: "The method specified in the request is not allowed for the resource.",
		408: "The server timed out waiting for the request.",
		429: "Too many requests have been sent in a given amount of time.",
		500: "The server encountered an unexpected condition that prevented it from fulfilling the request.",
		502: "The server received an invalid response from the upstream server.",
		503: "The server is currently unable to handle the request due to temporary overloading or maintenance.",
		504: "The server did not receive a timely response from the upstream server.",
	}

	if desc, ok := descriptions[code]; ok {
		return desc
	}

	if code >= 400 && code < 500 {
		return "An error occurred while processing your request."
	}
	return "The server encountered an error while processing your request."
}
