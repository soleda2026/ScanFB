package main

import (
	"os"

	"github.com/soleda2026/ScanFB/internal/bridge"
)

func main() {
	os.Exit(bridge.ServeReadiness(os.Stdin, os.Stdout, os.Stderr))
}
