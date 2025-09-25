package main

import (
	"context"
	"log"
	"os"

	"github.com/charmbracelet/fang"
)

func main() {
	rootCmd := NewRootCmd()

	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithNotifySignal(os.Interrupt, os.Kill),
	); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
