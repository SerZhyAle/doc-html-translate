package htmlgen

import "encoding/json"

// jsString renders s as a JavaScript string literal that is also safe inside an
// inline <script>: encoding/json escapes <, > and & (so a value holding
// "</script>" cannot end the element) and U+2028/U+2029, which Go's %q leaves
// raw and older JavaScript engines read as line breaks. Book titles, file names
// and hrefs reach generated scripts through this and nothing else.
func jsString(s string) string {
	b, _ := json.Marshal(s) // a string always marshals
	return string(b)
}
