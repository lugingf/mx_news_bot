package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"mx_news_bot/internal/models"
	"mx_news_bot/internal/pdfconverter"
	"mx_news_bot/internal/storage"
)

const (
	championshipSX    = "Monster Energy AMA Supercross"
	championshipProMX = "Pro Motocross Championship"
)

const (
	raceTypeMain     = "Main Event"
	raceTypeHeat1    = "Heat 1"
	raceTypeHeat2    = "Heat 2"
	raceTypeWestHeat = "West Heat"
	raceTypeEastHeat = "East Heat"
	raceTypeRace1    = "Race 1"
	raceTypeRace2    = "Race 2"
	raceTypeRace3    = "Race 3"
)

type AMASupercross struct {
	repo   *storage.Repository
	cfg    *Config
	logger *slog.Logger
}

type Config struct {
	DryRun     bool
	DataDir    string
	OutputFile string
}

func New(r *storage.Repository, cfg *Config, l *slog.Logger) *AMASupercross {
	return &AMASupercross{
		repo:   r,
		cfg:    cfg,
		logger: l,
	}
}

func (p *AMASupercross) ParseFile(pdfFile string, event models.EventToCheck) (models.RaceResult, error) {
	err := pdfconverter.ConvertPDFToText(pdfFile, p.cfg.OutputFile)
	if err != nil {
		return models.RaceResult{}, errors.Wrapf(err, "PDF to text convertation error: %v", err)
	}

	file, err := os.Open(p.cfg.OutputFile)
	if err != nil {
		return models.RaceResult{}, errors.Wrapf(err, "txt file open error: %v", err)
	}

	defer file.Close()

	raceResult, err := p.getRaceResult(pdfFile, file, event)
	if err != nil {
		p.logger.Error("can't get race result", "error", err, "file", pdfFile)
		return raceResult, errors.Wrapf(err, "can't get race result")
	}
	// debug
	p.outputJSON(raceResult)

	return raceResult, nil
}

func (p *AMASupercross) UploadRaceResult(raceResult models.RaceResult) error {
	if p.cfg.DryRun {
		return nil
	}

	err := p.repo.UploadRaceResultsSMX(raceResult)
	if err != nil {
		p.logger.Error("Cant upload race results", "race", raceResult.EventName, "error", err)
		return errors.Wrap(err, "Cant upload race results")
	}

	return nil
}

// FIXME do it right
func (p *AMASupercross) getRoundNumber(fileName string, event models.EventToCheck) string {
	if event.RoundNumber != "" {
		return event.RoundNumber
	}

	if strings.Contains(fileName, "Anaheim 1") {
		return "1"
	}
	if strings.Contains(fileName, "San Diego") {
		return "2"
	}
	if strings.Contains(fileName, "Anaheim 2") {
		return "3"
	}
	if strings.Contains(fileName, "Glendale") {
		return "4"
	}
	if strings.Contains(fileName, "Tampa") {
		return "5"
	}
	if strings.Contains(fileName, "Detroit") {
		return "6"
	}
	if strings.Contains(fileName, "Indianapolis") {
		return "9"
	}
	return "0"
}

func (p *AMASupercross) getRaceType(fileName string) string {
	switch {
	case strings.Contains(fileName, "Main_Event"):
		return raceTypeMain
	case strings.Contains(fileName, "Heat_1"):
		return raceTypeHeat1
	case strings.Contains(fileName, "Heat_2"):
		return raceTypeHeat2
	case strings.Contains(fileName, "West_Heat"):
		return raceTypeWestHeat
	case strings.Contains(fileName, "East_Heat"):
		return raceTypeEastHeat
	case strings.Contains(fileName, "Race#1"):
		return raceTypeRace1
	case strings.Contains(fileName, "Race#2"):
		return raceTypeRace2
	case strings.Contains(fileName, "Race#3"):
		return raceTypeRace3
	}

	return "Undefined"
}

