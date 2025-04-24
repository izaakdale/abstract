package main

import (
	"log/slog"
	"os"

	"github.com/izaakdale/abstract/internal/server/app"
)

func main() {
	if err := app.Run(); err != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{}))
		logger.Error("error running app", "error", err)
		os.Exit(1)
	}
}
