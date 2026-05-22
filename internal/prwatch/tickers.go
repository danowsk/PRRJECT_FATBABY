package prwatch

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/example/prrject-fatbaby/internal/identity"
)

const MaxScanBytes = 32 * 1024

var tickerRegex = regexp.MustCompile(
	`(?i)\(?\s*` +
		`(NYSE(?:\s+American)?|NASDAQ|OTC(?:QX|QB)?|AMEX|BATS|TSX|CBOE|LSE|HKEX)` +
		`\s*:\s*` +
		`([A-Z][A-Z0-9\.\-]{0,9})` +
		`\s*\)?`,
)

func ExtractTickers(text string) []identity.SecurityRef {
	if text == "" {
		return nil
	}
	all := tickerRegex.FindAllStringSubmatch(text, -1)
	if len(all) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(all))
	out := make([]identity.SecurityRef, 0, len(all))
	for _, m := range all {
		if len(m) < 3 {
			continue
		}
		ex := normalizeExchange(m[1])
		sym := strings.ToUpper(strings.TrimSpace(m[2]))
		key := ex + ":" + sym
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, identity.SecurityRef{Exchange: ex, Symbol: sym, Source: "regex", Confidence: 0.91, MatchedText: m[0]})
	}
	return out
}

func ExtractFromHTML(htmlContent []byte) []identity.SecurityRef {
	if len(htmlContent) == 0 {
		return nil
	}
	metaText, bodyText := tokenizeHTML(htmlContent)
	for _, s := range metaText {
		if refs := extractWithPrecheck([]byte(s)); len(refs) > 0 {
			return refs
		}
	}
	if len(bodyText) > MaxScanBytes {
		bodyText = bodyText[:MaxScanBytes]
	}
	if refs := extractWithPrecheck(bodyText); len(refs) > 0 {
		return refs
	}
	return extractWithPrecheck(htmlContent)
}

func extractWithPrecheck(body []byte) []identity.SecurityRef {
	u := bytes.ToUpper(body)
	if !bytes.Contains(u, []byte("NASDAQ")) &&
		!bytes.Contains(u, []byte("NYSE")) &&
		!bytes.Contains(u, []byte("OTC")) {
		return nil
	}
	return ExtractTickers(string(body))
}

var metaTagRe = regexp.MustCompile(`(?is)<meta\s+[^>]*>`)
var attrRe = regexp.MustCompile(`(?i)(name|property|itemprop|content)\s*=\s*["']([^"']+)["']`)
var stripScriptRe = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
var stripStyleRe = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
var stripNoScriptRe = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`)
var tagRe = regexp.MustCompile(`(?is)<[^>]+>`)

func tokenizeHTML(doc []byte) ([]string, []byte) {
	s := string(doc)
	meta := make([]string, 0, 4)
	for _, m := range metaTagRe.FindAllString(s, -1) {
		vals := map[string]string{}
		for _, a := range attrRe.FindAllStringSubmatch(m, -1) {
			vals[strings.ToLower(a[1])] = strings.TrimSpace(a[2])
		}
		if c := vals["content"]; c != "" {
			k := strings.ToLower(vals["name"])
			p := strings.ToLower(vals["property"])
			i := strings.ToLower(vals["itemprop"])
			if k == "description" || p == "og:description" || p == "twitter:description" || i == "description" {
				meta = append(meta, c)
			}
		}
	}
	lower := strings.ToLower(s)
	bi := strings.Index(lower, "<body")
	if bi < 0 {
		bi = 0
	}
	body := s[bi:]
	body = stripScriptRe.ReplaceAllString(body, " ")
	body = stripStyleRe.ReplaceAllString(body, " ")
	body = stripNoScriptRe.ReplaceAllString(body, " ")
	body = tagRe.ReplaceAllString(body, " ")
	return meta, []byte(body)
}

func normalizeExchange(v string) string {
	v = strings.Join(strings.Fields(strings.ToUpper(v)), " ")
	if v == "NYSE AMERICAN" {
		return "NYSE American"
	}
	return v
}
