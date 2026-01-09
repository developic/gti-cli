package session

import (
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gti/src/internal"
	"gti/src/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	// Word and text generation constants
	DefaultWordCount         = 10
	MaxQuoteCount           = 10
	CharsPerWord            = 5.0
	DefaultTimedSeconds     = 60

	// Rendering constants
	RenderWindowSize        = 200
	ScrollSpeed             = 3
	MinScrollIncrement      = 1
	ScrollOverlap           = 1

	// UI width thresholds
	MinWidthNarrow          = 40
	MinWidthMedium          = 60
	MinWidthWide            = 80

	// Time thresholds
	MinValidDurationSeconds = 15
	MinValidTextLength      = 60

	// Percentage and calculation constants
	PercentDenominator      = 100.0
	WordLengthEstimate      = 5.5
)

var Tips = []string{
	"Keep your fingers on the home row (ASDF for left, JKL; for right)",
	"Look at the screen, not your keyboard while typing",
	"Use all your fingers, not just your index fingers",
	"Practice regularly to build muscle memory",
	"Focus on accuracy first, speed will come naturally",
	"Take breaks to avoid fatigue and maintain focus",
	"Breathe steadily and stay relaxed while typing",
	"Maintain proper posture with feet flat on the floor",
	"Keep your wrists straight and at a comfortable height",
	"Practice difficult letter combinations separately",
	"Use the correct finger for each key to avoid bad habits",
	"Start slow and gradually increase your typing speed",
	"Minimize mistakes by focusing on the next character",
	"Rest your hands when not typing to prevent strain",
	"Practice typing common words and phrases regularly",
}

type SessionCompleteMsg struct{}
type TimerTickMsg struct{}

type Quote struct {
	Text   string
	Author string
}

// Embedded structs for better organization
type SessionState struct {
	position            int
	mistakes            int
	totalChars          int
	totalMistakes       int
	totalChunks         int
	maxChunks           int
	chunkIndex          int
	isGroupMode         bool
	pageSize            int
	currentPageChunks   int
}

type Timing struct {
	startTime  time.Time
	duration   time.Duration
	timeLimit  time.Duration
	timer      *time.Timer
	running    bool
	completed  bool
}

type TextData struct {
	text       string
	author     string
	userInput  string
	allChunks  []string
}

type UIState struct {
	layoutDirty           bool
	showContext           bool
	ttsUnavailableMessage string
	RemainingTimeDisplay  int
	ExternalMistakes      int
}

type Scrolling struct {
	scrollOffset int
	visibleLines int
}

type Performance struct {
	cachedLines []string
	textHash    uint32
}

type Statistics struct {
	backspaceCount    int
	correctedErrors   int
	uncorrectedErrors int
	correctChars      int
	avgWordLength     float64
}

type SessionConfig struct {
	Mode         string
	Tier         string
	Text         string
	Author       string
	AllChunks    []string
	ChunkIndex   int
	MaxChunks    int
	TimeLimit    time.Duration
	QuoteList    []Quote
	Language     string
	CodeCount    int
	File         string
	Start        int
}

// NewSessionWithOptions creates a session using the unified SessionConfig
func NewSessionWithOptions(cfg *config.Config, sessionConfig SessionConfig) *Session {
	session := &Session{
		config: cfg,
		mode:   sessionConfig.Mode,
		tier:   sessionConfig.Tier,
	}

	// Set text and related fields based on configuration
	session.setTextFromConfig(sessionConfig)

	// Set timing
	if sessionConfig.TimeLimit > 0 {
		session.timeLimit = sessionConfig.TimeLimit
	}

	// Set chunk index if specified
	if sessionConfig.ChunkIndex != 0 {
		session.chunkIndex = sessionConfig.ChunkIndex
	}

	// Set all chunks if provided
	if len(sessionConfig.AllChunks) > 0 {
		session.allChunks = sessionConfig.AllChunks
	}

	session.calculateAvgWordLength()
	return session
}

