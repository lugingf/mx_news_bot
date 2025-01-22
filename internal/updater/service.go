package updater

import (
	"context"
	"log/slog"

	"mx_news_bot/internal/models"
)

type Manager interface {
	Manage() error
	Name() string
}

type Checker interface {
	CheckResults() ([]string, error)
}

type Downloader interface {
	DownloadEventFiles(ctx context.Context, eventName string) error
}

type Parser interface {
	ParseFiles(files []string) ([]models.RaceResult, error)
}

type Uploader interface {
	UploadResults(results []models.RaceResult) error
}

type Service struct {
	managers []Manager
	log      *slog.Logger
}

func NewUpdater(log *slog.Logger) *Service {
	return &Service{
		managers: make([]Manager, 0),
		log:      log,
	}
}

func (s *Service) WithManager(m Manager) {
	s.managers = append(s.managers, m)
}

func (s *Service) Start() {
	for _, manager := range s.managers {
		err := manager.Manage()
		if err != nil {
			s.log.With("error", err, "name", manager.Name()).Error("failed during manager run")
		}
	}
}
