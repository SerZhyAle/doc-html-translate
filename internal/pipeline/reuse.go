package pipeline

import (
	"strings"

	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/outputpath"
)

// rebuildMessage tells the reader why an existing output is converted again instead of being
// opened. Without it a rebuild after an option change looks like the shortcut broke.
func rebuildMessage(reason outputpath.Reason, changed []string) string {
	switch reason {
	case outputpath.ReuseSourceChanged:
		return i18n.S("The document changed since it was converted - rebuilding the output.")
	case outputpath.ReuseOptionsChanged:
		return i18n.S("The existing output was made with different settings (%s) - rebuilding it.", strings.Join(changed, ", "))
	case outputpath.ReusePartial:
		return i18n.S("The existing output is only partially translated - rebuilding it.")
	case outputpath.ReuseUntranslated:
		return i18n.S("The existing output was never translated - rebuilding it.")
	default:
		return i18n.S("The existing output is incomplete (interrupted, or made by an older version) - rebuilding it.")
	}
}
