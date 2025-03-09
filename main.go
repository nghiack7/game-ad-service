package main

import (
	"os"

	"github.com/nghiack7/game-ad-service/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
