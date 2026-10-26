//go:build !android

package main

import (
	"fmt"
	"os"
)

// main, anywhere but Android, says how to build the program for the headset.
func main() {
	fmt.Fprintln(os.Stderr, "this program runs on a Meta Quest: build its package with tools/build-apk.sh ./cmd/explorer")
	os.Exit(2)
}
