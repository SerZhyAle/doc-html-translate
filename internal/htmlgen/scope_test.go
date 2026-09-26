package htmlgen

import (
	"strings"
	"testing"
)

func TestScopeCSS(t *testing.T) {
	tests := []struct {
		name      string
		css       string
		scope     string
		idRenames map[string]string
		want      []string
	}{
		{
			name:  "element and class selectors",
			css:   `p { margin: 1em; } .pdf-images img.pdf-flip-y { transform: scaleY(-1); }`,
			scope: ".dht-ch-1",
			want: []string{
				".dht-ch-1 p { margin: 1em; }",
				".dht-ch-1 .pdf-images img.pdf-flip-y { transform: scaleY(-1); }",
			},
		},
		{
			name:  "body and html selectors",
			css:   `body { font-family: serif; } html { height: 100%; } body.dark { background: #000; } body > p { color: red; }`,
			scope: ".dht-ch-1",
			want: []string{
				".dht-ch-1 { font-family: serif; }",
				".dht-ch-1 { height: 100%; }",
				".dht-ch-1.dark { background: #000; }",
				".dht-ch-1 > p { color: red; }",
			},
		},
		{
			name:  "comma separated selectors",
			css:   `h1, h2, h3 { margin-top: 1.5em; }`,
			scope: ".dht-ch-2",
			want: []string{
				".dht-ch-2 h1, .dht-ch-2 h2, .dht-ch-2 h3 { margin-top: 1.5em; }",
			},
		},
		{
			name:  "media query",
			css:   `@media (max-width: 700px) { .pdf-images { float: none; width: 100%; } p { font-size: 12px; } }`,
			scope: ".dht-ch-1",
			want: []string{
				"@media (max-width: 700px)",
				".dht-ch-1 .pdf-images { float: none; width: 100%; }",
				".dht-ch-1 p { font-size: 12px; }",
			},
		},
		{
			name:  "keyframes and font-face kept unscoped",
			css:   `@font-face { font-family: 'Custom'; src: url('f.woff2'); } @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }`,
			scope: ".dht-ch-1",
			want: []string{
				"@font-face",
				"font-family: 'Custom'",
				"@keyframes fadeIn",
			},
		},
		{
			name:      "renamed IDs updated",
			css:       `#top { color: red; } p a[href="#top"] { text-decoration: none; }`,
			scope:     ".dht-ch-2",
			idRenames: map[string]string{"top": "c2-top"},
			want: []string{
				".dht-ch-2 #c2-top { color: red; }",
				`.dht-ch-2 p a[href="#c2-top"] { text-decoration: none; }`,
			},
		},
		{
			name:  "comments stripped",
			css:   `/* comment */ p { color: blue; /* inner */ }`,
			scope: ".dht-ch-1",
			want: []string{
				".dht-ch-1 p { color: blue; }",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ScopeCSS(tc.css, tc.scope, tc.idRenames)
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("ScopeCSS output missing %q:\nGot:\n%s", w, got)
				}
			}
		})
	}
}