func (p *AMASupercross) getRaceResult(pdfFile string, file io.Reader, event models.EventToCheck) (models.RaceResult, error) {
	var raceResult models.RaceResult

	raceResult.RaceType = p.getRaceType(pdfFile)
	raceResult.ChampName = championshipSX
	raceResult.Round = p.getRoundNumber(pdfFile, event)

	totalRounds, err := p.repo.GetChampRoundsCount(event.ChampionshipID)
	if err != nil {
		p.logger.Error("can't get total rounds count", "error", err)
		totalRounds = 17
	}
	raceResult.TotalRounds = strconv.Itoa(totalRounds)

	scanner := bufio.NewScanner(file)
	err = p.parseTable(scanner, &raceResult)
	if err != nil {
		return models.RaceResult{}, errors.Wrapf(err, "parse race result %s", pdfFile)
	}

	raceResult.EventCode, err = p.getEventCode(raceResult.Date, raceResult.ChampName, raceResult.Round)
	if err != nil {
		return models.RaceResult{}, errors.Wrapf(err, "cant get eventCode %s", pdfFile)
	}

	return raceResult, nil
}

func (p *AMASupercross) getEventCode(date time.Time, name, roundNum string) (string, error) {
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

func (p *AMASupercross) CollectFiles(eventNames []string) ([]string, error) {
	var filePaths []string

	// Проходим по всем поддиректориям
	err := filepath.Walk(p.cfg.DataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "can't walk on directories")
		}

		if !info.IsDir() {
			if len(eventNames) == 0 {
				filePaths = append(filePaths, path)
				return nil
			}

			name := info.Name()
			if p.inList(name, eventNames) {
				filePaths = append(filePaths, path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", p.cfg.DataDir, err)
	}

	return filePaths, nil
}

func (p *AMASupercross) inList(name string, list []string) bool {
	for _, need := range list {
		if strings.Contains(name, need) {
			return true
		}
	}
	return false
}

// Monster Energy AMA Supercross results parser

func (p *AMASupercross) parseTable(scanner *bufio.Scanner, result *models.RaceResult) error {
	parsers := map[int]func(string, *models.RaceResult) error{
		1: p.parseCityTrack,
		2: p.parseDate,
		4: p.parseClass,
	}

	lineIndex := 0
	ridersParsed := false // Флаг, чтобы избежать лишних проверок
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if parser, ok := parsers[lineIndex]; ok && !ridersParsed {
			if err := parser(line, result); err != nil {
				return err
			}
		}

		// Начинаем парсинг гонщиков только один раз
		if !ridersParsed && strings.Contains(line, "POS") {
			fmt.Println("parsing riders")
			p.parseRiders(scanner, &result.Results)
			ridersParsed = true
		}

		lineIndex++
	}

	return nil
}

func (p *AMASupercross) parseCityTrack(line string, result *models.RaceResult) error {
	result.EventName = p.titleCaseWithExceptions(strings.ToLower(line))
	return nil
}

func (p *AMASupercross) parseDate(line string, result *models.RaceResult) error {
	// Example: "Jan 11, 2025"
	layout := "Jan 2, 2006"
	date, err := time.Parse(layout, p.capitalizeMonth(strings.ToLower(line)))
	if err != nil {
		return errors.Wrap(err, "failed to parse date")
	}
	result.Date = date
	return nil
}

func (p *AMASupercross) parseClass(line string, result *models.RaceResult) error {
	result.Class = strings.Split(line, " ")[0] + "SX"
	return nil
}

func (p *AMASupercross) capitalizeMonth(input string) string {
	if len(input) == 0 {
		return input
	}
	return string(input[0]-32) + input[1:]
}

func (p *AMASupercross) titleCaseWithExceptions(input string) string {
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

func (p *AMASupercross) parseRiders(scanner *bufio.Scanner, results *[]models.Rider) {
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "Generated by") {
			fmt.Println("End of table")
			break
		}

		fields := p.splitByColumns(line)
		if len(fields) < 5 {
			slog.Info("Short row", "fields", fields)
			continue
		}

		team, ok := fields["TEAM"]
		if !ok {
			team = ""
		}

		*results = append(*results, models.Rider{
			Position:    strings.TrimSpace(fields["POS"]),
			RiderNumber: strings.TrimSpace(fields["NUMBER"]),
			Name:        strings.TrimSuffix(strings.TrimSpace(fields["RIDER"]), " (HS)"),
			Hometown:    strings.TrimSpace(fields["HOMETOWN"]),
			Bike:        strings.TrimSpace(fields["BIKE"]),
			Team:        strings.TrimSpace(team),
		})
	}
}

func (p *AMASupercross) splitByColumns(line string) map[string]string {
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

func (p *AMASupercross) outputJSON(result models.RaceResult) {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		slog.Error("marshaling error: %v", err)
		return
	}

	fmt.Println(string(jsonData))
}
