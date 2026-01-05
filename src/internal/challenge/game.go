
package challenge

import (
	"fmt"
	"strings"
	"time"

	"gti/src/internal"
	"gti/src/internal/config"
	"gti/src/internal/session"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BossRound struct {
	Words        int
	TimeLimit    int
	Name         string
	TriggerChunk int
}

type Level struct {
	Name        string
	Difficulty  string
	Time        int
	ChunkSize   int
	BossRound   *BossRound
	BossRounds  []BossRound
	Message     string
	IsBoss      bool
	MinAccuracy float64
	MaxMistakes int
	MinChars    int
	MinWords    int
}

type GameTiming struct {
	StartTime      time.Time
	LevelStartTime time.Time
	TimeLeft       int
}

type GameState struct {
	Levels      []Level
	CurrentLevel int
	Phase       string
	ChunkIndex  int
	GameTiming
	WordsTyped  int
	Mistakes    int
	TotalChars  int
	BossResults []BossResult
}

type BossResult struct {
	Name      string
	WPM       float64
	Accuracy  float64
	Completed bool
}

type GameModel struct {
	config  *config.Config
	state   *GameState
	sess    *session.Session
	width   int
	height  int
	mode    string
}

func NewGameModel(cfg *config.Config, levels []Level) GameModel {
	startingLevel := GetStartingLevel(cfg)
	now := time.Now()

	state := &GameState{
		Levels:      levels,
		CurrentLevel: startingLevel,
		Phase:       "normal",
		ChunkIndex:  0,
		GameTiming: GameTiming{
			TimeLeft:       levels[startingLevel].Time,
			StartTime:      now,
			LevelStartTime: now,
		},
		BossResults: []BossResult{},
	}

	sess := session.NewSessionWithChallenge(cfg, fmt.Sprintf("lv%d", state.CurrentLevel+1))

	model := GameModel{
		config: cfg,
		state:  state,
		sess:   sess,
	}

	currentLevel := levels[startingLevel]
	model.resetLevelState(currentLevel)

	return model
}

func (m GameModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.sess.Start(),
		m.tickTimer(),
	)
}

func (m GameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case session.SessionCompleteMsg:
		return m.handleSessionComplete()
	case TickMsg:
		return m.handleTick()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	}
	return m, nil
}

func (m GameModel) View() string {
	switch m.mode {
	case "help":
		return m.viewHelp()
	case "quit":
		return m.viewQuit()
	default:
		switch m.state.Phase {
		case "complete":
			return m.viewLevelComplete()
		case "failed":
			return m.viewLevelFailed()
		default:
			return m.viewNormalPlay()
		}
	}
}

func (m GameModel) viewNormalPlay() string {

	content := m.sess.View(m.width, m.height)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Background(lipgloss.Color(m.config.Theme.Colors.Background)).
		Render(content)
}

func (m GameModel) viewLevelComplete() string {
	level := m.state.Levels[m.state.CurrentLevel]

	content := fmt.Sprintf(`Level %d Complete!
%s

Stats:
WPM: %.1f
Accuracy: %.1f%%
Mistakes: %d
Words Typed: %d

Boss Rounds: %d/%d completed`,
		m.state.CurrentLevel+1,
		level.Message,
		m.calculateWPM(),
		m.calculateAccuracy(),
		m.state.Mistakes,
		m.state.WordsTyped,
		m.countCompletedBosses(),
		len(m.state.BossResults),
	)

	if m.state.CurrentLevel < len(m.state.Levels)-1 {
		content += "\n\nPress Enter to continue to next level..."
	} else {
		content += "\n\nCongratulations! All levels completed!"
	}

	return m.renderLevelDialog(content, m.config.Theme.Colors.TextPrimary)
}

func (m GameModel) viewLevelFailed() string {
	level := m.state.Levels[m.state.CurrentLevel]

	requirements := m.getLevelRequirements(level)

	content := fmt.Sprintf(`❌ Level %d Failed!

Your Stats:
Accuracy: %.1f%% (Required: %.1f%%)
Mistakes: %d (Max allowed: %d)
Chars Typed: %d (Required: %d)
Words Typed: %d (Required: %d)

Requirements not met. Try again!

Press R to retry this level
Press Q to quit`,
		m.state.CurrentLevel+1,
		m.calculateAccuracy(), requirements.MinAccuracy,
		m.state.Mistakes, requirements.MaxMistakes,
		m.state.TotalChars, level.MinChars,
		m.state.WordsTyped, requirements.MinWords,
	)

	return m.renderLevelDialog(content, "red")
}