// setTextFromConfig sets the text and related fields based on the session configuration
func (s *Session) setTextFromConfig(sessionConfig SessionConfig) {
	// Set text and related fields based on configuration
	if sessionConfig.Text != "" {
		s.text = sessionConfig.Text
		s.author = sessionConfig.Author
	} else if sessionConfig.File != "" {
		// Load from file
		if sessionConfig.Mode == "code" || sessionConfig.Mode == "custom-code" {
			if sessionConfig.Start == 1 {
				// For code mode with default start, load entire file as one snippet
				text, err := loadTextFromFile(sessionConfig.File)
				if err != nil {
					s.text = config.DefaultPracticeText
				} else {
					s.text = text
				}
			} else {
				// For code mode with custom start, load lines in chunks of 6 starting from specified chunk
				paragraphs := loadParagraphs(sessionConfig.File)
				linesPerChunk := 6
				chunkIndex := sessionConfig.Start - 1 // 0-based chunk index
				if chunkIndex < 0 {
					chunkIndex = 0
				}

				startLine := chunkIndex * linesPerChunk
				endLine := startLine + linesPerChunk
				if endLine > len(paragraphs) {
					endLine = len(paragraphs)
				}
				if startLine >= len(paragraphs) {
					startLine = len(paragraphs) - 1
					endLine = len(paragraphs)
				}

				// Collect lines for this chunk
				var selectedParagraphs []string
				for i := startLine; i < endLine; i++ {
					selectedParagraphs = append(selectedParagraphs, paragraphs[i])
				}

				s.text = strings.Join(selectedParagraphs, "\n")
				s.allChunks = paragraphs
				s.chunkIndex = startLine // Store the starting line index for line numbering
			}
		} else {
			// For other modes, split into paragraphs
			paragraphs := loadParagraphs(sessionConfig.File)
			s.text = getParagraphAtStart(paragraphs, sessionConfig.Start)
			s.allChunks = paragraphs
			s.chunkIndex = sessionConfig.Start - 1
		}
	} else if len(sessionConfig.QuoteList) > 0 {
		// Handle quotes
		if len(sessionConfig.QuoteList) == 1 {
			s.text = sessionConfig.QuoteList[0].Text
			s.author = sessionConfig.QuoteList[0].Author
		} else {
			var quoteTexts []string
			for _, q := range sessionConfig.QuoteList {
				quoteTexts = append(quoteTexts, q.Text)
			}
			s.text = quoteTexts[0]
			s.allChunks = quoteTexts
			s.author = sessionConfig.QuoteList[0].Author
			s.chunkIndex = 0
		}
	} else if sessionConfig.Language != "" && sessionConfig.CodeCount > 0 {
		// Generate multiple code snippets
		s.text = internal.GenerateCodeSnippets(sessionConfig.CodeCount, sessionConfig.Language)
	} else if sessionConfig.Language != "" {
		// Generate single code snippet
		s.text = internal.GenerateCodeSnippet(sessionConfig.Language)
		if !strings.Contains(sessionConfig.Mode, "code") {
			s.mode = sessionConfig.Language + "-code"
		}
		} else if sessionConfig.MaxChunks > 0 {
			// Practice mode with chunk limit
			isGroupMode := sessionConfig.MaxChunks > 2
			pageSize := 3
			var currentPageChunks int
			if sessionConfig.MaxChunks <= 1 || !isGroupMode {
				s.text = internal.GenerateWordsDynamic(16, s.config.Language.Default)
				pageSize = 1
				currentPageChunks = 1
			} else {
				currentPageChunks = min(pageSize, sessionConfig.MaxChunks)
				var chunks []string
				for i := 0; i < currentPageChunks; i++ {
					chunks = append(chunks, internal.GenerateWordsDynamic(17, s.config.Language.Default))
				}
				s.text = strings.Join(chunks, "\n\n")
			}
			s.maxChunks = sessionConfig.MaxChunks
			s.isGroupMode = isGroupMode
			s.pageSize = pageSize
			s.currentPageChunks = currentPageChunks
		} else {
			// Default text generation based on mode
			switch sessionConfig.Mode {
			case "words":
				s.text = internal.GenerateWordsDynamic(DefaultWordCount, s.config.Language.Default)
				s.timeLimit = time.Duration(DefaultTimedSeconds) * time.Second
			case "timed":
				s.text = internal.GenerateWordsDynamic(DefaultWordCount, s.config.Language.Default)
				s.timeLimit = time.Duration(s.config.Timed.DefaultSeconds) * time.Second
			case "practice":
				s.text = internal.GenerateWordsDynamic(DefaultWordCount, s.config.Language.Default)
			case "quote":
				s.text = config.DefaultPracticeText
			default:
				s.text = config.DefaultPracticeText
			}
		}
}

type Session struct {
	config *config.Config
	mode   string
	tier   string

	SessionState
	TextData
	Timing
	UIState
	Scrolling
	Performance
	Statistics
}

// saveRecord saves a session record with the given mistakes count
func (s *Session) saveRecord(mistakes int) {
	record := &SessionRecord{
		Mode:              s.mode,
		Tier:              s.tier,
		TextLength:        len(s.text),
		DurationMs:        s.duration.Milliseconds(),
		WPM:               s.CalculateWPM(),
		CPM:               s.CalculateCPM(),
		Accuracy:          s.CalculateAccuracy(),
		Mistakes:          mistakes,
		QuoteAuthor:       s.author,
		NetWPM:            CalculateNetWPM(s.totalChars+len(s.userInput), s.GetUncorrectedErrors(), s.duration),
		AdjustedWPM:       CalculateAdjustedWPM(s.GetCorrectChars(), s.GetAvgWordLength(), s.duration),
		CorrectedErrors:   s.GetCorrectedErrors(),
		UncorrectedErrors: s.GetUncorrectedErrors(),
		BackspaceCount:    s.GetBackspaceCount(),
		AvgWordLength:     s.GetAvgWordLength(),
	}
	SaveSessionRecord(s.config, record)
}

// Unified session creation with options pattern
func NewSession(cfg *config.Config, mode string, opts ...SessionOption) *Session {
	config := SessionConfig{Mode: mode}

	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	// Set defaults based on mode
	switch mode {
	case "words":
		if config.TimeLimit == 0 {
			config.TimeLimit = time.Duration(DefaultTimedSeconds) * time.Second
		}
	case "timed":
		if config.TimeLimit == 0 {
			config.TimeLimit = time.Duration(cfg.Timed.DefaultSeconds) * time.Second
		}
	case "quote":
		// Quote fetching should be handled at command level
	default:
		// Check if it's a code mode
		if strings.Contains(mode, "code") || mode == "snippet" {
			if config.Language == "" {
				config.Language = extractLanguageFromMode(mode)
			}
		}
	}

	return NewSessionWithOptions(cfg, config)
}

