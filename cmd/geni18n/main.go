// Command geni18n writes web/src/i18n/data.gen.ts from the rule data texts (German) and
// internal/data/i18n/en.json (English). Run it via `just gen-i18n` after changing a text in
// internal/data.
package main

import (
	"flag"
	"fmt"
	"os"

	"ruderspiel/internal/data"
)

func main() {
	out := flag.String("out", "web/src/i18n/data.gen.ts", "output file")
	flag.Parse()

	src, err := data.MustLoad().GenerateTS()
	if err == nil {
		err = os.WriteFile(*out, src, 0o644) //nolint:gosec // generated source file, world-readable is fine
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "geni18n:", err)
		os.Exit(1)
	}
}
