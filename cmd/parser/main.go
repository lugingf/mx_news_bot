package main

import (
	"context"
	"log/slog"
	"mx_news_bot/config"
	prsr "mx_news_bot/internal/parser"
	"mx_news_bot/internal/storage"
	"os"
)

const (
	dataDir    = "./data/2025"
	outputFile = "output/result.txt"
)

func main() {
	cfg, err := config.New(context.Background())
	if err != nil {
		slog.Error("Config initialization failed", "error", err)
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbm, err := config.OpenSQLXConn(cfg.DB)
	if err != nil {
		logger.Error("DB Connection Failed", "error", err)
		return
	}

	repository := storage.New(dbm, logger)

	pconf := prsr.Config{
		DryRun:     false,
		DataDir:    dataDir,
		OutputFile: outputFile,
	}

	parser := prsr.New(repository, &pconf, logger)

	files, err := parser.CollectFiles([]string{})
	if err != nil {
		logger.Error("Files collect error: %v", err)
		return
	}
	logger.Info("Files collected", "files", files)

	for _, pdfFile := range files {
		parser.ParseFile(pdfFile)
	}

	logger.Info("All files parsed successfully")
}