// SessionOption allows flexible session configuration
type SessionOption func(*SessionConfig)

// WithCustomText sets custom file and start position
func WithCustomText(file string, start int) SessionOption {
	return func(c *SessionConfig) {
		c.File = file
		c.Start = start
	}
}

// WithChallenge sets challenge tier
func WithChallenge(tier string) SessionOption {
	return func(c *SessionConfig) {
		c.Tier = tier
	}
}

// WithQuotes sets quote list
func WithQuotes(quoteList []Quote) SessionOption {
	return func(c *SessionConfig) {
		c.QuoteList = quoteList
	}
}

// WithChunkLimit sets maximum chunks for practice
func WithChunkLimit(maxChunks int) SessionOption {
	return func(c *SessionConfig) {
		c.MaxChunks = maxChunks
	}
}

// WithCodeLanguage sets programming language for code mode
func WithCodeLanguage(language string) SessionOption {
	return func(c *SessionConfig) {
		c.Language = language
	}
}

// WithCodeCount sets number of code snippets
func WithCodeCount(count int) SessionOption {
	return func(c *SessionConfig) {
		if count <= 0 {
			count = 1
		}
		if count > MaxQuoteCount {
			count = MaxQuoteCount
		}
		c.CodeCount = count
	}
}

// WithTimeLimit sets time limit
func WithTimeLimit(seconds int) SessionOption {
	return func(c *SessionConfig) {
		if seconds > 0 {
			c.TimeLimit = time.Duration(seconds) * time.Second
		}
	}
}

// WithText sets custom text directly
func WithText(text string, allChunks []string, chunkIndex int) SessionOption {
	return func(c *SessionConfig) {
		c.Text = text
		c.AllChunks = allChunks
		c.ChunkIndex = chunkIndex
	}
}

// extractLanguageFromMode extracts language from mode string (e.g., "go-code" -> "go")
func extractLanguageFromMode(mode string) string {
	languageMap := map[string]string{
		"python":     "python",
		"javascript": "javascript",
		"java":       "java",
		"cpp":        "cpp",
		"rust":       "rust",
		"typescript": "typescript",
	}

	for key, lang := range languageMap {
		if strings.Contains(mode, key) {
			return lang
		}
	}
	return "go" // default
}

// Legacy functions for backward compatibility
func NewSessionWithCustomText(cfg *config.Config, mode, file string, start int) *Session {
	return NewSession(cfg, mode, WithCustomText(file, start))
}

func NewSessionWithChallenge(cfg *config.Config, tier string) *Session {
	return NewSession(cfg, "challenge", WithChallenge(tier))
}

func NewSessionWithQuotes(cfg *config.Config, quoteList []Quote) *Session {
	return NewSession(cfg, "quotes", WithQuotes(quoteList))
}

func NewSessionWithChunkLimit(cfg *config.Config, maxChunks int) *Session {
	return NewSession(cfg, "practice", WithChunkLimit(maxChunks))
}

func NewSessionWithCodeSnippet(cfg *config.Config, mode string) *Session {
	return NewSession(cfg, "code", WithCodeLanguage(extractLanguageFromMode(mode)))
}

func NewSessionWithCodeSnippets(cfg *config.Config, language string, count int) *Session {
	return NewSession(cfg, "code", WithCodeLanguage(language), WithCodeCount(count))
}

func NewSessionWithCodeSnippetsTimed(cfg *config.Config, language string, count int, seconds int) *Session {
	return NewSession(cfg, "code", WithCodeLanguage(language), WithCodeCount(count), WithTimeLimit(seconds))
}

func NewSessionTimed(cfg *config.Config, mode string, text string, allChunks []string, chunkIndex int, seconds int) *Session {
	return NewSession(cfg, mode, WithText(text, allChunks, chunkIndex), WithTimeLimit(seconds))
}



func loadTextFromFile(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// loadFileContent provides unified file loading with fallback
func loadFileContent(filePath string, fallback string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fallback
	}
	return string(data)
}

func splitTextIntoChunks(text string, wordsPerChunk int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var chunks []string
	for i := 0; i < len(words); i += wordsPerChunk {
		end := i + wordsPerChunk
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)
	}
	return chunks
}

func splitTextIntoParagraphs(text string) []string {
	paragraphs := strings.Split(text, "\n\n")
	var result []string

	for _, para := range paragraphs {
		lines := strings.Split(strings.TrimSpace(para), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				result = append(result, line)
			}
		}
	}

	return result
}

func LoadParagraphs(file string) []string {
	text, err := loadTextFromFile(file)
	if err != nil {
		text = config.DefaultPracticeText
	}
	return splitTextIntoParagraphs(text)
}

func GetParagraphAtStart(paragraphs []string, start int) string {
	if len(paragraphs) == 0 {
		return config.DefaultPracticeText
	}
	startIndex := start - 1
	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex >= len(paragraphs) {
		startIndex = len(paragraphs) - 1
	}
	return paragraphs[startIndex]
}

func loadParagraphs(file string) []string {
	return LoadParagraphs(file)
}

