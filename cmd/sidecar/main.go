package main

import (
	"fmt"
	"os"

	"github.com/w6xian/keeper/service"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	if err := service.Run(server_name, func() {
		_ = rootCmd.Execute()
	}); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
