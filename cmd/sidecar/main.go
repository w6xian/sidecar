package main

import (
	"context"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	rootCommand(context.Background(), "sidecar").Execute()
}