func getParagraphAtStart(paragraphs []string, start int) string {
	return GetParagraphAtStart(paragraphs, start)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func speak(word string) {
	go func() {
		switch runtime.GOOS {
		case "linux":
			exec.Command("espeak", word).Run()
		case "darwin":
			exec.Command("say", word).Run()
		case "windows":
			exec.Command("powershell", "-c", "Add-Type -AssemblyName System.Speech; (New-Object System.Speech.Synthesis.SpeechSynthesizer).Speak('"+word+"')").Run()
		}
	}()
}

func ttsAvailable() bool {
	switch runtime.GOOS {
	case "linux":
		_, err := exec.LookPath("espeak")
		return err == nil
	case "darwin":
		_, err := exec.LookPath("say")
		return err == nil
	case "windows":
		return true
	default:
		return false
	}
}

func (s *Session) Start() tea.Cmd {
	s.startTime = time.Now()
	s.running = true
	return s.tickTimer()
}

func (s *Session) Restart() tea.Cmd {
	s.userInput = ""
	s.position = 0
	s.mistakes = 0
	s.totalMistakes = 0
	s.totalChars = 0
	s.totalChunks = 0
	s.chunkIndex = 0
	s.duration = 0
	s.completed = false
	return s.Start()
}

func (s *Session) ToggleContext() {
	if !s.showContext && !ttsAvailable() {
		s.ttsUnavailableMessage = "Linux users must install espeak-ng to use TTS."
		s.layoutDirty = true
		return
	}
	s.showContext = !s.showContext
	if s.showContext {
		next := s.getNextWord()
		if next != "" {
			speak(next)
		}
	}
	s.ttsUnavailableMessage = ""
	s.layoutDirty = true
}

func (s *Session) getNextWord() string {
	words := strings.Fields(s.text)
	if len(words) == 0 {
		return ""
	}

	charIndex := s.position
	wordIndex := 0
	currentCharCount := 0
	for _, word := range words {
		wordLen := len(word)
		if currentCharCount+wordLen >= charIndex {
			break
		}
		wordIndex++
		currentCharCount += wordLen + 1
	}
	if wordIndex+1 < len(words) {
		return words[wordIndex+1]
	}
	return ""
}

func (s *Session) HandleInput(key tea.KeyMsg) tea.Cmd {
	if !s.running || s.completed {
		return nil
	}

	switch key.Type {
	case tea.KeyBackspace:
		if len(s.userInput) > 0 {
			s.backspaceCount++
			removedChar := s.userInput[len(s.userInput)-1]
			s.userInput = s.userInput[:len(s.userInput)-1]
			if s.position > 0 {
				s.position--
				if removedChar != s.text[s.position] {
					s.correctedErrors++
					s.uncorrectedErrors--
				}
			}
		}
	default:
		char := key.String()
		if len(char) == 1 {
			s.userInput += char
			if s.position < len(s.text) {
				expectedChar := string(s.text[s.position])
				if char == expectedChar {
					s.correctChars++
				} else {
					s.mistakes++
					s.uncorrectedErrors++
				}
			}
			s.position++
			if char == " " && s.showContext {
				next := s.getNextWord()
				if next != "" {
					speak(next)
				}
			}
		}
	}

	// Update duration for smooth stats updates
	if s.running {
		s.duration = time.Since(s.startTime)
	}

	// Check for completion conditions
	if s.position >= len(s.text) {
		if s.mode == "practice" && s.maxChunks > 0 {
			return s.handlePracticeCompletion()
		} else if s.mode == "custom" || s.mode == "quotes" {
			return s.handleChunkCompletion()
		} else if s.mode == "timed" || s.mode == "words" || (s.mode == "practice" && s.maxChunks == 0) {
			s.handleContinuousCompletion()
		} else {
			return s.handleDefaultCompletion()
		}
	}

	return nil
}

// handleContinuousCompletion handles completion for modes that continue indefinitely
func (s *Session) handleContinuousCompletion() {
	s.totalChars += len(s.userInput)
	s.totalMistakes += s.mistakes

	s.text = internal.GenerateWordsDynamic(DefaultWordCount, s.config.Language.Default)
	s.invalidateLineCache()
	s.position = 0
	s.userInput = ""
	s.mistakes = 0
	s.layoutDirty = true
}

// handlePracticeCompletion handles completion for practice mode with chunk limits
func (s *Session) handlePracticeCompletion() tea.Cmd {
	if s.isGroupMode {
		s.totalChars += len(s.userInput)
		s.totalMistakes += s.mistakes
		s.totalChunks += s.currentPageChunks

		if s.totalChunks >= s.maxChunks {
			return s.completeSession()
		} else {
			s.currentPageChunks = min(s.pageSize, s.maxChunks-s.totalChunks)
			var chunks []string
			for i := 0; i < s.currentPageChunks; i++ {
				chunks = append(chunks, internal.GenerateWordsDynamic(DefaultWordCount, s.config.Language.Default))
			}
			s.text = strings.Join(chunks, "\n\n")
			s.position = 0
			s.userInput = ""
			s.mistakes = 0
			s.layoutDirty = true
		}
	} else {
		s.totalChunks++
		s.totalChars += len(s.userInput)
		s.totalMistakes += s.mistakes

		if s.totalChunks >= s.maxChunks {
			return s.completeSession()
		} else {
			s.text = internal.GenerateWordsDynamic(DefaultWordCount, s.config.Language.Default)
			s.position = 0
			s.userInput = ""
			s.mistakes = 0
			s.layoutDirty = true
		}
	}
	return nil
}

// handleChunkCompletion handles completion for custom and quotes modes
func (s *Session) handleChunkCompletion() tea.Cmd {
	s.chunkIndex++
	s.totalChars += len(s.userInput)
	s.totalMistakes += s.mistakes

	if s.chunkIndex >= len(s.allChunks) {
		return s.completeSession()
	} else {
		s.text = s.allChunks[s.chunkIndex]
		s.invalidateLineCache()
		s.position = 0
		s.userInput = ""
		s.mistakes = 0
		s.layoutDirty = true
	}
	return nil
}

// handleDefaultCompletion handles completion for all other modes
func (s *Session) handleDefaultCompletion() tea.Cmd {
	s.totalChars += len(s.userInput)
	s.totalMistakes += s.mistakes
	return s.completeSession()
}

func (s *Session) completeSession() tea.Cmd {
	s.completed = true
	s.running = false
	s.duration = time.Since(s.startTime)
	if s.mode != "challenge" {
		s.saveRecord(s.totalMistakes)
	}
	s.mistakes = 0
	return func() tea.Msg { return SessionCompleteMsg{} }
}

func (s *Session) UpdateTimer() tea.Cmd {
	if s.running {
		s.duration = time.Since(s.startTime)
		if s.timeLimit > 0 && s.duration >= s.timeLimit {
			s.completed = true
			s.running = false
			s.duration = s.timeLimit

			mistakes := s.mistakes
			if s.mode == "challenge" {
				mistakes = s.totalMistakes
			}
			if s.mode == "timed" || s.mode == "words" || s.mode == "practice" {
				mistakes = s.totalMistakes + s.mistakes
			}

			s.saveRecord(mistakes)
			s.mistakes = 0

			return func() tea.Msg { return SessionCompleteMsg{} }
		}



		return s.tickTimer()
	}
	return nil
}

func (s *Session) tickTimer() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TimerTickMsg{}
	})
}

