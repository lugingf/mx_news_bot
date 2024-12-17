package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"

	"mx_news_bot/config"
	"mx_news_bot/internal/models"
	"mx_news_bot/pkg/yandexai"
)

func main() {
	pdfFile := "data/2024/S2485/450SX/S1F1RES.pdf"
	textFile := "output/out.txt"

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.New(ctx)
	if err != nil {
		slog.Error("Config initialization failed", "error", err)
		return
	}

	c := cfg.Yandex
	checkChan := make(chan yandexai.PendingRequests)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	aiClient := yandexai.NewAPIClient(c.BaseURL, c.ApiKey, c.FolderID, checkChan, logger)

	resChan := make(chan string)
	checker := yandexai.NewChecker(aiClient, checkChan, resChan, logger)

	checker.Start(ctx)

	err = convertPDFToText(pdfFile, textFile)
	if err != nil {
		fmt.Println("Ошибка при конвертации PDF в текст:", err)
		return
	}

	data, err := os.ReadFile(textFile)
	if err != nil {
		fmt.Println("Ошибка при открытии текстового файла:", err)
		return
	}

	text := cleanRequest(string(data))

	if err = aiClient.MakeRequest(text); err != nil {
		fmt.Println("Ошибка при запросе в AI:", err)
		return
	}

	var parsed string

	select {
	case <-ctx.Done():
		return
	case parsed = <-resChan:
	}

	parsed = cleanResponse(parsed)

	logger.Info("TEXT lower ======================")
	logger.Info(parsed)

	var riders models.RaceResult

	err = json.Unmarshal([]byte(parsed), &riders)
	if err != nil {
		logger.Error("Unmarshal error", "error", err)
	}

	fmt.Println(fmt.Sprintf("%v", riders))
}

func cleanRequest(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		if !strings.Contains(line, "<a href=") {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func cleanResponse(response string) string {
	re := regexp.MustCompile("```|\\n|\\r|")
	cleaned := re.ReplaceAllString(response, "")
	escaped := strings.ReplaceAll(cleaned, `\"`, `"`)
	return strings.TrimSpace(escaped)
}

func convertPDFToText(pdfFile, textFile string) error {
	fmt.Printf("Converting PDF %s to text\n", pdfFile)
	cmd := exec.Command("pdftotext", "-f", "1", "-l", "1", "-layout", pdfFile, textFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
