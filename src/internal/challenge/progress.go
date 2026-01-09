package challenge

import (
	"os"
	"path/filepath"

	"gti/src/internal/config"
)

type GameProgress struct {
	HighestLevelCompleted int `json:"highest_level_completed"`
}

func LoadProgress(cfg *config.Config) (*GameProgress, error) {
	progressFile := filepath.Join(config.ConfigDir, "challenge_progress.json")

	progress := &GameProgress{
		HighestLevelCompleted: 0,
	}

	if _, err := os.Stat(progressFile); os.IsNotExist(err) {
		return progress, nil
	}

	err := config.LoadJSONData(progressFile, progress)
	if err != nil {
		return &GameProgress{HighestLevelCompleted: 0}, nil
	}

	return progress, nil
}

func SaveProgress(cfg *config.Config, progress *GameProgress) error {
	progressFile := filepath.Join(config.ConfigDir, "challenge_progress.json")
	return config.SaveJSONData(progressFile, progress)
}

func GetStartingLevel(cfg *config.Config) int {
	progress, err := LoadProgress(cfg)
	if err != nil {
		return 0
	}

	return progress.HighestLevelCompleted + 1
}

func UpdateProgress(cfg *config.Config, levelCompleted int) error {
	progress, err := LoadProgress(cfg)
	if err != nil {
		progress = &GameProgress{HighestLevelCompleted: 0}
	}

	if levelCompleted > progress.HighestLevelCompleted {
		progress.HighestLevelCompleted = levelCompleted
		return SaveProgress(cfg, progress)
	}

	return nil
}