func (s *Session) View(width, height int) string {
	status := s.renderStatus(width)
	textArea := s.renderText(width, height)
	tipOrContext := s.renderTip(width)
	if s.showContext {
		tipOrContext = s.renderContext(width)
	}
	hint := s.renderHint(width)

	var content string
	if height >= 6 {

		content = lipgloss.JoinVertical(lipgloss.Left, status, textArea, tipOrContext, hint)
	} else if height >= 4 {

		content = lipgloss.JoinVertical(lipgloss.Left, status, textArea, hint)
	} else if height >= 2 {

		content = lipgloss.JoinVertical(lipgloss.Left, textArea, hint)
	} else {

		content = textArea
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Background(lipgloss.Color(s.config.Theme.Colors.Background)).
		Render(content)
}

func (s *Session) calculateProgress() float64 {
	if s.isGroupMode {
		return float64(s.totalChunks)/float64(s.maxChunks)*100 + float64(s.position)/float64(len(s.text))*float64(s.currentPageChunks)/float64(s.maxChunks)*100
	} else if s.maxChunks > 0 {
		completedChunks := s.totalChunks
		currentProgress := float64(s.position) / float64(len(s.text))
		return (float64(completedChunks) + currentProgress) / float64(s.maxChunks) * 100
	} else if len(s.allChunks) > 0 {
		completedChunks := s.chunkIndex
		currentProgress := float64(s.position) / float64(len(s.text))
		return (float64(completedChunks) + currentProgress) / float64(len(s.allChunks)) * 100
	} else {
		return float64(s.position) / float64(len(s.text)) * 100
	}
}

func (s *Session) renderStatus(width int) string {
	mode := strings.Title(s.mode)
	if s.tier != "" {
		mode += " (" + s.tier + ")"
	}
	timer := "00:00"
	if s.running {
		if s.mode == "challenge" {
			timer = fmt.Sprintf("%ds", s.RemainingTimeDisplay)
		} else {
			elapsed := time.Since(s.startTime)
			timer = elapsed.Truncate(time.Second).String()
		}
	}
	wpm := s.CalculateWPM()
	accuracy := s.CalculateAccuracy()

	mistakes := s.mistakes
	if s.mode == "practice" && s.maxChunks > 0 {
		mistakes = s.totalMistakes + s.mistakes
	} else if s.mode == "challenge" && s.ExternalMistakes > 0 {
		mistakes = s.ExternalMistakes + s.mistakes
	}

	progress := s.calculateProgress()

	var statusText string
	if width >= 80 {

		statusText = fmt.Sprintf("Mode: %s | Timer: %s | WPM: %.1f | Accuracy: %.1f%% | Mistakes: %d | Progress: %.1f%%", mode, timer, wpm, accuracy, mistakes, progress)
	} else if width >= 60 {

		statusText = fmt.Sprintf("%s | %s | %.1f WPM | %.1f%% | %d mistakes", mode, timer, wpm, accuracy, mistakes)
	} else if width >= 40 {

		statusText = fmt.Sprintf("%s | %s | %.1f WPM | %d errors", mode, timer, wpm, mistakes)
	} else {

		statusText = fmt.Sprintf("%s | %.1f WPM", mode, wpm)
	}

	status := lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.config.Theme.Colors.TextPrimary)).
		Background(lipgloss.Color(s.config.Theme.Colors.StatusBar)).
		Width(width).
		Align(lipgloss.Center).
		Render(statusText)

	return status
}

