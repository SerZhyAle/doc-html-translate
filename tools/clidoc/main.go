// Command clidoc is the command-line reference of the product and its gate (APP-COMMANDS rules 18 and 20; ticket E0143):
// docs/cli-commands.json is the manifest, internal/commands/cli-commands.json is the executable's own copy of it, and
// docs/CLI.md is rendered from it. The check holds the three equal and the manifest valid. It ends in one CHECK-VERDICT
// line.
//
//	go run ./tools/clidoc            the check, from the repository root (scripts/cli-doc.ps1 runs it)
//	go run ./tools/clidoc -render    copy the manifest into internal/commands and rewrite docs/CLI.md, then stop
//
// Exit 0 PASS, 1 FAIL, 2 COULD NOT VERIFY.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"doc-html-translate/internal/commands"
)

const (
	manifestPath = "docs/cli-commands.json"
	embeddedPath = "internal/commands/cli-commands.json"
	docPath      = "docs/CLI.md"
)

func main() {
	render := flag.Bool("render", false, "rewrite "+embeddedPath+" and "+docPath+" from "+manifestPath)
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		verdict(2, "no working directory")
	}
	source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(manifestPath)))
	if err != nil {
		fmt.Println("cli-doc:", manifestPath, "cannot be read:", err)
		verdict(2, "no "+manifestPath)
	}

	if *render {
		if problems := commands.Check(source); len(problems) > 0 {
			for _, p := range problems {
				fmt.Println("FAIL:", p)
			}
			verdict(1, "the manifest is not valid, nothing was written")
		}
		m, err := commands.ParseManifest(source)
		if err != nil {
			fmt.Println("cli-doc:", err)
			verdict(1, "the manifest does not parse")
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(embeddedPath)), source, 0o644); err != nil {
			fmt.Println("cli-doc:", err)
			verdict(2, "the embedded copy was not written")
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(docPath)), []byte(commands.Markdown(m)), 0o644); err != nil {
			fmt.Println("cli-doc:", err)
			verdict(2, "the reference was not written")
		}
		fmt.Println("cli-doc:", embeddedPath, "and", docPath, "written")
		verdict(0, "rendered")
	}

	fmt.Printf("cli-doc: subject = %s against its embedded copy and %s in %s\n", manifestPath, docPath, root)
	var fails []string
	for _, p := range commands.Check(source) {
		fails = append(fails, manifestPath+": "+p)
	}

	copyBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(embeddedPath)))
	switch {
	case err != nil:
		fails = append(fails, embeddedPath+" cannot be read: "+err.Error())
	case !bytes.Equal(copyBytes, source):
		fails = append(fails, embeddedPath+" differs from "+manifestPath+" (run go run ./tools/clidoc -render; never edit the copy)")
	}

	if m, err := commands.ParseManifest(source); err == nil {
		got, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(docPath)))
		switch {
		case readErr != nil:
			fails = append(fails, docPath+" cannot be read: "+readErr.Error()+" (run go run ./tools/clidoc -render)")
		case string(got) != commands.Markdown(m):
			fails = append(fails, docPath+" is not the render of "+manifestPath+" (run go run ./tools/clidoc -render; never edit the page by hand)")
		}
	} else {
		fails = append(fails, manifestPath+": "+err.Error())
	}

	for _, f := range fails {
		fmt.Println("FAIL:", f)
	}
	if len(fails) > 0 {
		verdict(1, fmt.Sprintf("%d finding(s)", len(fails)))
	}
	verdict(0, "")
}

func verdict(code int, detail string) {
	word := map[int]string{0: "PASS", 1: "FAIL", 2: "COULD NOT VERIFY"}[code]
	if detail == "" {
		fmt.Printf("cli-doc: %s\n", word)
	} else {
		fmt.Printf("cli-doc: %s (%s)\n", word, detail)
	}
	os.Exit(code)
}
