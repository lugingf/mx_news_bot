package main

import (
	"context"
	"log/slog"
	"mx_news_bot/config"
	"mx_news_bot/internal/models"
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

	eventNames := []string{"Anaheim 1", "San Diego", "Anaheim 2"}
	files, err := parser.CollectFiles(eventNames)
	if err != nil {
		logger.Error("Files collect error: %v", err)
		return
	}
	logger.Info("Files collected", "files", files)

	for _, pdfFile := range files {
		result, err := parser.ParseFile(pdfFile, models.EventToCheck{RoundNumber: ""})
		if err != nil {
			logger.Error("Can't parse file", "file", pdfFile)
			continue
		}

		err = parser.UploadRaceResult(result)
		if err != nil {
			logger.Error("Can't upload race result", "file", pdfFile)
			continue
		}
	}

	logger.Info("Process finished")
}