func (s *Session) calculateDynamicWidth(content string, terminalWidth int) int {
	actualWidth := lipgloss.Width(content)
	contentWidth := actualWidth + 2
	minWidth := min(40, terminalWidth-4)
	maxWidth := min(80, terminalWidth-4)
	finalWidth := contentWidth
	if finalWidth < minWidth {
		finalWidth = minWidth
	}
	if finalWidth > maxWidth {
		finalWidth = maxWidth
	}
	if (terminalWidth-finalWidth)%2 != 0 {
		finalWidth--
	}
	return finalWidth
}

func (s *Session) findCurrentWordBoundaries() (int, int) {
	if s.position >= len(s.text) || s.text[s.position] == ' ' {
		return -1, -1
	}
	start := s.position
	for start > 0 && s.text[start-1] != ' ' {
		start--
	}
	end := s.position
	for end < len(s.text) && s.text[end] != ' ' {
		end++
	}
	return start, end - 1
}

func (s *Session) renderTextContent() string {
	// Check if this is code mode
	isCodeMode := strings.Contains(s.mode, "code") || s.mode == "snippet"

	if isCodeMode {
		return s.renderCodeContent()
	}

	// Optimized rendering: only render a window around current position for performance
	windowSize := RenderWindowSize // characters before and after current position
	start := max(0, s.position-windowSize)
	end := min(len(s.text), s.position+windowSize)

	// Original word-based rendering for non-code modes
	wordStart, wordEnd := s.findCurrentWordBoundaries()

	var rendered strings.Builder
	for i := start; i < end; i++ {
		char := rune(s.text[i])
		style := lipgloss.NewStyle().Background(lipgloss.Color(s.config.Theme.Colors.Background))
		if i < s.position {
			if i < len(s.userInput) && rune(s.userInput[i]) == char {
				style = style.Foreground(lipgloss.Color(s.config.Theme.Colors.Correct))
			} else {
				style = style.Foreground(lipgloss.Color(s.config.Theme.Colors.Incorrect))
			}
		} else if i == s.position {
			style = style.Foreground(lipgloss.Color(s.config.Theme.Colors.WordHighlight)).Faint(true)
			if s.config.Theme.Styles.UnderlineCurrent {
				style = style.Underline(true)
			}
		} else {
			if i >= wordStart && i <= wordEnd {
				style = style.Foreground(lipgloss.Color(s.config.Theme.Colors.WordHighlight))
			} else {
				style = style.Foreground(lipgloss.Color(s.config.Theme.Colors.Pending))
				if s.config.Theme.Styles.DimPending {
					style = style.Faint(true)
				}
			}
		}
		rendered.WriteString(style.Render(string(char)))
	}

	return rendered.String()
}

func (s *Session) renderCodeContent() string {
	lines := s.getCachedLines()

	// Calculate line number width - use absolute line numbers for custom code
	var maxLineNum int
	var lineNumOffset int
	if s.chunkIndex >= 0 && strings.Contains(s.mode, "code") && s.allChunks != nil {
		// For custom code with start parameter, show absolute line numbers
		lineNumOffset = s.chunkIndex + 1 // chunkIndex is 0-based paragraph index, +1 for 1-based line numbers
		maxLineNum = s.chunkIndex + len(lines)
	} else {
		// Default behavior for generated code
		lineNumOffset = 1
		maxLineNum = len(lines)
	}
	lineNumWidth := len(strconv.Itoa(maxLineNum))

	// Auto-scroll to keep current position visible
	s.autoScrollToCurrentPosition(lines)

	// Determine visible lines based on scrolling
	startLine := s.scrollOffset
	endLine := startLine + s.visibleLines
	if endLine > len(lines) {
		endLine = len(lines)
	}

	var renderedLines []string

	// Track global character position
	globalPos := 0
	for i := 0; i < startLine; i++ {
		globalPos += len(lines[i]) + 1 // +1 for newline
	}

	for lineIdx := startLine; lineIdx < endLine && lineIdx < len(lines); lineIdx++ {
		line := lines[lineIdx]
		var lineStr strings.Builder

		// Add line number if enabled
		showLineNumbers := true // This should come from config
		if showLineNumbers {
			lineNum := strconv.Itoa(lineIdx + lineNumOffset)
			lineNumPadded := fmt.Sprintf("%*s", lineNumWidth, lineNum)
			lineStr.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(s.config.Theme.Colors.TextSecondary)).
				Render(lineNumPadded + " "))
		}

		// Apply character-level typing colors
		for charIdx, char := range line {
			currentGlobalPos := globalPos + charIdx

			style := lipgloss.NewStyle().Background(lipgloss.Color(s.config.Theme.Colors.Background))

			if currentGlobalPos < s.position {
				if currentGlobalPos < len(s.userInput) && rune(s.userInput[currentGlobalPos]) == char {
					style = style.Foreground(lipgloss.Color(s.config.Theme.Colors.Correct))
				} else {
					style = style.Foreground(lipgloss.Color(s.config.Theme.Colors.Incorrect))
				}
			} else if currentGlobalPos == s.position {
				style = style.Foreground(lipgloss.Color(s.config.Theme.Colors.WordHighlight)).Faint(true)
				if s.config.Theme.Styles.UnderlineCurrent {
					style = style.Underline(true)
				}
			} else {
				style = style.Foreground(lipgloss.Color(s.config.Theme.Colors.Pending))
				if s.config.Theme.Styles.DimPending {
					style = style.Faint(true)
				}
			}

			lineStr.WriteString(style.Render(string(char)))
		}

		renderedLines = append(renderedLines, lineStr.String())
		globalPos += len(line) + 1 // +1 for newline
	}

	return strings.Join(renderedLines, "\n")
}

