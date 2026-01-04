package app

import (
	"fmt"

	"gti/src/internal/challenge"
	"gti/src/internal/config"
	"gti/src/internal/session"
	"gti/src/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func runTUIModel(cfg *config.Config, opts tui.ModelOptions) error {
	model := tui.NewModel(cfg, opts)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type PracticeOptions struct {
	ChunkCount int
	Language   string
}

func StartPractice() error {
	return StartPracticeWithOptions(PracticeOptions{})
}

func StartPracticeWithChunks(chunkCount int) error {
	return StartPracticeWithOptions(PracticeOptions{ChunkCount: chunkCount})
}

func StartPracticeWithChunksAndLanguage(chunkCount int, language string) error {
	return StartPracticeWithOptions(PracticeOptions{ChunkCount: chunkCount, Language: language})
}

func StartPracticeWithOptions(opts PracticeOptions) error {
	cfg := config.GetConfig()
	if opts.Language != "" {
		cfg.Language.Default = opts.Language
	}

	var modelOpts tui.ModelOptions
	if opts.ChunkCount > 0 {
		sess := session.NewSessionWithChunkLimit(cfg, opts.ChunkCount)
		modelOpts = tui.ModelOptions{Session: sess}
	} else {
		modelOpts = tui.ModelOptions{Mode: "practice"}
	}

	return runTUIModel(cfg, modelOpts)
}

type CustomOptions struct {
	File    string
	Start   int
	Seconds int
}

func StartCustom(file string, start int) error {
	return StartCustomWithOptions(CustomOptions{File: file, Start: start})
}

func StartCustomTimed(file string, start int, seconds int) error {
	return StartCustomWithOptions(CustomOptions{File: file, Start: start, Seconds: seconds})
}

func StartCustomWithOptions(opts CustomOptions) error {
	cfg := config.GetConfig()
	mode := "custom"
	if opts.Seconds > 0 {
		mode = "custom-timed"
	}

	modelOpts := tui.ModelOptions{
		Mode:    mode,
		File:    opts.File,
		Start:   opts.Start,
		Seconds: opts.Seconds,
	}

	return runTUIModel(cfg, modelOpts)
}

func StartWords() error {
	cfg := config.GetConfig()
	return runTUIModel(cfg, tui.ModelOptions{Mode: "words"})
}

func StartTimed(seconds int) error {
	cfg := config.GetConfig()
	return runTUIModel(cfg, tui.ModelOptions{Mode: "timed", Seconds: seconds})
}

func StartChallengeGame() error {
	levels := []challenge.Level{}

	for i, level := range challenge.GetBuiltInLevels() {
		challengeLevel := challenge.Level{
			Name:        level.Name,
			Difficulty:  fmt.Sprintf("lv%d", i+1),
			Time:        level.TimeSeconds,
			ChunkSize:   10,
			Message:     "Level completed!",
			IsBoss:      level.IsBoss,
			MinAccuracy: level.MinAccuracy,
			MaxMistakes: level.MaxMistakes,
			MinChars:    level.MinChars,
			MinWords:    level.MinWords,
		}

		if level.IsBoss {
			words := level.MinChars / 5
			if words < 1 {
				words = 1
			}
			challengeLevel.BossRound = &challenge.BossRound{
				Words:     words,
				TimeLimit: level.TimeSeconds,
				Name:      level.Name,
			}
		}

		levels = append(levels, challengeLevel)
	}

	return challenge.StartChallengeGame(levels)
}

type CodeOptions struct {
	Language string
	Count    int
	Seconds  int
}

func StartCodePractice(language string, count int) error {
	return StartCodePracticeWithOptions(CodeOptions{
		Language: language,
		Count:    count,
	})
}

func StartCodePracticeTimed(language string, count int, seconds int) error {
	return StartCodePracticeWithOptions(CodeOptions{
		Language: language,
		Count:    count,
		Seconds:  seconds,
	})
}

func StartCodePracticeWithOptions(opts CodeOptions) error {
	cfg := config.GetConfig()

	mode := opts.Language + "-code"
	if opts.Seconds > 0 {
		mode = opts.Language + "-code-timed"
	}

	var modelOpts tui.ModelOptions
	if opts.Count > 1 {
		// Multiple snippets
		sess := session.NewSessionWithCodeSnippets(cfg, opts.Language, opts.Count)
		modelOpts = tui.ModelOptions{Session: sess}
	} else if opts.Seconds > 0 {
		// Timed single snippet
		sess := session.NewSessionWithCodeSnippetTimed(cfg, opts.Language, opts.Seconds)
		modelOpts = tui.ModelOptions{Session: sess}
	} else {
		// Single snippet
		sess := session.NewSessionWithCodeSnippet(cfg, mode)
		modelOpts = tui.ModelOptions{Session: sess}
	}

	return runTUIModel(cfg, modelOpts)
}
