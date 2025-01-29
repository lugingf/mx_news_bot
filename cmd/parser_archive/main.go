package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"
	"log/slog"
	"mx_news_bot/internal/pdfconverter"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mx_news_bot/config"
	"mx_news_bot/internal/models"
	"mx_news_bot/internal/storage"
)

const (
	codeClass450SX = "S1"
	codeClass250SX = "S2"

	codeRaceMainEvent  = "F1"
	codeRaceQual       = "Q%d"
	codeRaceLastChance = "L1"
	codeRaceHeat       = "H%d"

	codeResult               = "RES"
	codeIndivSeg             = "IND"
	codeIndivLap             = "RID"
	codeLapChart             = "LAP"
	codeLineUp               = "LINEUP"
	codeBestLapTimes         = "OVR"
	codeBestLapTimesCombined = "COVR"
)

const (
	championshipSX    = "Monster Energy AMA Supercross"
	championshipProMX = "Pro Motocross Championship"
)

const raceTypeMain = "MainEvent"

const (
	dataDir    = "./data"
	outputFile = "output/result.txt"
)

// Main events results
var fileSet = []string{"Anaheim #1_250_Main_Event.pdf", "Anaheim #1_450_Main_Event.pdf"}

func main() {
	cfg := config.New("config.json")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbm, err := config.OpenSQLXConn(cfg.DB)
	if err != nil {
		logger.Error("DB Connection Failed", "error", err)
		return
	}

	repository := storage.New(dbm, logger)

	files, err := collectFiles(dataDir, fileSet)
	logger.Info("Files collected", "files", files)
	if err != nil {
		log.Fatal(err)
	}

	for _, pdfFile := range files {
		err = pdfconverter.ConvertPDFToText(pdfFile, outputFile)
		if err != nil {
			logger.Error("Ошибка при конвертации PDF в текст: %v", err)
			return
		}

		file, err := os.Open(outputFile)
		if err != nil {
			if file != nil {
				file.Close()
			}

			logger.Error("Ошибка при открытии файла текста: %v", err)
			return
		}

		raceResult, err := getRaceResult(pdfFile, file)
		if err != nil {
			file.Close()
			logger.Error("can't get race result", "error", err, "file", pdfFile)
			return
		}
		// debug
		outputJSON(raceResult)

		err = repository.UploadRaceResultsSMX(raceResult)
		if err != nil {
			logger.Error("Cant upload race results from file", "file", pdfFile, "error", err)
		}

		file.Close()
	}

}

func getRaceResult(pdfFile string, file io.Reader) (models.RaceResult, error) {
	var raceResult models.RaceResult
	if strings.Contains(pdfFile, codeRaceMainEvent) {
		raceResult.RaceType = raceTypeMain
	}
	if strings.Contains(pdfFile, codeClass450SX) || strings.Contains(pdfFile, codeClass250SX) {
		raceResult.ChampName = championshipSX
	}

	scanner := bufio.NewScanner(file)
	err := parseText(scanner, &raceResult)
	if err != nil {
		return models.RaceResult{}, errors.Wrapf(err, "parse race result %s", pdfFile)
	}
	raceResult.EventCode, err = getEventCode(raceResult.Date, raceResult.ChampName, raceResult.Round)
	if err != nil {
		return models.RaceResult{}, errors.Wrapf(err, "cant get eventCode %s", pdfFile)
	}

	return raceResult, nil
}

func getEventCode(date time.Time, name, roundNum string) (string, error) {
	var prefix string
	switch name {
	case championshipSX:
		prefix = "S"
	case championshipProMX:
		prefix = "M"
	default:
		return "", errors.Errorf("unknown champ type %s", name)
	}

	lastTwoDigits := date.Year() % 100

	roundInt, err := strconv.Atoi(roundNum)
	if err != nil {
		return "", errors.Wrapf(err, "parse round number %s", roundNum)
	}
	rN := roundInt * 5
	if rN > 85 {
		rN = 99
	}

	return fmt.Sprintf("%s%d%02d", prefix, lastTwoDigits, rN), nil
}

func collectFiles(baseDir string, fileNames []string) ([]string, error) {
	var filePaths []string

	fileNameMap := make(map[string]struct{})
	for _, name := range fileNames {
		fileNameMap[name] = struct{}{}
	}

	// Проходим по всем поддиректориям
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if _, exists := fileNameMap[info.Name()]; exists {
				filePaths = append(filePaths, path)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", baseDir, err)
	}

	return filePaths, nil
}

