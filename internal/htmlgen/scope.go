package htmlgen

import (
	"regexp"
	"strings"
)

var idSelectorRe = regexp.MustCompile(`#([a-zA-Z0-9_\-]+)`)

// ScopeCSS parses css and scopes all rules so they only match elements
// inside scopeSelector (e.g. ".dht-ch-1"). Selectors targeting "body",
// "html", or ":root" are transformed to target scopeSelector directly.
// IDs listed in idRenames (old -> new) are rewritten in selectors.
// At-rules like @media and @supports are recursively scoped.
// @keyframes, @font-face, and @page rules are preserved without selector prefixing.
func ScopeCSS(css, scopeSelector string, idRenames map[string]string) string {
	css = stripComments(css)
	var sb strings.Builder
	p := &cssParser{input: css, scope: scopeSelector, idRenames: idRenames}
	p.parseRules(&sb)
	return strings.TrimSpace(sb.String())
}

type cssParser struct {
	input     string
	pos       int
	scope     string
	idRenames map[string]string
}

func (p *cssParser) skipWhitespace() {
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' {
			p.pos++
		} else {
			break
		}
	}
}

func (p *cssParser) skipString() {
	quote := p.input[p.pos]
	p.pos++ // consume opening quote
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == '\\' {
			p.pos += 2 // skip escaped character
			continue
		}
		if ch == quote {
			p.pos++ // consume closing quote
			return
		}
		p.pos++
	}
}

func (p *cssParser) parseRules(sb *strings.Builder) {
	for p.pos < len(p.input) {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}
		if p.input[p.pos] == '@' {
			p.parseAtRule(sb)
		} else {
			p.parseStyleRule(sb)
		}
	}
}

func (p *cssParser) parseAtRule(sb *strings.Builder) {
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != '{' && p.input[p.pos] != ';' {
		if p.input[p.pos] == '"' || p.input[p.pos] == '\'' {
			p.skipString()
		} else {
			p.pos++
		}
	}
	if p.pos >= len(p.input) {
		sb.WriteString(strings.TrimSpace(p.input[start:]))
		return
	}
	if p.input[p.pos] == ';' {
		p.pos++
		sb.WriteString(strings.TrimSpace(p.input[start:p.pos]))
		sb.WriteString("\n")
		return
	}

	header := strings.TrimSpace(p.input[start:p.pos])
	p.pos++ // consume '{'

	bodyStart := p.pos
	depth := 1
	for p.pos < len(p.input) && depth > 0 {
		ch := p.input[p.pos]
		if ch == '"' || ch == '\'' {
			p.skipString()
		} else if ch == '{' {
			depth++
			p.pos++
		} else if ch == '}' {
			depth--
			p.pos++
		} else {
			p.pos++
		}
	}
	bodyEnd := p.pos - 1
	if depth != 0 {
		bodyEnd = len(p.input)
	}
	body := p.input[bodyStart:bodyEnd]

	lowerHeader := strings.ToLower(header)
	if strings.HasPrefix(lowerHeader, "@media") || strings.HasPrefix(lowerHeader, "@supports") || strings.HasPrefix(lowerHeader, "@container") {
		inner := ScopeCSS(body, p.scope, p.idRenames)
		if strings.TrimSpace(inner) != "" {
			sb.WriteString(header)
			sb.WriteString(" {\n")
			sb.WriteString(inner)
			sb.WriteString("\n}\n")
		}
	} else {
		sb.WriteString(header)
		sb.WriteString(" {\n")
		sb.WriteString(strings.TrimSpace(body))
		sb.WriteString("\n}\n")
	}
}

func (p *cssParser) parseStyleRule(sb *strings.Builder) {
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != '{' && p.input[p.pos] != '}' && p.input[p.pos] != ';' {
		if p.input[p.pos] == '"' || p.input[p.pos] == '\'' {
			p.skipString()
		} else {
			p.pos++
		}
	}
	if p.pos >= len(p.input) || p.input[p.pos] != '{' {
		if p.pos < len(p.input) {
			p.pos++
		}
		return
	}
	selector := strings.TrimSpace(p.input[start:p.pos])
	p.pos++ // consume '{'

	bodyStart := p.pos
	depth := 1
	for p.pos < len(p.input) && depth > 0 {
		ch := p.input[p.pos]
		if ch == '"' || ch == '\'' {
			p.skipString()
		} else if ch == '{' {
			depth++
			p.pos++
		} else if ch == '}' {
			depth--
			p.pos++
		} else {
			p.pos++
		}
	}
	bodyEnd := p.pos - 1
	if depth != 0 {
		bodyEnd = len(p.input)
	}
	body := strings.TrimSpace(p.input[bodyStart:bodyEnd])
	if selector == "" || body == "" {
		return
	}

	scopedSelector := scopeSelectorList(selector, p.scope, p.idRenames)
	if scopedSelector != "" {
		sb.WriteString(scopedSelector)
		sb.WriteString(" { ")
		sb.WriteString(body)
		sb.WriteString(" }\n")
	}
}

