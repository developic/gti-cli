package app

import (
	"fmt"

	"gti/src/internal"
	"gti/src/internal/challenge"
	"gti/src/internal/config"
	"gti/src/internal/session"
	"gti/src/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

// AppOptions defines all possible options for starting the typing application
type AppOptions struct {
	Mode      string // "practice", "words", "timed", "custom", "code"
	Language  string // for code mode
	ChunkCount int   // for practice mode
	File       string // for custom mode
	Start      int    // for custom mode
	Seconds    int    // for timed modes
	CodeCount  int    // for code mode (multiple snippets)
}

func runTUIModel(cfg *config.Config, opts tui.ModelOptions) error {
	model := tui.NewModel(cfg, opts)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// StartApp starts the typing application with the given options
func StartApp(opts AppOptions) error {
	cfg := config.GetConfig()

	// Override language if specified
	if opts.Language != "" {
		cfg.Language.Default = opts.Language
	}

	var modelOpts tui.ModelOptions

	switch opts.Mode {
	case "practice":
		if opts.ChunkCount > 0 {
			sess := session.NewSessionWithChunkLimit(cfg, opts.ChunkCount)
			modelOpts = tui.ModelOptions{Session: sess}
		} else {
			modelOpts = tui.ModelOptions{Mode: "practice"}
		}

	case "words":
		modelOpts = tui.ModelOptions{Mode: "words"}

	case "timed":
		modelOpts = tui.ModelOptions{Mode: "timed", Seconds: opts.Seconds}

	case "custom":
		mode := "custom"
		if opts.Seconds > 0 {
			mode = "custom"
		}
		modelOpts = tui.ModelOptions{
			Mode:    mode,
			File:    opts.File,
			Start:   opts.Start,
			Seconds: opts.Seconds,
		}

	case "custom-code":
		mode := "code"
		if opts.Seconds > 0 {
			mode = "code" // Keep as "code" for timed mode too
		}
		modelOpts = tui.ModelOptions{
			Mode:    mode,
			File:    opts.File,
			Start:   opts.Start,
			Seconds: opts.Seconds,
		}

	case "code":
		if opts.Seconds > 0 && opts.CodeCount > 1 {
			// Multiple timed snippets - create a chunked session with timed chunks
			sess := session.NewSessionWithCodeSnippetsTimed(cfg, opts.Language, opts.CodeCount, opts.Seconds)
			modelOpts = tui.ModelOptions{Session: sess}
		} else if opts.Seconds > 0 {
			// Single timed snippet
			text := internal.GenerateCodeSnippet(opts.Language)
			sess := session.NewSessionTimed(cfg, "code", text, nil, 0, opts.Seconds)
			modelOpts = tui.ModelOptions{Session: sess}
		} else if opts.CodeCount > 1 {
			// Multiple snippets
			sess := session.NewSessionWithCodeSnippets(cfg, opts.Language, opts.CodeCount)
			modelOpts = tui.ModelOptions{Session: sess}
		} else {
			// Single snippet
			sess := session.NewSessionWithCodeSnippet(cfg, "code")
			modelOpts = tui.ModelOptions{Session: sess}
		}

	default:
		return fmt.Errorf("unknown mode: %s", opts.Mode)
	}

	return runTUIModel(cfg, modelOpts)
}

// Legacy functions for backward compatibility
func StartPractice() error {
	return StartApp(AppOptions{Mode: "practice"})
}

func StartPracticeWithChunks(chunkCount int) error {
	return StartApp(AppOptions{Mode: "practice", ChunkCount: chunkCount})
}

func StartPracticeWithChunksAndLanguage(chunkCount int, language string) error {
	return StartApp(AppOptions{Mode: "practice", ChunkCount: chunkCount, Language: language})
}

func StartWords() error {
	return StartApp(AppOptions{Mode: "words"})
}

func StartTimed(seconds int) error {
	return StartApp(AppOptions{Mode: "timed", Seconds: seconds})
}

func StartCustom(file string, start int) error {
	return StartApp(AppOptions{Mode: "custom", File: file, Start: start})
}

func StartCustomTimed(file string, start int, seconds int) error {
	return StartApp(AppOptions{Mode: "custom", File: file, Start: start, Seconds: seconds})
}

func StartCodePractice(language string, count int) error {
	return StartApp(AppOptions{Mode: "code", Language: language, CodeCount: count})
}

func StartCodePracticeTimed(language string, count int, seconds int) error {
	return StartApp(AppOptions{Mode: "code", Language: language, CodeCount: count, Seconds: seconds})
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
