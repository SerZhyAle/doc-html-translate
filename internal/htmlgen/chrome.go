package htmlgen

import (
	"strings"

	gohtml "golang.org/x/net/html"
)

// chromeRootClasses are the classes of the elements that hold the reader chrome this package
// writes into a page: the chapter and single-page bar (buildNavBarHTML, buildSinglePageHeader)
// and the index toolbar. Everything the reader sees of the app, not of the book, sits inside one.
var chromeRootClasses = []string{"dht-navbar", "dht-toolbar"}

// IsReaderChrome reports whether n is the root of injected reader chrome. The translation step
// leaves such a subtree out: the file name, the page count and the interface words are not the
// book's text, and a paid engine must not be sent them or bill for them.
func IsReaderChrome(n *gohtml.Node) bool {
	if n.Type != gohtml.ElementNode {
		return false
	}
	for _, a := range n.Attr {
		if a.Namespace != "" || a.Key != "class" {
			continue
		}
		for _, c := range strings.Fields(a.Val) {
			for _, root := range chromeRootClasses {
				if c == root {
					return true
				}
			}
		}
	}
	return false
}
