package main

import (
	"os"

	"github.com/soleda2026/ScanFB/internal/bridge"
)

func main() {
	os.Exit(bridge.Serve(os.Stdin, os.Stdout, os.Stderr))
}
