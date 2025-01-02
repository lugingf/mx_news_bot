package bot

import (
	"log/slog"
	"mx_news_bot/internal/service"
	"sync"
)

type StateController struct {
	app        *service.BotBackend
	log        *slog.Logger
	userStates map[int64]string
	mu         sync.Mutex
}

func (sc *StateController) SetUserState(userID int64, state string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.userStates[userID] = state
}

func (sc *StateController) getUserState(userID int64) string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.userStates[userID]
}

func (sc *StateController) deleteUserState(userID int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.userStates, userID)
}

const (
	askNameResponse        = "Как вас зовут?"
	askBirthdateResponse   = "Когда вы родились? (DD-MM-YYYY)."
	askBirthplaceResponse  = "Где вы родились (город)?"
	askLivingPlaceResponse = "Где вы живете сейчас?"
	detailsSavedResponse   = "Информация сохранена. Теперь можно запросить гороскоп"

	errorSaveName          = "Не получилось сохранить имя. Пожалуйста, попробуйте позже."
	errorInvalidDateFormat = "Кажется не тот формат даты. Пожалуйста, напишите в виде DD-MM-YYYY (ДД-ММ-ГГГГ)."
	errorInvalidYear       = "Я не уверен, что введенный год верный. Пожалуйста укажите реальный (ДД-ММ-ГГГГ)."
	errorSaveBirthdate     = "Не получилось сохранить дату рождения. Пожалуйста, попробуйте позже."
	errorSaveBirthplace    = "Не получилось сохранить место рождения. Пожалуйста, попробуйте позже."
	errorSaveLivingPlace   = "Не получилось сохранить место где вы живете. Пожалуйста, попробуйте позже."
	defaultErrorResponse   = "Запрос непонятен. Пожалуйста, используйте команду /start чтобы активировать бота"

	dateLayout = "02-01-2006"
)

const (
	stateAskName        = "ask_name"
	stateAskBirthdate   = "ask_birthdate"
	stateAskBirthplace  = "ask_birthplace"
	stateAskLivingPlace = "ask_living_place"
)

func (sc *StateController) HandleBtnUpdateName(userID int64) string {
	sc.SetUserState(userID, stateAskName)
	return askNameResponse
}

func (sc *StateController) HandleBtnUpdateBirthdate(userID int64) string {
	sc.SetUserState(userID, stateAskBirthdate)
	return askBirthdateResponse
}

func (sc *StateController) HandleBtnUpdateBirthplace(userID int64) string {
	sc.SetUserState(userID, stateAskBirthplace)
	return askBirthplaceResponse
}

func (sc *StateController) HandleBtnUpdateLivingPlace(userID int64) string {
	sc.SetUserState(userID, stateAskLivingPlace)
	return askLivingPlaceResponse
}