func (m GameModel) viewHelp() string {
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.config.Theme.Colors.TextPrimary)).
		Background(lipgloss.Color(m.config.Theme.Colors.Background)).
		Render("Help overlay - Press ESC to close\n\nShortcuts:\nCtrl+Q: Quit confirmation\nCtrl+C: Force quit\nEsc: Restart level\nCtrl+H: Help\nBackspace: Delete\nLeft/Right: Navigate segments\n\nChallenge Mode:\nComplete levels with increasing difficulty\nEnter: Continue to next level\nR: Retry failed level")

	return m.renderDialogBox(helpText, 2, 1, m.config.Theme.Colors.TextPrimary, true)
}

func (m GameModel) viewQuit() string {
	quitText := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.config.Theme.Colors.TextPrimary)).
		Background(lipgloss.Color(m.config.Theme.Colors.Background)).
		Render(`Are you sure you want to quit the challenge?

"Quit?" (y/n)`)

	return m.renderDialogBox(quitText, 4, 2, m.config.Theme.Colors.TextPrimary, true)
}

func (m *GameModel) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case "help":
		if key.String() == "esc" {
			m.mode = ""
		}
		return m, nil
	case "quit":
		if key.String() == "y" || key.String() == "Y" {
			return m, tea.Quit
		}
		m.mode = ""
		return m, nil
	default:
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+q":
			m.mode = "quit"
			return m, nil
		case "ctrl+h":
			m.mode = "help"
			return m, nil
		case "esc":
			return m.retryLevel()
		}

		switch m.state.Phase {
		case "complete":
			if key.String() == "enter" {
				return m.advanceLevel()
			}
			return m, nil
		case "failed":
			if key.String() == "r" {
				return m.retryLevel()
			}
			return m, nil
		}

		cmd := m.sess.HandleInput(key)
		return m, cmd
	}
}

func (m *GameModel) handleSessionComplete() (tea.Model, tea.Cmd) {
	level := m.state.Levels[m.state.CurrentLevel]

	wordsInChunk := strings.Fields(m.sess.GetText())
	m.state.WordsTyped += len(wordsInChunk)
	m.state.Mistakes += m.sess.GetMistakes()
	m.state.TotalChars += len(m.sess.TypedText())

	if m.state.Phase == "boss" {
		var bossName string
		if level.BossRound != nil {
			bossName = level.BossRound.Name
		} else {
			for _, boss := range level.BossRounds {
				if boss.TriggerChunk == m.state.ChunkIndex {
					bossName = boss.Name
					break
				}
			}
		}

		result := m.createBossResult(bossName, true)
		m.state.BossResults = append(m.state.BossResults, result)
		m.state.Phase = "normal"
		m.state.ChunkIndex++
		m.generateNextChunk()
	} else if level.BossRound != nil {
		if m.checkLevelRequirements(level) {
			result := m.createBossResult(level.BossRound.Name, true)
			m.state.BossResults = append(m.state.BossResults, result)
			m.state.Phase = "complete"
		} else {
			m.state.Phase = "failed"
		}
	} else {
		// For non-boss levels, always continue to next chunk until time runs out
		m.state.ChunkIndex++
		m.generateNextChunk()
	}

	return m, nil
}

func (m *GameModel) handleTick() (tea.Model, tea.Cmd) {
	m.state.TimeLeft--
	m.sess.RemainingTimeDisplay = m.state.TimeLeft
	if m.state.TimeLeft <= 0 {
		level := m.state.Levels[m.state.CurrentLevel]
		if level.BossRound != nil {
			result := m.createBossResult(level.BossRound.Name, false)
			m.state.BossResults = append(m.state.BossResults, result)
			m.state.Phase = "complete"
		} else {

			if m.checkLevelRequirements(level) {
				m.state.Phase = "complete"
			} else {
				m.state.Phase = "failed"
			}
		}
		return m, nil
	}
	return m, m.tickTimer()
}

func (m *GameModel) generateNextChunk() {
	level := m.state.Levels[m.state.CurrentLevel]
	chunkText := internal.GenerateWordsDynamic(level.ChunkSize, m.config.Language.Default)
	m.sess.SetText(chunkText)
	m.sess.ExternalMistakes = m.state.Mistakes
	m.sess.Start()
}

