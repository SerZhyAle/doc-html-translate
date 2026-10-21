// Command settingsinv is the settings inventory of the GUI launcher and its gate (APP-SETTINGS rule 11; ticket 122,
// item 4): docs/settings-inventory.json is the render of what cmd/doc-html-ui/ui.html persists, and the check holds it
// equal to the window, to the help map, to the help copy deck and to the README flag tables. It ends in one
// CHECK-VERDICT line.
//
//	go run ./tools/settingsinv            the check, from the repository root (scripts/settings-inventory.ps1 runs it)
//	go run ./tools/settingsinv -render    rewrite docs/settings-inventory.json from the window, then stop
//
// Exit 0 PASS, 1 FAIL, 2 COULD NOT VERIFY.
package main

import (
	"flag"
	"fmt"
	"os"

	"doc-html-translate/internal/settingsinv"
)

func main() {
	render := flag.Bool("render", false, "rewrite "+settingsinv.InventoryPath+" from the window")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		verdict(2, "no working directory")
	}
	if *render {
		r, err := settingsinv.Render(root)
		if err != nil {
			fmt.Println("settings-inventory:", err)
			verdict(2, "the inventory was not written")
		}
		for _, f := range r.Failures {
			fmt.Println("FAIL:", f)
		}
		if len(r.Failures) > 0 {
			verdict(1, "the window does not trace to an inventory")
		}
		fmt.Println("settings-inventory:", settingsinv.InventoryPath, "written")
		verdict(0, "rendered")
	}
	fmt.Printf("settings-inventory: subject = %s against the window, the help map, the help copy deck and the README flag tables in %s\n", settingsinv.InventoryPath, root)
	r, err := settingsinv.Run(root)
	if err != nil {
		fmt.Println("settings-inventory:", err)
		verdict(2, "the inputs cannot be read")
	}
	for _, f := range r.Failures {
		fmt.Println("FAIL:", f)
	}
	verdict(r.Code(), r.Detail())
}

func verdict(code int, detail string) {
	word := map[int]string{0: "PASS", 1: "FAIL", 2: "COULD NOT VERIFY"}[code]
	fmt.Printf("settings-inventory: %s (%s)\n", word, detail)
	os.Exit(code)
}
