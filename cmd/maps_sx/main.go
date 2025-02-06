package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	// URL целевой страницы
	url := "https://www.supercrosslive.com/2025-track-maps/"

	// Получаем содержимое страницы
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error fetching page: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: status code %d\n", resp.StatusCode)
		return
	}

	// Создаём документ из тела ответа
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Printf("Error parsing HTML: %v\n", err)
		return
	}

	// Компилируем регулярное выражение для имен файлов
	re := regexp.MustCompile(`(?i)^Rd.*\.png$`)

	// Ищем элементы <img> и обрабатываем подходящие
	doc.Find("img").Each(func(index int, item *goquery.Selection) {
		src, exists := item.Attr("src")
		if !exists {
			return
		}

		// Извлекаем имя файла из URL
		fileName := path.Base(src)
		if re.MatchString(fileName) {
			// Если URL относительный, формируем абсолютный
			if !strings.HasPrefix(src, "http") {
				src = strings.TrimRight(url, "/") + "/" + strings.TrimLeft(src, "/")
			}
			fmt.Printf("Downloading: %s\n", src)
			err := downloadFile(fileName, src)
			if err != nil {
				fmt.Printf("Error downloading %s: %v\n", src, err)
			} else {
				fmt.Printf("Saved: %s\n", fileName)
			}
		}
	})
}

// downloadFile загружает файл по указанному URL и сохраняет его в локальную файловую систему.
func downloadFile(filePath string, url string) error {
	// Получаем данные по URL
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Создаём локальный файл
	out, err := os.Create("maps/" + filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Копируем содержимое ответа в файл
	_, err = io.Copy(out, resp.Body)
	return err
}
