package bot

import (
	"log/slog"
	"sync"
	
	"mx_news_bot/internal/service"
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
	askNameResponse = "Как вас зовут?"

	dateLayout = "02-01-2006"
)

const (
	stateAskName = "ask_name"
)

func (sc *StateController) HandleBtnUpdateName(userID int64) string {
	sc.SetUserState(userID, stateAskName)
	return askNameResponse
}
