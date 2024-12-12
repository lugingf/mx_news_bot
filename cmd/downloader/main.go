package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

var eventLinks = []string{
	"https://archives.amasupercross.com/2024/index.html?EventID=S2405",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2305",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2205",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2105",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2410",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2315",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2210",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2110",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2415",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2320",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2215",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2115",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2420",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2325",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2220",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2120",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2425",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2330",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2225",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2125",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2430",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2333",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2230",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2130",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2435",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2335",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2235",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2135",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2440",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2340",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2240",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2140",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2445",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2345",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2245",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2145",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2450",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2350",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2250",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2150",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2455",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2355",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2255",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2155",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2460",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2360",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2260",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2160",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2465",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2365",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2265",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2165",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2470",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2370",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2270",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2170",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2475",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2375",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2275",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2175",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2480",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2380",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2280",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2180",
	"https://archives.amasupercross.com/2024/index.html?EventID=S2485",
	"https://archives.amasupercross.com/2023/index.html?EventID=S2385",
	"https://archives.amasupercross.com/2022/index.html?EventID=S2285",
	"https://archives.amasupercross.com/2021/index.html?EventID=S2185",
}

func generateEventLinks() []string {
	var eventLinks []string
	for year := 24; year <= 24; year++ {
		yearString := fmt.Sprintf("S%d", year)
		eventID := fmt.Sprintf("%s%02d", yearString, 99)
		eventLink := fmt.Sprintf("https://archives.amasupercross.com/%d/index.html?EventID=%s", year+2000, eventID)
		eventLinks = append(eventLinks, eventLink)
	}
	return eventLinks
}

func main() {
	// Создаем контекст для chromedp
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Регулярное выражение для поиска PDF ссылок
	pdfRegex := regexp.MustCompile(`(?i)href\s*=\s*['"]([^'" ]+\.pdf)['"]`)

	eventLinks := generateEventLinks()
	// Посещаем каждую страницу события и ищем PDF ссылки
	for _, link := range eventLinks {
		fmt.Println("Посещение страницы события:", link)

		var pageContent string
		// Выполняем chromedp задачи для загрузки страницы и получения ее содержимого
		err := chromedp.Run(ctx,
			chromedp.Navigate(link),
			chromedp.Sleep(10*time.Second), // Ждем, чтобы динамическое содержимое загрузилось
			chromedp.OuterHTML("html", &pageContent),
		)
		fmt.Println("10 секунд прошло")
		if err != nil {
			log.Printf("Не удалось посетить страницу события: %v", err)
			continue
		}

		// Ищем ссылки на PDF файлы в полученном содержимом страницы
		matches := pdfRegex.FindAllStringSubmatch(pageContent, -1)
		for _, match := range matches {
			if len(match) > 1 {
				pdfLink := match[1]
				// Преобразуем относительные ссылки в абсолютные
				if !strings.HasPrefix(pdfLink, "http") {
					pdfLink = toAbsoluteURL(pdfLink, link)
				}

				// Скачиваем PDF файл без проверки на EventID
				fmt.Println("Найдена PDF ссылка:", pdfLink)
				downloadPDF(pdfLink, link)
			}
		}
	}
}

// toAbsoluteURL преобразует относительную ссылку в абсолютную на основе базового URL
func toAbsoluteURL(href string, baseURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}
	if u, err := url.Parse(href); err == nil {
		return base.ResolveReference(u).String()
	}
	return href
}

// загружает PDF файл и сохраняет его в папку downloads/YEAR/EVENTID/
func downloadPDF(link string, baseURL string) {
	// Разбираем ссылку для получения информации о файле и папке назначения
	parsedURL, err := url.Parse(link)
	if err != nil {
		log.Printf("Ошибка при разборе URL: %v", err)
		return
	}

	// Извлекаем параметры YEAR и EVENTID из базового URL
	baseParsedURL, err := url.Parse(baseURL)
	if err != nil {
		log.Printf("Ошибка при разборе базового URL: %v", err)
		return
	}
	year := "unknown"
	eventID := "unknown"
	if query, err := url.ParseQuery(baseParsedURL.RawQuery); err == nil {
		if val, ok := query["EventID"]; ok && len(val) > 0 {
			eventID = val[0]
			if len(eventID) >= 5 {
				year = "20" + eventID[1:3]
			}
		}
	}

	// Определяем путь для сохранения файла
	dirPath := filepath.Join("downloads", year, eventID)
	os.MkdirAll(dirPath, os.ModePerm)

	fileName := filepath.Base(parsedURL.Path)
	savePath := filepath.Join(dirPath, fileName)

	// Загружаем файл
	resp, err := http.Get(link)
	if err != nil {
		log.Printf("Ошибка загрузки файла %s: %v", link, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Ошибка HTTP статуса при загрузке файла %s: %s", link, resp.Status)
		return
	}

	// Создаем файл
	out, err := os.Create(savePath)
	if err != nil {
		log.Printf("Ошибка создания файла %s: %v", savePath, err)
		return
	}
	defer out.Close()

	// Копируем содержимое ответа в файл
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Printf("Ошибка при сохранении файла %s: %v", savePath, err)
		return
	}

	fmt.Println("Файл успешно загружен:", savePath)
}

// Этот код использует chromedp для управления браузером и загрузки страниц с выполнением JavaScript.
// После этого выполняется поиск всех PDF ссылок в загруженном HTML с использованием регулярного выражения.
// Найденные PDF файлы затем загружаются и сохраняются в соответствующие папки с годом и EventID.