// autoScrollToCurrentPosition automatically scrolls to keep the current typing position visible
func (s *Session) autoScrollToCurrentPosition(lines []string) {
	if len(lines) == 0 || s.visibleLines <= 0 {
		return
	}

	// Find which line contains the current position
	currentLine := 0
	charCount := 0
	for i, line := range lines {
		lineLen := len(line) + 1 // +1 for newline
		if charCount+lineLen > s.position {
			currentLine = i
			break
		}
		charCount += lineLen
	}

	// Calculate target scroll position to keep current line visible
	var targetScroll int
	if currentLine < s.scrollOffset {
		targetScroll = currentLine
	} else if currentLine >= s.scrollOffset+s.visibleLines {
		targetScroll = currentLine - s.visibleLines + 1
	} else {
		return // Already visible, no need to scroll
	}

	// Ensure target scroll offset is within bounds
	maxScroll := len(lines) - s.visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if targetScroll > maxScroll {
		targetScroll = maxScroll
	}
	if targetScroll < 0 {
		targetScroll = 0
	}

	// Set scroll position directly (no smooth animation)
	if targetScroll != s.scrollOffset {
		s.scrollOffset = targetScroll
		s.layoutDirty = true
	}
}

// ScrollUp scrolls up smoothly
func (s *Session) ScrollUp() {
	if s.scrollOffset > 0 {
		// Smooth scrolling - scroll by smaller increments
		scrollAmount := MinScrollIncrement
		if s.visibleLines > 10 {
			scrollAmount = MinScrollIncrement // Keep it to 1 line for better control
		}
		s.scrollOffset -= scrollAmount
		if s.scrollOffset < 0 {
			s.scrollOffset = 0
		}
		s.layoutDirty = true
	}
}

// ScrollDown scrolls down smoothly
func (s *Session) ScrollDown() {
	lines := s.getCachedLines()
	maxScroll := len(lines) - s.visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.scrollOffset < maxScroll {
		// Smooth scrolling - scroll by smaller increments
		scrollAmount := MinScrollIncrement
		if s.visibleLines > 10 {
			scrollAmount = MinScrollIncrement // Keep it to 1 line for better control
		}
		s.scrollOffset += scrollAmount
		if s.scrollOffset > maxScroll {
			s.scrollOffset = maxScroll
		}
		s.layoutDirty = true
	}
}

// ScrollUpPage scrolls up by one page
func (s *Session) ScrollUpPage() {
	scrollAmount := s.visibleLines - ScrollOverlap // Keep one line of overlap
	if scrollAmount < MinScrollIncrement {
		scrollAmount = MinScrollIncrement
	}
	s.scrollOffset -= scrollAmount
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
	s.layoutDirty = true
}

// ScrollDownPage scrolls down by one page
func (s *Session) ScrollDownPage() {
	lines := s.getCachedLines()
	scrollAmount := s.visibleLines - ScrollOverlap // Keep one line of overlap
	if scrollAmount < MinScrollIncrement {
		scrollAmount = MinScrollIncrement
	}
	maxScroll := len(lines) - s.visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	s.scrollOffset += scrollAmount
	if s.scrollOffset > maxScroll {
		s.scrollOffset = maxScroll
	}
	s.layoutDirty = true
}

func (s *Session) renderText(width, height int) string {
	content := s.renderTextContent()

	var textHeight int
	if height >= 6 {
		textHeight = height - 4
	} else if height >= 4 {
		textHeight = height - 3
	} else if height >= 2 {
		textHeight = height - 2
	} else {
		textHeight = height
	}

	if textHeight < 1 {
		textHeight = 1
	}

	// Update visible lines for scrolling (only for code mode)
	isCodeMode := strings.Contains(s.mode, "code") || s.mode == "snippet"
	if isCodeMode {
		// Limit visible lines to 6 for better UX with long code files
		maxVisibleLines := 6
		if textHeight < maxVisibleLines {
			s.visibleLines = textHeight
		} else {
			s.visibleLines = maxVisibleLines
		}
	}

	dynamicWidth := s.calculateDynamicWidth(content, width)
	styledContent := lipgloss.NewStyle().
		Width(dynamicWidth).
		Background(lipgloss.Color(s.config.Theme.Colors.Background)).
		Render(content)
	return lipgloss.Place(width, textHeight, lipgloss.Center, lipgloss.Center, styledContent,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(s.config.Theme.Colors.Background)))
}

func (s *Session) renderCenteredText(text string, fgColor string, width int) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(fgColor)).
		Background(lipgloss.Color(s.config.Theme.Colors.Background)).
		Width(width).
		Align(lipgloss.Center).
		Render(text)
}

func (s *Session) renderTip(width int) string {
	if s.ttsUnavailableMessage != "" {
		return s.renderCenteredText(s.ttsUnavailableMessage, s.config.Theme.Colors.TextPrimary, width)
	}

	if len(Tips) == 0 {
		return ""
	}

	tipIndex := int(s.position) % len(Tips)
	tip := "💡 " + Tips[tipIndex]

	return s.renderCenteredText(tip, s.config.Theme.Colors.Accent, width)
}

