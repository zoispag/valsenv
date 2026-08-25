package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "valsenv: usage: valsenv render [-f file] [-o file]")
	os.Exit(2)
}