func scopeSelectorList(selectorList, scope string, idRenames map[string]string) string {
	parts := splitSelectorList(selectorList)
	var scopedParts []string
	seen := make(map[string]bool)
	for _, sel := range parts {
		s := strings.TrimSpace(sel)
		if s == "" {
			continue
		}
		if len(idRenames) > 0 {
			s = renameIDsInSelector(s, idRenames)
		}
		scoped := scopeSingleSelector(s, scope)
		if scoped != "" && !seen[scoped] {
			seen[scoped] = true
			scopedParts = append(scopedParts, scoped)
		}
	}
	return strings.Join(scopedParts, ", ")
}

func scopeSingleSelector(s, scope string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if lower == "body" || lower == "html" || lower == ":root" {
		return scope
	}
	if strings.HasPrefix(lower, "body.") || strings.HasPrefix(lower, "body#") ||
		strings.HasPrefix(lower, "body[") || strings.HasPrefix(lower, "body:") {
		return scope + s[4:]
	}
	if strings.HasPrefix(lower, "html.") || strings.HasPrefix(lower, "html#") ||
		strings.HasPrefix(lower, "html[") || strings.HasPrefix(lower, "html:") {
		return scope + s[4:]
	}
	if strings.HasPrefix(lower, "body ") || strings.HasPrefix(lower, "body\t") ||
		strings.HasPrefix(lower, "body\n") || strings.HasPrefix(lower, "body\r") {
		return scope + " " + strings.TrimSpace(s[4:])
	}
	if strings.HasPrefix(lower, "html ") || strings.HasPrefix(lower, "html\t") ||
		strings.HasPrefix(lower, "html\n") || strings.HasPrefix(lower, "html\r") {
		return scope + " " + strings.TrimSpace(s[4:])
	}
	if strings.HasPrefix(lower, "body>") || strings.HasPrefix(lower, "body+") || strings.HasPrefix(lower, "body~") {
		return scope + " " + strings.TrimSpace(s[4:])
	}
	if strings.HasPrefix(lower, "html>") || strings.HasPrefix(lower, "html+") || strings.HasPrefix(lower, "html~") {
		return scope + " " + strings.TrimSpace(s[4:])
	}
	return scope + " " + s
}

func splitSelectorList(s string) []string {
	var parts []string
	start := 0
	depthParen := 0
	depthBracket := 0
	inQuote := byte(0)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inQuote != 0 {
			if ch == inQuote && (i == 0 || s[i-1] != '\\') {
				inQuote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			inQuote = ch
			continue
		}
		if ch == '(' {
			depthParen++
		} else if ch == ')' && depthParen > 0 {
			depthParen--
		} else if ch == '[' {
			depthBracket++
		} else if ch == ']' && depthBracket > 0 {
			depthBracket--
		} else if ch == ',' && depthParen == 0 && depthBracket == 0 {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func stripComments(css string) string {
	var sb strings.Builder
	inQuote := byte(0)
	i := 0
	for i < len(css) {
		if inQuote != 0 {
			sb.WriteByte(css[i])
			if css[i] == inQuote && (i == 0 || css[i-1] != '\\') {
				inQuote = 0
			}
			i++
			continue
		}
		if css[i] == '"' || css[i] == '\'' {
			inQuote = css[i]
			sb.WriteByte(css[i])
			i++
			continue
		}
		if i+1 < len(css) && css[i] == '/' && css[i+1] == '*' {
			end := strings.Index(css[i+2:], "*/")
			if end >= 0 {
				i += 2 + end + 2
			} else {
				break
			}
			continue
		}
		sb.WriteByte(css[i])
		i++
	}
	return sb.String()
}

func renameIDsInSelector(s string, idRenames map[string]string) string {
	return idSelectorRe.ReplaceAllStringFunc(s, func(m string) string {
		id := m[1:]
		if target, ok := idRenames[id]; ok && target != id {
			return "#" + target
		}
		return m
	})
}
