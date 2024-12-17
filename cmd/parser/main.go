package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"mx_news_bot/internal/models"
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

func main() {
	//pdfFile := "data/2014/S1499/MonsterEnergyCUP/MCF1RES.pdf"
	//pdfFile := "data/2024/S2485/250SX/S2F1RES.pdf"
	pdfFile := "data/2024/S2485/450SX/S1F1RES.pdf"
	textFile := "output/S1F1RES.txt"

	err := convertPDFToText(pdfFile, textFile)
	if err != nil {
		log.Fatalf("Ошибка при конвертации PDF в текст: %v", err)
	}

	file, err := os.Open(textFile)
	if err != nil {
		log.Fatalf("Ошибка при открытии файла текста: %v", err)
	}
	defer file.Close()

	var raceResult models.RaceResult
	scanner := bufio.NewScanner(file)
	parseText(scanner, &raceResult)

	outputJSON(raceResult)
}

func convertPDFToText(pdfFile, textFile string) error {
	fmt.Printf("Converting PDF %s to text\n", pdfFile)
	cmd := exec.Command("pdftotext", "-f", "1", "-l", "1", "-layout", pdfFile, textFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func parseText(scanner *bufio.Scanner, result *models.RaceResult) {
	lineIndex := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch lineIndex {
		case 0:
			result.Event = line
		case 1:
			result.City = line
		case 2:
			result.Stadium = line
		case 3:
			if parts := strings.SplitN(line, " - ", 2); len(parts) == 2 {
				fmt.Sscanf(parts[0], "ROUND %s OF %s", &result.Round, &result.TotalRounds)
				result.Date = parts[1]
			}
		case 4:
			result.Class = line
		}
		if strings.HasPrefix(line, "POS.") {
			parseRiders(scanner, &result.Results)
		}
		lineIndex++
	}
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
	columnOrder := []string{"POS", "NUMBER", "RIDER", "HOMETOWN", "BIKE", "INTERVAL", "BEST_TIME", "TEAM"}
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
