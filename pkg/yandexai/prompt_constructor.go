package yandexai

import (
	"bytes"
	"text/template"
)

const (
	basePromptTemplate = `Ниже будет представлен текст, взятый из документа - результата гонки. 
Вытащи всю информацию и оформи ее в JSON следующего формата:
{
  "event": "Event name from data",
  "city": "City from Data",
  "stadium": "Track Name from data",
  "date": "DD-MM-YYYY",
  "round": "current round number",
  "total_rounds": total rounds amount,
  "class": "450SX or 250SX or any other",
  "results": [
    {
      "pos": "1",
      "rider_number": 1,
      "rider": "Chase Sexton",
      "hometown": "LaMoille, IL",
      "bike": "KTM 450 SX-F FE",
      "team": "Red Bull KTM Factory Racing"
    },
    {
      "pos": "2",
      "rider_number": 32,
      "rider": "Justin Cooper",
      "hometown": "Cold Springs Harbor, NY",
      "bike": "Yamaha YZ450F",
      "team": "Monster Energy Yamaha Star Racing"
    },
... continue with all rest riders
  ]
}

Вот эти данные: 
'''
{{.data}}
'''	

Выведи всех гонщиков
Ответ выведи в JSON формате, без вводных слов и объяснений. Не используй markdown или любое другое форматирование. Не делай переносов строк или отступов. Пусть все будет в одну строку. 
Весь ответ должен быть валидным JSON, чтобы можно было заанмаршалить в структуру Go
type RaceResult struct {
	Event       string  json:"event"
	City        string  json:"city"
	Track     string  json:"stadium"
	Date        string  json:"date"
	Round       string  json:"round"
	TotalRounds string  json:"total_rounds"
	Class       string  json:"class"
	Results     []Name json:"results"
}

type Name struct {
	Position    string json:"pos"
	RiderNumber string  json:"rider_number"
	Name       string json:"rider"
	Hometown    string json:"hometown"
	Bike        string json:"bike"
	Team        string json:"team"
}
`

	instructionsTemplate = `Ты - умный помощник. Система парсинга данных из различных форматов. 
`
)

func (c *APIClient) constructPrompt(data string) (string, string, error) {
	tmplPrompt, err := template.New("basePrompt").Parse(basePromptTemplate)
	if err != nil {
		return "", "", err
	}

	tmplInstructions, err := template.New("instructions").Parse(instructionsTemplate)
	if err != nil {
		return "", "", err
	}

	var promptBuffer, instrBuffer bytes.Buffer
	if err := tmplPrompt.Execute(&promptBuffer, map[string]string{"data": data}); err != nil {
		return "", "", err
	}

	if err := tmplInstructions.Execute(&instrBuffer, ""); err != nil {
		return "", "", err
	}

	return promptBuffer.String(), instrBuffer.String(), nil
}
