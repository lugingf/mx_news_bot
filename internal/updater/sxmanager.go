package updater

import (
	"context"
	"github.com/pkg/errors"

	"mx_news_bot/config"
	"mx_news_bot/internal/models"
)

const managerName = "SXManager"

type SXManager struct {
	config config.ChampionshipConfig
	ch     Checker
	dwn    Downloader
	prs    Parser
	upl    Uploader
}

func NewSXManager(config config.ChampionshipConfig) *SXManager {
	return &SXManager{
		config: config,
	}
}

func (m *SXManager) Manage() error {
	files, err := m.ch.CheckResults()
	if err != nil {
		return errors.Wrap(err, "checking results")
	}

	ctx := context.Background()
	err = m.dwn.DownloadEventFiles(ctx, "Event Name")
	if err != nil {
		return errors.Wrap(err, "downloading files")
	}

	results, err := m.prs.ParseFiles(files)
	if err != nil {
		return errors.Wrap(err, "parsing files")
	}

	err = m.upl.UploadResults(results)
	if err != nil {
		return errors.Wrap(err, "uploading results")
	}

	return nil
}

func (m *SXManager) Name() string {
	return managerName
}

func (m *SXManager) checkResults() ([]string, error) {
	return m.ch.CheckResults()
}

func (m *SXManager) downloadFiles(ctx context.Context, eventName string) error {
	return m.dwn.DownloadEventFiles(ctx, eventName)
}

func (m *SXManager) parseFiles(files []string) ([]models.RaceResult, error) {
	return m.prs.ParseFiles(files)
}

func (m *SXManager) uploadResults(results []models.RaceResult) error {
	return m.upl.UploadResults(results)
}
