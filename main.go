// Command valsenv renders a dotenv stream, resolving vals ref+ references.
package main

import (
	"os"

	"github.com/zoispag/valsenv/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
