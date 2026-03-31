package main

import (
	"context"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	daemonCommand(context.Background(), "server").Execute()
}
