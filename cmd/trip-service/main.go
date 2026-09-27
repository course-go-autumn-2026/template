package main

import (
	"context"
	"log"
	"os"

	"github.com/imrtlDrifter/template/internal/app"
)

func main() {
	if err := app.Run(context.Background()); err != nil {
		log.Printf("application stopped: %v", err)
		os.Exit(1)
	}
}
