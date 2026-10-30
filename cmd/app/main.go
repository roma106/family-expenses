package main

import (
	"family-expenses/internal/app"
	"log/slog"
	"os"
)

func main() {
	a, err := app.New()
	if err != nil {
		slog.Error("Failed to create app: ", "error", err)
		os.Exit(1)
	}
	if err := a.Run(); err != nil {
		slog.Error("Failed to run app: ", "error", err)
		os.Exit(1)
	}
}
