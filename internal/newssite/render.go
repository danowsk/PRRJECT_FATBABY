package newssite

import (
	"fmt"
	"html"
	"io"
	"strings"
	"time"
)

// RenderListPage writes the source document listing page HTML.
func RenderListPage(w io.Writer, entries []DocEntry) {
	writePageStart(w, "Source Documents")
	fmt.Fprint(w, "<h1>Source Documents</h1>")
	if len(entries) == 0 {
		fmt.Fprint(w, "<p>No source documents have been persisted yet. The processor has not run or no filings have been discovered.</p>")
		writePageEnd(w)
		return
	}
	for _, entry := range entries {
		title := html.EscapeString(entry.Ticker)
		id := html.EscapeString(entry.Identity)
		sourceLabel := html.EscapeString(sourceTypeLabel(entry.SourceType))
		form := html.EscapeString(entry.Form)
		preview := html.EscapeString(entry.BodyPreview)
		timeText := html.EscapeString(formatTimeUTC(entry.PersistedAt))

		fmt.Fprint(w, "<article>")
		fmt.Fprintf(w, "<h2><a href=\"/doc/%s\">%s</a></h2>", id, title)
		fmt.Fprint(w, "<p class=\"meta\" style=\"font-family: system-ui, sans-serif; font-size: 0.85rem; color: #555\">")
		fmt.Fprintf(w, "%s", sourceLabel)
		if form != "" {
			fmt.Fprintf(w, " · %s", form)
		}
		fmt.Fprintf(w, " · %s", timeText)
		fmt.Fprint(w, "</p>")
		fmt.Fprintf(w, "<p>%s…</p>", preview)
		fmt.Fprintf(w, "<p><a href=\"/doc/%s\">Read full document →</a></p>", id)
		fmt.Fprint(w, "</article><hr>")
	}
	writePageEnd(w)
}

// RenderDetailPage writes a source document detail page HTML.
func RenderDetailPage(w io.Writer, entry DocEntry) {
	sourceLabel := html.EscapeString(sourceTypeLabel(entry.SourceType))
	formOrType := sourceLabel
	if entry.Form != "" {
		formOrType = html.EscapeString(entry.Form)
	}
	title := fmt.Sprintf("%s — %s", entry.Ticker, formOrType)
	writePageStart(w, title)

	fmt.Fprint(w, "<nav><a href=\"/\">← All documents</a></nav>")
	fmt.Fprintf(w, "<h1>%s</h1>", html.EscapeString(title))
	fmt.Fprint(w, "<p style=\"font-family: system-ui, sans-serif; font-size: 0.85rem; color: #555\">")
	fmt.Fprintf(w, "%s", sourceLabel)
	if entry.Form != "" {
		fmt.Fprintf(w, " · %s", html.EscapeString(entry.Form))
	}
	fmt.Fprintf(w, " · %s · %d characters", html.EscapeString(formatTimeUTC(entry.PersistedAt)), entry.CharCount)
	fmt.Fprint(w, "</p>")
	if isSafeLink(entry.DocumentURL) {
		esc := html.EscapeString(entry.DocumentURL)
		fmt.Fprintf(w, "<p><a href=\"%s\">%s</a></p>", esc, esc)
	} else {
		fmt.Fprintf(w, "<p>%s</p>", html.EscapeString(entry.DocumentURL))
	}
	fmt.Fprint(w, "<hr>")
	fmt.Fprintf(w, "<pre style=\"white-space: pre-wrap; word-break: break-word; font-family: Georgia, serif; font-size: 0.95rem; line-height: 1.7\">%s</pre>", html.EscapeString(entry.FullText))
	writePageEnd(w)
}

func writePageStart(w io.Writer, title string) {
	fmt.Fprint(w, "<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	fmt.Fprintf(w, "<title>%s</title>", html.EscapeString(title))
	fmt.Fprint(w, "<style>body{font-family:Georgia,'Times New Roman',serif;font-size:1rem;line-height:1.7;color:#111;background:#fafafa;margin:0;}main{max-width:680px;margin:0 auto;padding:1.5rem 1rem;}h1,h2,h3{font-family:system-ui,sans-serif;line-height:1.3;}h1{font-size:1.75rem;}h2{font-size:1.25rem;}h3{font-size:1rem;}a{color:#1a1a8c;}a:visited{color:#551a8b;}hr{border:none;border-top:1px solid #e0e0e0;margin:2rem 0;}</style></head><body><main>")
}

func writePageEnd(w io.Writer) {
	fmt.Fprint(w, "</main></body></html>")
}

func sourceTypeLabel(sourceType string) string {
	switch sourceType {
	case "sec_8k":
		return "SEC Filing"
	case "press_release":
		return "Press Release"
	default:
		replaced := strings.ReplaceAll(sourceType, "_", " ")
		if replaced == "" {
			return "Unknown"
		}
		return strings.Title(replaced)
	}
}

func formatTimeUTC(t time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}
	return t.UTC().Format("2 Jan 2006 15:04 UTC")
}

func isSafeLink(url string) bool {
	return strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")
}
