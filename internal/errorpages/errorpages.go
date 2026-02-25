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
	"strconv"
	"strings"
	"time"
)

// TemplateData holds all the data that can be used in error page templates
type TemplateData struct {
	Code         int    // HTTP status code (e.g., 404, 500)
	Message      string // HTTP status message (e.g., "Not Found", "Internal Server Error")
	Description  string // Longer description of the error
	ShowDetails  bool   // Whether to show detailed information
	Host         string // Request Host header
	OriginalURI  string // Original request URI
	ForwardedFor string // X-Forwarded-For header
	RequestID    string // Request ID for tracing
	NowUnix      int64  // Current Unix timestamp
	L10nEnabled  bool   // Whether localization is enabled
	L10nScript   string // Localization script content
}

// Handler manages error page templates and detection
type Handler struct {
	template string // Raw template content
	version  string
}

// NewWithTemplate creates a handler that uses a Go template for error pages
func NewWithTemplate(templateBytes []byte, version string) (*Handler, error) {
	return &Handler{
		template: string(templateBytes),
		version:  version,
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
	// Set timestamp if not already set
	if data.NowUnix == 0 {
		data.NowUnix = time.Now().Unix()
	}

	// Set message and description based on status code if not provided
	if data.Message == "" {
		data.Message = getStatusMessage(data.Code)
	}
	if data.Description == "" {
		data.Description = getStatusDescription(data.Code)
	}

	// Render the template
	result := h.renderTemplate(h.template, data)

	// Post-process to remove empty table rows and leftover conditionals
	result = h.cleanupEmptyRows(result)

	return []byte(result), nil
}

// renderTemplate performs simple template rendering with conditionals
func (h *Handler) renderTemplate(template string, data *TemplateData) string {
	result := template

	// Handle conditional blocks first
	result = h.processConditionals(result, data)

	// Replace simple variables
	replacements := map[string]string{
		"{{ code }}":          strconv.Itoa(data.Code),
		"{{ message }}":       data.Message,
		"{{ description }}":   data.Description,
		"{{ message | escape }}": htmlEscape(data.Message),
		"{{ description | escape }}": htmlEscape(data.Description),
		"{{ host }}":          data.Host,
		"{{ original_uri }}":  data.OriginalURI,
		"{{ forwarded_for }}": data.ForwardedFor,
		"{{ request_id }}":    data.RequestID,
		"{{ nowUnix }}":       strconv.FormatInt(data.NowUnix, 10),
		"{{ l10nScript }}":    data.L10nScript,
	}

	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// processConditionals handles conditional blocks in the template
func (h *Handler) processConditionals(template string, data *TemplateData) string {
	result := template

	// Process complex conditional for auto-refresh first
	shouldAutoRefresh := data.Code == 408 || data.Code == 425 || data.Code == 429 ||
		data.Code == 500 || data.Code == 502 || data.Code == 503 || data.Code == 504
	result = h.processComplexRefreshConditional(result, shouldAutoRefresh)

	// Process {{ if show_details }} blocks
	result = h.processIfBlock(result, "show_details", data.ShowDetails)

	// Process {{ if l10n_enabled }} blocks
	result = h.processIfBlock(result, "l10n_enabled", data.L10nEnabled)

	return result
}

// processIfBlock handles simple {{ if condition }} ... {{ end }} blocks with nested conditionals
func (h *Handler) processIfBlock(template, condition string, show bool) string {
	result := template

	// Try different comment styles used in the template
	patterns := []struct {
		start string
		end   string
	}{
		{"<!-- {{- if " + condition + " -}} -->", "<!-- {{- end -}} -->"},
		{"<!-- {{ if " + condition + " }} -->", "<!-- {{ end }} -->"},
		{"<!-- {{- if " + condition + " }} -->", "<!-- {{ end }} -->"},
		{"<!-- {{ if " + condition + " -}} -->", "<!-- {{- end -}} -->"},
		{"<!-- {{if " + condition + "}} -->", "<!-- {{end}} -->"},
		{"<!-- {{- if " + condition + " -}}-->", "<!--{{- end -}}-->"},
	}

	for _, p := range patterns {
		startIdx := strings.Index(result, p.start)
		if startIdx == -1 {
			continue
		}

		// Find the matching end marker by counting nesting level
		searchPos := startIdx + len(p.start)
		nestLevel := 1
		endIdx := -1

		for searchPos < len(result) {
			// Check for nested if statements (but not the chained ones)
			nextIfIdx := strings.Index(result[searchPos:], "<!-- {{- if ")
			nextEndIdx := strings.Index(result[searchPos:], p.end)

			// If we find an end before another if (or no if found)
			if nextEndIdx != -1 && (nextIfIdx == -1 || nextEndIdx < nextIfIdx) {
				// Check if this is a chained conditional ({{- end }}{{ if)
				isChained := false
				if nextEndIdx > 0 {
					checkPos := searchPos + nextEndIdx
					if checkPos+len(p.end) < len(result) {
						afterEnd := result[checkPos+len(p.end) : min(checkPos+len(p.end)+10, len(result))]
						if strings.HasPrefix(afterEnd, "{{ if ") {
							isChained = true
						}
					}
				}

				if !isChained {
					nestLevel--
					if nestLevel == 0 {
						endIdx = searchPos + nextEndIdx
						break
					}
				}
				searchPos += nextEndIdx + len(p.end)
			} else if nextIfIdx != -1 {
				// Found a nested if
				nestLevel++
				searchPos += nextIfIdx + len("<!-- {{- if ")
			} else {
				// No more if or end markers found
				break
			}
		}

		if endIdx == -1 {
			continue
		}

		if show {
			// Remove the conditional markers but keep the content
			content := result[startIdx+len(p.start) : endIdx]
			result = result[:startIdx] + content + result[endIdx+len(p.end):]
		} else {
			// Remove the entire block
			result = result[:startIdx] + result[endIdx+len(p.end):]
		}

		// Process recursively in case there are multiple blocks
		return h.processIfBlock(result, condition, show)
	}

	return result
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// processComplexRefreshConditional handles the auto-refresh meta tag conditional
func (h *Handler) processComplexRefreshConditional(template string, show bool) string {
	result := template

	// Look for the refresh meta tag conditional
	start := "<!-- {{ if or (eq code 408) (eq code 425) (eq code 429) (eq code 500) (eq code 502) (eq code 503) (eq code 504) }} -->"
	end := "<!-- {{ end }} -->"

	startIdx := strings.Index(result, start)
	if startIdx == -1 {
		return result
	}

	endIdx := strings.Index(result[startIdx:], end)
	if endIdx == -1 {
		return result
	}

	endIdx += startIdx

	if show {
		// Remove the conditional markers but keep the content
		content := result[startIdx+len(start) : endIdx]
		result = result[:startIdx] + content + result[endIdx+len(end):]
	} else {
		// Remove the entire block
		result = result[:startIdx] + result[endIdx+len(end):]
	}

	return result
}

// cleanupEmptyRows removes leftover conditional comments and empty table rows
func (h *Handler) cleanupEmptyRows(html string) string {
	result := html

	// Remove all conditional comment markers
	result = strings.ReplaceAll(result, "<!-- {{- if show_details -}} -->", "")
	result = strings.ReplaceAll(result, "<!-- {{- if host -}} -->", "")
	result = strings.ReplaceAll(result, "<!-- {{- end }}{{ if original_uri -}} -->", "")
	result = strings.ReplaceAll(result, "<!-- {{- end }}{{ if forwarded_for -}} -->", "")
	result = strings.ReplaceAll(result, "<!-- {{- end }}{{ if request_id -}} -->", "")
	result = strings.ReplaceAll(result, "<!-- {{- end -}} -->", "")
	result = strings.ReplaceAll(result, "<!-- {{- if l10n_enabled -}} -->", "")

	// Remove table rows with empty values
	lines := strings.Split(result, "\n")
	var cleaned []string
	skipUntilEndTr := false

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Check if we're starting a table row
		if strings.Contains(line, "<tr>") {
			// Look ahead to see if this row has an empty value
			hasEmptyValue := false
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if strings.Contains(nextLine, `<td class="value"></td>`) {
					hasEmptyValue = true
					skipUntilEndTr = true
					break
				}
				if strings.Contains(nextLine, "</tr>") {
					break
				}
			}
			if hasEmptyValue {
				continue
			}
		}

		// If we're skipping an empty row, skip until we find </tr>
		if skipUntilEndTr {
			if strings.Contains(line, "</tr>") {
				skipUntilEndTr = false
			}
			continue
		}

		// Keep this line
		cleaned = append(cleaned, lines[i])
	}

	return strings.Join(cleaned, "\n")
}

// htmlEscape escapes HTML special characters
func htmlEscape(s string) string {
	replacements := []struct {
		old string
		new string
	}{
		{"&", "&amp;"},
		{"<", "&lt;"},
		{">", "&gt;"},
		{`"`, "&quot;"},
		{"'", "&#39;"},
	}

	result := s
	for _, r := range replacements {
		result = strings.ReplaceAll(result, r.old, r.new)
	}
	return result
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

	// Fallback for unknown codes
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

	// Generic descriptions
	if code >= 400 && code < 500 {
		return "An error occurred while processing your request."
	}
	return "The server encountered an error while processing your request."
}
