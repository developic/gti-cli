package tui

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

type Mode string

const (
	ModeTyping  Mode = "typing"
	ModeHelp    Mode = "help"
	ModeResults Mode = "results"
	ModeQuit    Mode = "quit"
)

type Model struct {
	config    *config.Config
	mode      Mode
	sess      *session.Session
	startTime time.Time
	timer     *time.Timer
	quitting  bool
	width     int
	height    int
}

type ModelOptions struct {
	Mode    string
	File    string
	Start   int
	Seconds int
	Session *session.Session
}

func NewModel(cfg *config.Config, opts ModelOptions) Model {
	var sess *session.Session

	if opts.Session != nil {
		sess = opts.Session
	} else if opts.File != "" && opts.Seconds > 0 {
		paragraphs := session.LoadParagraphs(opts.File)
		text := session.GetParagraphAtStart(paragraphs, opts.Start)
		sess = session.NewSessionTimed(cfg, "custom-timed", text, paragraphs, opts.Start-1, opts.Seconds)
	} else if opts.File != "" {
		sess = session.NewSessionWithCustomText(cfg, opts.Mode, opts.File, opts.Start)
	} else if opts.Seconds > 0 {
		text := internal.GenerateWordsDynamic(session.DefaultWordCount, cfg.Language.Default)
		sess = session.NewSessionTimed(cfg, "timed", text, nil, 0, opts.Seconds)
	} else {
		sess = session.NewSession(cfg, opts.Mode)
	}

	return Model{
		config: cfg,
		mode:   ModeTyping,
		sess:   sess,
	}
}

func NewModelWithCustomText(cfg *config.Config, mode, file string, start int) Model {
	return NewModel(cfg, ModelOptions{Mode: mode, File: file, Start: start})
}

func NewModelWithTimed(cfg *config.Config, seconds int) Model {
	return NewModel(cfg, ModelOptions{Mode: "timed", Seconds: seconds})
}

func NewModelWithCustomTimed(cfg *config.Config, file string, start int, seconds int) Model {
	return NewModel(cfg, ModelOptions{Mode: "custom-timed", File: file, Start: start, Seconds: seconds})
}

func NewModelWithSession(cfg *config.Config, sess *session.Session) Model {
	return NewModel(cfg, ModelOptions{Session: sess})
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.sess.Start(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sess.MarkLayoutDirty()
		return m, nil
	case session.SessionCompleteMsg:
		m.mode = ModeResults
		return m, nil
	case session.TimerTickMsg:
		return m, m.sess.UpdateTimer()
	}
	return m, nil
}

func (m Model) View() string {
	if m.width < 40 || m.height < 10 {
		return "Terminal too small. Please resize to at least 40x10.\nPress Ctrl+C to quit."
	}
	switch m.mode {
	case ModeTyping:
		return m.viewTyping()
	case ModeHelp:
		return m.viewHelp()
	case ModeResults:
		return m.viewResults()
	case ModeQuit:
		return m.viewQuit()
	default:
		return "Unknown mode"
	}
}

func (m *Model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case ModeTyping:
		return m.handleTypingKey(key)
	case ModeHelp:
		if key.String() == "esc" {
			m.mode = ModeTyping
		}
		return m, nil
	case ModeResults:
		if key.String() == "enter" {
			m.mode = ModeTyping
			return m, m.sess.Restart()
		}
		if key.String() == "esc" {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	case ModeQuit:
		if key.String() == "y" || key.String() == "Y" {
			m.quitting = true
			return m, tea.Quit
		}
		m.mode = ModeTyping
		return m, nil
	}
	return m, nil
}

func (m *Model) handleTypingKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle scrolling for code mode
	isCodeMode := strings.Contains(m.sess.GetMode(), "code") || m.sess.GetMode() == "snippet"
	if isCodeMode {
		switch key.Type {
		case tea.KeyUp:
			m.sess.ScrollUp()
			return m, nil
		case tea.KeyDown:
			m.sess.ScrollDown()
			return m, nil
		case tea.KeyPgUp:
			m.sess.ScrollUpPage()
			return m, nil
		case tea.KeyPgDown:
			m.sess.ScrollDownPage()
			return m, nil
		}
	}

	switch key.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "ctrl+q":
		m.mode = ModeQuit
		return m, nil
	case "ctrl+h":
		m.mode = ModeHelp
		return m, nil
	case "ctrl+w":
		m.sess.ToggleContext()
		return m, nil
	case "esc":
		return m, m.sess.Restart()
	default:
		return m, m.sess.HandleInput(key)
	}
}

func (m Model) viewTyping() string {
	content := m.sess.View(m.width, m.height)

	placedContent := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(m.config.Theme.Colors.Background)))

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Background(lipgloss.Color(m.config.Theme.Colors.Background)).
		Render(placedContent)
}

func (m Model) createStyledBox(content string, paddingX, paddingY int) string {
	styledContent := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.config.Theme.Colors.TextPrimary)).
		Background(lipgloss.Color(m.config.Theme.Colors.Background)).
		Render(content)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.config.Theme.Colors.TextPrimary)).
		BorderBackground(lipgloss.Color(m.config.Theme.Colors.Background)).
		Background(lipgloss.Color(m.config.Theme.Colors.Background)).
		Padding(paddingY, paddingX).
		Align(lipgloss.Center).
		Render(styledContent)

	placedBox := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(m.config.Theme.Colors.Background)))

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Background(lipgloss.Color(m.config.Theme.Colors.Background)).
		Render(placedBox)
}

func (m Model) viewHelp() string {
	helpText := "Help overlay - Press ESC to close\n\nShortcuts:\nCtrl+Q: Quit\nCtrl+C: Force quit\nEsc: Restart\nCtrl+H: Help\nCtrl+W: TTS\nBackspace: Delete\nLeft/Right: Navigate segments"
	return m.createStyledBox(helpText, 2, 1)
}

func (m Model) viewResults() string {
	calculator := session.NewResultsCalculator()
	results := calculator.CalculateResults(m.sess, m.sess.GetMode())

	content := fmt.Sprintf(`Results

WPM: %.1f
Accuracy: %.1f%%
CPM: %.1f
Duration: %.2fs
Mistakes: %d

Press Enter to restart or Esc to exit`, results.WPM, results.Accuracy, results.CPM, results.Duration.Seconds(), results.Mistakes)

	return m.createStyledBox(content, 4, 3)
}

func (m Model) viewQuit() string {
	quitText := `Are you sure you want to quit?

"Quit?" (y/n)`
	return m.createStyledBox(quitText, 4, 2)
}