func (m *GameModel) advanceLevel() (tea.Model, tea.Cmd) {
	UpdateProgress(m.config, m.state.CurrentLevel)

	record := &session.SessionRecord{
		Mode:       "challenge",
		Tier:       fmt.Sprintf("lv%d", m.state.CurrentLevel+1),
		TextLength: len(m.sess.GetText()),
		DurationMs: time.Since(m.state.StartTime).Milliseconds(),
		WPM:        m.calculateWPM(),
		CPM:        float64(m.state.WordsTyped) / time.Since(m.state.StartTime).Minutes() * 5,
		Accuracy:   m.calculateAccuracy(),
		Mistakes:   m.state.Mistakes,
	}
	session.SaveSessionRecord(m.config, record)

	m.state.CurrentLevel++
	if m.state.CurrentLevel >= len(m.state.Levels) {
		return m, tea.Quit
	}

	nextLevel := m.state.Levels[m.state.CurrentLevel]
	m.resetLevelState(nextLevel)

	return m, nil
}

func (m GameModel) calculateWPM() float64 {
	levelDuration := time.Since(m.state.LevelStartTime)
	return session.CalculateWPM(m.state.TotalChars, levelDuration)
}

func (m GameModel) calculateAccuracy() float64 {
	return session.CalculateAccuracy(m.state.TotalChars, m.state.Mistakes)
}

func (m GameModel) countCompletedBosses() int {
	count := 0
	for _, result := range m.state.BossResults {
		if result.Completed {
			count++
		}
	}
	return count
}

func (m GameModel) createBossResult(name string, completed bool) BossResult {
	return BossResult{
		Name:      name,
		WPM:       m.calculateWPM(),
		Accuracy:  m.calculateAccuracy(),
		Completed: completed,
	}
}

func (m *GameModel) startHiddenBossRound(boss BossRound) {
	bossText := internal.GenerateWordsDynamic(boss.Words, m.config.Language.Default)
	m.sess.SetText(bossText)
	m.state.Phase = "boss"
	m.state.TimeLeft = boss.TimeLimit
	m.sess.Start()
}

func (m GameModel) tickTimer() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

type TickMsg struct{}

type LevelRequirements struct {
	MinWPM      float64
	MinAccuracy float64
	MaxMistakes int
	MinWords    int
}

func (m GameModel) checkLevelRequirements(level Level) bool {
	accuracy := m.calculateAccuracy()
	mistakes := m.state.Mistakes

	return accuracy >= level.MinAccuracy &&
		mistakes <= level.MaxMistakes &&
		m.state.TotalChars >= level.MinChars &&
		m.state.WordsTyped >= level.MinWords
}

// renderDialogBox creates a unified dialog box with configurable styling
func (m GameModel) renderDialogBox(content string, paddingX, paddingY int, borderColor string, includeBackground bool) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(paddingY, paddingX).
		Align(lipgloss.Center)

	if includeBackground {
		boxStyle = boxStyle.
			BorderBackground(lipgloss.Color(m.config.Theme.Colors.Background)).
			Background(lipgloss.Color(m.config.Theme.Colors.Background))
	}

	box := boxStyle.Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(m.config.Theme.Colors.Background)))
}

func (m GameModel) renderLevelDialog(content string, borderColor string) string {
	return m.renderDialogBox(content, 2, 1, borderColor, false)
}

func (m GameModel) getLevelRequirements(level Level) LevelRequirements {
	return LevelRequirements{
		MinWPM:      0,
		MinAccuracy: level.MinAccuracy,
		MaxMistakes: level.MaxMistakes,
		MinWords:    level.MinWords,
	}
}

func (m *GameModel) resetLevelState(level Level) {
	m.state.Phase = "normal"
	m.state.ChunkIndex = 0
	m.state.TimeLeft = level.Time
	m.state.LevelStartTime = time.Now()
	m.state.WordsTyped = 0
	m.state.Mistakes = 0
	m.state.TotalChars = 0

	var text string
	if level.BossRound != nil {
		text = internal.GenerateWordsDynamic(level.BossRound.Words, m.config.Language.Default)
	} else {
		text = internal.GenerateWordsDynamic(level.ChunkSize, m.config.Language.Default)
	}
	m.sess.SetText(text)
	m.sess.ExternalMistakes = m.state.Mistakes
	m.sess.SetTier(fmt.Sprintf("lv%d", m.state.CurrentLevel+1))
	m.sess.Start()
}

func (m *GameModel) retryLevel() (tea.Model, tea.Cmd) {
	level := m.state.Levels[m.state.CurrentLevel]
	m.resetLevelState(level)
	return m, nil
}

func StartChallengeGame(levels []Level) error {
	cfg := config.GetConfig()
	model := NewGameModel(cfg, levels)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