// Monster Energy AMA Supercross results parser
func parseText(scanner *bufio.Scanner, result *models.RaceResult) error {
	lineIndex := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch lineIndex {
		case 1:
			result.City = titleCaseWithExceptions(strings.ToLower(line))
		case 2:
			if parts := strings.SplitN(line, " - ", 2); len(parts) == 2 {
				result.Track = titleCaseWithExceptions(strings.ToLower(parts[0]))
				if parts := strings.SplitN(line, ", ", 2); len(parts) == 2 {
					result.State = parts[1]
				}
			}

		case 3:
			if parts := strings.SplitN(line, " - ", 2); len(parts) == 2 {
				_, err := fmt.Sscanf(parts[0], "ROUND %s OF %s", &result.Round, &result.TotalRounds)
				if err != nil {
					return errors.Wrapf(err, "can't scan string %s", parts[0])
				}

				layout := "January 2, 2006"
				date, err := time.Parse(layout, capitalizeMonth(strings.ToLower(parts[1])))
				if err != nil {
					return errors.Wrap(err, "failed to parse date")
				}
				result.Date = date
			}

		case 4:
			result.Class = line
		}

		if strings.HasPrefix(line, "POS.") {
			parseRiders(scanner, &result.Results)
		}

		lineIndex++
	}

	return nil
}

func capitalizeMonth(input string) string {
	if len(input) == 0 {
		return input
	}
	return string(input[0]-32) + input[1:]
}

func titleCaseWithExceptions(input string) string {
	words := strings.Fields(strings.ToLower(input)) // Приводим все к нижнему регистру
	for i, word := range words {
		if strings.ToUpper(word) == "AMA" { // Если слово "AMA", оставляем его в верхнем регистре
			words[i] = "AMA"
		} else {
			words[i] = strings.Title(word) // Иначе делаем первую букву заглавной
		}
	}
	return strings.Join(words, " ")
}

func parseRiders(scanner *bufio.Scanner, results *[]models.Rider) {
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "Humidity:") {
			break
		}

		fields := splitByColumns(line)
		if len(fields) < 5 {
			log.Println("Short row", fields)
			continue
		}

		team, ok := fields["TEAM"]
		if !ok {
			team = ""
		}

		*results = append(*results, models.Rider{
			Position:    strings.TrimSpace(fields["POS"]),
			RiderNumber: strings.TrimSpace(fields["NUMBER"]),
			Rider:       strings.TrimSpace(fields["RIDER"]),
			Hometown:    strings.TrimSpace(fields["HOMETOWN"]),
			Bike:        strings.TrimSpace(fields["BIKE"]),
			Team:        strings.TrimSpace(team),
		})
	}
}

func splitByColumns(line string) map[string]string {
	fields := make(map[string]string)
	currentField := strings.Builder{}
	spaceCount := 0
	columnOrder := []string{"POS", "NUMBER", "RIDER", "BIKE", "INTERVAL", "BEST_LAP", "HOMETOWN"}
	currentIndex := 0

	for _, r := range line {
		if r == ' ' {
			spaceCount++
			if currentIndex == 1 && currentField.Len() > 0 && spaceCount >= 1 {
				fields[columnOrder[currentIndex]] = strings.TrimSpace(currentField.String())
				currentField.Reset()
				currentIndex++
				spaceCount = 0
				continue
			}
			if spaceCount >= 3 && currentField.Len() > 0 {
				fields[columnOrder[currentIndex]] = strings.TrimSpace(currentField.String())
				currentField.Reset()
				currentIndex++
				if currentIndex >= len(columnOrder) {
					break
				}
			}
		} else {
			if spaceCount > 0 && currentField.Len() > 0 {
				currentField.WriteString(strings.Repeat(" ", spaceCount))
			}
			currentField.WriteRune(r)
			spaceCount = 0
		}
	}

	if currentField.Len() > 0 && currentIndex < len(columnOrder) {
		fields[columnOrder[currentIndex]] = strings.TrimSpace(currentField.String())
	}
	return fields
}

func outputJSON(result models.RaceResult) {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Ошибка при создании JSON: %v", err)
	}
	fmt.Println(string(jsonData))
}
