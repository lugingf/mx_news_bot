package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"mx_news_bot/internal/models"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/net/html"
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
	pdfFile := "data/2024/S2485/450SX/S1F1RES.pdf"
	htmlFile := "output/text.html"
	htmlTextsFile := "output/texts.html"

	err := convertPDFToHTML(pdfFile, htmlFile)
	if err != nil {
		fmt.Println("Ошибка при конвертации PDF в HTML:", err)
		return
	}

	file, err := os.Open(htmlTextsFile)
	if err != nil {
		fmt.Println("Ошибка при открытии файла HTML:", err)
		return
	}
	defer file.Close()

	doc, err := html.Parse(file)
	if err != nil {
		fmt.Println("Ошибка при парсинге HTML:", err)
		return
	}

	var riders []models.SXResultsRider

	var traverse func(*html.Node)
	var current models.SXResultsRider
	var collectData bool
	var fieldIndex int
	positionCounter := 1 // Счётчик для автоматической позиции

	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "b" {
			// Извлечение текста из <b> тегов
			if n.FirstChild != nil {
				text := strings.TrimSpace(n.FirstChild.Data)
				if isNumeric(text) {
					if current.Position != "" {
						riders = append(riders, current)
					}
					current = models.SXResultsRider{}
					fieldIndex = 1
					collectData = true
				}
			}
		} else if n.Type == html.TextNode && collectData {
			// Извлечение данных в зависимости от поля
			text := strings.TrimSpace(n.Data)
			if text != "" {
				switch fieldIndex {
				case 1:
					current.Number = text
				case 2:
					current.Name = text
				case 3:
					current.Hometown = text
				case 4:
					current.Bike = text
				case 5:
					current.Interval = text
				case 6:
					current.BestLap = text
				case 7:
					current.Team = text
					if current.Position == "" {
						current.Position = fmt.Sprintf("%d", positionCounter)
						positionCounter++
					}
					collectData = false
				}
				fieldIndex++
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	// Добавляем последнюю запись
	if current.Position != "" {
		riders = append(riders, current)
	}

	// Выводим результат
	fmt.Println("POS\t#\tRIDER\t\tHOMETOWN\t\tBIKE\t\tINTERVAL\tBEST LAP\tTEAM")
	for _, rider := range riders {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			rider.Position, rider.Number, rider.Name, rider.Hometown, rider.Bike, rider.Interval, rider.BestLap, rider.Team)
	}

	writeToCSV(riders)
}

func writeToCSV(riders []models.SXResultsRider) {
	file, err := os.Create("results.csv")
	if err != nil {
		log.Fatalf("Ошибка создания файла: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"POS", "NUMBER", "RIDER", "HOMETOWN", "BIKE", "INTERVAL", "BEST LAP", "TEAM"}
	if err := writer.Write(headers); err != nil {
		log.Fatalf("Ошибка записи заголовков: %v", err)
	}

	for _, rider := range riders {
		record := []string{
			rider.Position, rider.Number, rider.Name, rider.Hometown,
			rider.Bike, rider.Interval, rider.BestLap, rider.Team,
		}
		if err := writer.Write(record); err != nil {
			log.Fatalf("Ошибка записи строки: %v", err)
		}
	}
}

// выполняет команду pdftohtml для преобразования PDF в HTML
func convertPDFToHTML(pdfFile, htmlFile string) error {
	fmt.Printf("Opening PDF %s", pdfFile)
	cmd := exec.Command("pdftohtml", pdfFile, htmlFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Проверка, является ли строка числом
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