func (s *Session) renderHint(width int) string {
	isCodeMode := strings.Contains(s.mode, "code") || s.mode == "snippet"
	var hint string
	if isCodeMode {
		hint = "↑↓: Scroll | PgUp/PgDn: Page | Esc: Restart | Ctrl+H: Help | Ctrl+Q: Quit"
	} else {
		hint = "Esc: Restart | Ctrl+H: Help | Ctrl+W: TTS | Ctrl+Q: Quit"
	}
	return s.renderCenteredText(hint, s.config.Theme.Colors.TextSecondary, width)
}

func (s *Session) renderContext(width int) string {
	return ""
}

func (s *Session) GetResults() string {
	calculator := NewResultsCalculator()
	results := calculator.CalculateResults(s, s.GetMode())

	return fmt.Sprintf("Results\n\nWPM: %.1f\nCPM: %.1f\nAccuracy: %.1f%%\nDuration: %.2fs\nMistakes: %d\n\nPress Enter or Esc to exit", results.WPM, results.CPM, results.Accuracy, results.Duration.Seconds(), results.Mistakes)
}

func (s *Session) ViewTextOnly(width, height int) string {
	content := s.renderTextContent()
	textHeight := height - 2

	paddedContent := lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2).
		Width(width - 4).
		Render(content)

	return lipgloss.Place(width, textHeight, lipgloss.Center, lipgloss.Top, paddedContent,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(s.config.Theme.Colors.Background)))
}

func (s *Session) CalculateWPM() float64 {
	if s.duration == 0 {
		return 0
	}
	minutes := s.duration.Minutes()
	totalChars := s.totalChars + len(s.userInput)
	words := float64(totalChars) / 5.0
	return words / minutes
}

func (s *Session) CalculateCPM() float64 {
	if s.duration == 0 {
		return 0
	}
	minutes := s.duration.Minutes()
	return float64(s.totalChars+len(s.userInput)) / minutes
}

func (s *Session) CalculateAccuracy() float64 {
	totalChars := s.totalChars + len(s.userInput)
	totalMistakes := s.totalMistakes + s.mistakes

	if totalChars == 0 {
		return 100.0
	}
	return float64(totalChars-totalMistakes) / float64(totalChars) * 100
}



func (s *Session) IsLayoutDirty() bool {
	return s.layoutDirty
}

func (s *Session) MarkLayoutDirty() {
	s.layoutDirty = true
}

func (s *Session) ClearLayoutDirty() {
	s.layoutDirty = false
}

func (s *Session) CursorIndex() int {
	return s.position
}

func (s *Session) TypedText() string {
	return s.userInput
}

func (s *Session) GetText() string {
	return s.text
}

func (s *Session) SetText(text string) {
	if s.text != text {
		s.text = text
		s.invalidateLineCache()
	}
	s.ResetForNewText()
}

// getCachedLines returns cached lines, computing them if necessary
func (s *Session) getCachedLines() []string {
	currentHash := s.computeTextHash()
	if s.textHash != currentHash || s.cachedLines == nil {
		s.cachedLines = strings.Split(s.text, "\n")
		s.textHash = currentHash
	}
	return s.cachedLines
}

// computeTextHash computes a simple hash of the text for cache invalidation
func (s *Session) computeTextHash() uint32 {
	h := fnv.New32a()
	h.Write([]byte(s.text))
	return h.Sum32()
}

// invalidateLineCache clears the cached lines
func (s *Session) invalidateLineCache() {
	s.cachedLines = nil
	s.textHash = 0
}

func (s *Session) GetMistakes() int {
	return s.mistakes
}

func (s *Session) GetTotalMistakes() int {
	return s.totalMistakes
}

func (s *Session) GetTotalChars() int {
	return s.totalChars
}

func (s *Session) GetDuration() time.Duration {
	return s.duration
}

func (s *Session) GetMode() string {
	return s.mode
}

func (s *Session) GetTier() string {
	return s.tier
}

func (s *Session) SetTier(tier string) {
	s.tier = tier
}

func (s *Session) ResetForNewText() {
	s.position = 0
	s.userInput = ""
	s.mistakes = 0
	s.completed = false
	s.layoutDirty = true
}

type StatsSnapshot struct {
	WPM      float64
	CPM      float64
	Accuracy float64
	Mistakes int
	Duration time.Duration
	Progress float64
}

func (s *Session) GetStatsSnapshot() StatsSnapshot {
	return StatsSnapshot{
		WPM:      s.CalculateWPM(),
		CPM:      s.CalculateCPM(),
		Accuracy: s.CalculateAccuracy(),
		Mistakes: s.mistakes,
		Duration: s.duration,
		Progress: s.calculateProgress(),
	}
}

func (s *Session) calculateAvgWordLength() {
	words := strings.Fields(s.text)
	if len(words) == 0 {
		s.avgWordLength = 5.0
		return
	}

	totalChars := 0
	for _, word := range words {
		totalChars += len(word)
	}
	s.avgWordLength = float64(totalChars) / float64(len(words))
}

func (s *Session) GetBackspaceCount() int {
	return s.backspaceCount
}

func (s *Session) GetCorrectedErrors() int {
	return s.correctedErrors
}

func (s *Session) GetUncorrectedErrors() int {
	return s.uncorrectedErrors
}

func (s *Session) GetCorrectChars() int {
	return s.correctChars
}

func (s *Session) GetAvgWordLength() float64 {
	return s.avgWordLength
}
