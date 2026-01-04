package session

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gti/src/internal"
	"gti/src/internal/config"
	"gti/src/internal/syntax"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type QuoteResponse struct {
	Q string `json:"q"`
	A string `json:"a"`
}

func FetchQuote(cfg *config.Config) string {
	q := FetchQuoteWithAuthor(cfg)
	return q.Text
}

func FetchQuoteWithAuthor(cfg *config.Config) Quote {
	client := &http.Client{
		Timeout: time.Duration(cfg.Network.TimeoutMs) * time.Millisecond,
	}

	resp, err := client.Get("https://zenquotes.io/api/random")
	if err != nil {
		return Quote{Text: config.DefaultPracticeText, Author: "Unknown"}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Quote{Text: config.DefaultPracticeText, Author: "Unknown"}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Quote{Text: config.DefaultPracticeText, Author: "Unknown"}
	}

	var quotes []QuoteResponse
	err = json.Unmarshal(body, &quotes)
	if err != nil {
		return Quote{Text: config.DefaultPracticeText, Author: "Unknown"}
	}

	if len(quotes) == 0 {
		return Quote{Text: config.DefaultPracticeText, Author: "Unknown"}
	}

	quote := quotes[0]
	if quote.Q == "" {
		return Quote{Text: config.DefaultPracticeText, Author: "Unknown"}
	}

	return Quote{Text: quote.Q, Author: quote.A}
}

func FetchMultipleQuotes(cfg *config.Config, count int) []Quote {
	if count <= 0 {
		count = 1
	}
	if count > 10 {
		count = 10
	}

	var quotes []Quote
	client := &http.Client{
		Timeout: time.Duration(cfg.Network.TimeoutMs) * time.Millisecond,
	}

	for i := 0; i < count; i++ {
		resp, err := client.Get("https://zenquotes.io/api/random")
		if err != nil {
			quotes = append(quotes, Quote{Text: config.DefaultPracticeText, Author: "Unknown"})
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			quotes = append(quotes, Quote{Text: config.DefaultPracticeText, Author: "Unknown"})
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			quotes = append(quotes, Quote{Text: config.DefaultPracticeText, Author: "Unknown"})
			continue
		}

		var quoteResponses []QuoteResponse
		err = json.Unmarshal(body, &quoteResponses)
		if err != nil || len(quoteResponses) == 0 {
			quotes = append(quotes, Quote{Text: config.DefaultPracticeText, Author: "Unknown"})
			continue
		}

		qr := quoteResponses[0]
		if qr.Q != "" {
			quotes = append(quotes, Quote{Text: qr.Q, Author: qr.A})
		} else {
			quotes = append(quotes, Quote{Text: config.DefaultPracticeText, Author: "Unknown"})
		}
	}

	if len(quotes) == 0 {
		return []Quote{{Text: config.DefaultPracticeText, Author: "Unknown"}}
	}

	return quotes
}

type Session struct {
	config                *config.Config
	mode                  string
	tier                  string
	text                  string
	author                string
	userInput             string
	position              int
	mistakes              int
	totalChars            int
	totalMistakes         int
	totalChunks           int
	maxChunks             int
	allChunks             []string
	chunkIndex            int
	isGroupMode           bool
	pageSize              int
	currentPageChunks     int
	startTime             time.Time
	duration              time.Duration
	timeLimit             time.Duration
	timer                 *time.Timer
	running               bool
	completed             bool
	layoutDirty           bool
	showContext           bool
	ttsUnavailableMessage string
	RemainingTimeDisplay  int
	ExternalMistakes      int

	// Scrolling support
	scrollOffset          int  // Current scroll position (line number)
	visibleLines          int  // Number of lines that can be displayed

	backspaceCount    int
	correctedErrors   int
	uncorrectedErrors int
	correctChars      int
	avgWordLength     float64
}

func NewSession(cfg *config.Config, mode string) *Session {
	var text string
	var timeLimit time.Duration
	switch mode {
	case "words":
		text = internal.GenerateWordsDynamic(10, cfg.Language.Default)
		timeLimit = time.Duration(60) * time.Second
	case "quote":
		text = FetchQuote(cfg)
	case "timed":
		text = internal.GenerateWordsDynamic(10, cfg.Language.Default)
		timeLimit = time.Duration(cfg.Timed.DefaultSeconds) * time.Second
	case "practice":
		text = internal.GenerateWordsDynamic(10, cfg.Language.Default)
	default:
		// Check if it's a code mode
		if strings.Contains(mode, "code") || mode == "snippet" {
			return NewSessionWithCodeSnippet(cfg, mode)
		}
		text = config.DefaultPracticeText
	}
	session := &Session{
		config:    cfg,
		mode:      mode,
		text:      text,
		timeLimit: timeLimit,
	}
	session.calculateAvgWordLength()
	return session
}

func NewSessionWithCustomText(cfg *config.Config, mode, file string, start int) *Session {
	paragraphs := loadParagraphs(file)
	text := getParagraphAtStart(paragraphs, start)

	return &Session{
		config:     cfg,
		mode:       mode,
		text:       text,
		allChunks:  paragraphs,
		chunkIndex: start - 1,
	}
}

func NewSessionWithTimed(cfg *config.Config, seconds int) *Session {
	return &Session{
		config:    cfg,
		mode:      "timed",
		text:      internal.GenerateWordsDynamic(10, cfg.Language.Default),
		timeLimit: time.Duration(seconds) * time.Second,
	}
}

func NewSessionWithCustomTimed(cfg *config.Config, file string, start int, seconds int) *Session {
	paragraphs := loadParagraphs(file)
	text := getParagraphAtStart(paragraphs, start)

	return &Session{
		config:     cfg,
		mode:       "custom-timed",
		text:       text,
		allChunks:  paragraphs,
		chunkIndex: start - 1,
		timeLimit:  time.Duration(seconds) * time.Second,
	}
}

func NewSessionWithChallenge(cfg *config.Config, tier string) *Session {
	return &Session{
		config: cfg,
		mode:   "challenge",
		tier:   tier,
	}
}

func NewSessionWithQuotes(cfg *config.Config, quoteList []Quote) *Session {
	if len(quoteList) == 0 {
		return &Session{
			config: cfg,
			mode:   "quote",
			text:   config.DefaultPracticeText,
			author: "Unknown",
		}
	}

	if len(quoteList) == 1 {
		return &Session{
			config: cfg,
			mode:   "quote",
			text:   quoteList[0].Text,
			author: quoteList[0].Author,
		}
	}

	var quoteTexts []string
	for _, q := range quoteList {
		quoteTexts = append(quoteTexts, q.Text)
	}

	return &Session{
		config:     cfg,
		mode:       "quotes",
		text:       quoteTexts[0],
		allChunks:  quoteTexts,
		chunkIndex: 0,
		author:     quoteList[0].Author,
	}
}

func NewSessionWithChunkLimit(cfg *config.Config, maxChunks int) *Session {
	var text string
	isGroupMode := maxChunks > 2
	pageSize := 3
	var currentPageChunks int
	if maxChunks <= 1 || !isGroupMode {
		text = internal.GenerateWordsDynamic(16, cfg.Language.Default)
		pageSize = 1
		currentPageChunks = 1
	} else {
		currentPageChunks = min(pageSize, maxChunks)
		var chunks []string
		for i := 0; i < currentPageChunks; i++ {
			chunks = append(chunks, internal.GenerateWordsDynamic(17, cfg.Language.Default))
		}
		text = strings.Join(chunks, "\n\n")
	}
	return &Session{
		config:            cfg,
		mode:              "practice",
		text:              text,
		maxChunks:         maxChunks,
		isGroupMode:       isGroupMode,
		pageSize:          pageSize,
		currentPageChunks: currentPageChunks,
	}
}

func NewSessionWithCodeSnippet(cfg *config.Config, mode string) *Session {
	// Extract language from mode (e.g., "go-code" -> "go")
	language := "go" // default
	if strings.Contains(mode, "python") {
		language = "python"
	} else if strings.Contains(mode, "javascript") {
		language = "javascript"
	} else if strings.Contains(mode, "java") {
		language = "java"
	} else if strings.Contains(mode, "cpp") {
		language = "cpp"
	} else if strings.Contains(mode, "rust") {
		language = "rust"
	} else if strings.Contains(mode, "typescript") {
		language = "typescript"
	}

	// Generate code snippet
	text := internal.GenerateCodeSnippet(language)

	return &Session{
		config: cfg,
		mode:   mode,
		text:   text,
	}
}

func NewSessionWithCodeSnippets(cfg *config.Config, language string, count int) *Session {
	if count <= 0 {
		count = 1
	}
	if count > 10 {
		count = 10
	}

	// Generate multiple code snippets
	text := internal.GenerateCodeSnippets(count, language)

	return &Session{
		config: cfg,
		mode:   language + "-code",
		text:   text,
	}
}

func NewSessionWithCodeSnippetTimed(cfg *config.Config, language string, seconds int) *Session {
	text := internal.GenerateCodeSnippet(language)

	return &Session{
		config:    cfg,
		mode:      language + "-code-timed",
		text:      text,
		timeLimit: time.Duration(seconds) * time.Second,
	}
}

func loadTextFromFile(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
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

func loadParagraphs(file string) []string {
	text, err := loadTextFromFile(file)
	if err != nil {
		text = config.DefaultPracticeText
	}
	return splitTextIntoParagraphs(text)
}

func getParagraphAtStart(paragraphs []string, start int) string {
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

	if (s.mode == "timed" || s.mode == "words" || (s.mode == "practice" && s.maxChunks == 0)) && s.position >= len(s.text) {
		s.totalChars += len(s.userInput)
		s.totalMistakes += s.mistakes

		s.text = internal.GenerateWordsDynamic(10, s.config.Language.Default)
		s.position = 0
		s.userInput = ""
		s.mistakes = 0
		s.layoutDirty = true
	}

	if s.mode == "practice" && s.maxChunks > 0 && s.position >= len(s.text) {
		if s.isGroupMode {
			s.totalChars += len(s.userInput)
			s.totalMistakes += s.mistakes
			s.totalChunks += s.currentPageChunks

			if s.totalChunks >= s.maxChunks {
				s.completed = true
				s.running = false
				s.duration = time.Since(s.startTime)

				record := &SessionRecord{
					Mode:              s.mode,
					Tier:              s.tier,
					TextLength:        len(s.text),
					DurationMs:        s.duration.Milliseconds(),
					WPM:               s.calculateWPM(),
					CPM:               s.calculateCPM(),
					Accuracy:          s.calculateAccuracy(),
					Mistakes:          s.totalMistakes,
					QuoteAuthor:       s.author,
					NetWPM:            CalculateNetWPM(s.totalChars+len(s.userInput), s.GetUncorrectedErrors(), s.duration),
					AdjustedWPM:       CalculateAdjustedWPM(s.GetCorrectChars(), s.GetAvgWordLength(), s.duration),
					CorrectedErrors:   s.GetCorrectedErrors(),
					UncorrectedErrors: s.GetUncorrectedErrors(),
					BackspaceCount:    s.GetBackspaceCount(),
					AvgWordLength:     s.GetAvgWordLength(),
				}
				SaveSessionRecord(s.config, record)
				s.mistakes = 0

				return func() tea.Msg { return SessionCompleteMsg{} }
			} else {
				s.currentPageChunks = min(s.pageSize, s.maxChunks-s.totalChunks)
				var chunks []string
				for i := 0; i < s.currentPageChunks; i++ {
					chunks = append(chunks, internal.GenerateWordsDynamic(10, s.config.Language.Default))
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
					s.completed = true
					s.running = false
					s.duration = time.Since(s.startTime)

					record := &SessionRecord{
						Mode:        s.mode,
						Tier:        s.tier,
						TextLength:  len(s.text),
						DurationMs:  s.duration.Milliseconds(),
						WPM:         s.calculateWPM(),
						CPM:         s.calculateCPM(),
						Accuracy:    s.calculateAccuracy(),
						Mistakes:    s.totalMistakes,
						QuoteAuthor: s.author,
					}
					SaveSessionRecord(s.config, record)

					return func() tea.Msg { return SessionCompleteMsg{} }
				} else {
					s.text = internal.GenerateWordsDynamic(10, s.config.Language.Default)
					s.position = 0
					s.userInput = ""
					s.mistakes = 0
					s.layoutDirty = true
				}
			}
	}

	if s.mode == "custom" && s.position >= len(s.text) {
		s.chunkIndex++
		s.totalChars += len(s.userInput)
		s.totalMistakes += s.mistakes

		if s.chunkIndex >= len(s.allChunks) {
			s.completed = true
			s.running = false
			s.duration = time.Since(s.startTime)

			record := &SessionRecord{
				Mode:              s.mode,
				Tier:              s.tier,
				TextLength:        len(s.text),
				DurationMs:        s.duration.Milliseconds(),
				WPM:               s.calculateWPM(),
				CPM:               s.calculateCPM(),
				Accuracy:          s.calculateAccuracy(),
				Mistakes:          s.totalMistakes,
				QuoteAuthor:       s.author,
				NetWPM:            CalculateNetWPM(s.totalChars+len(s.userInput), s.GetUncorrectedErrors(), s.duration),
				AdjustedWPM:       CalculateAdjustedWPM(s.GetCorrectChars(), s.GetAvgWordLength(), s.duration),
				CorrectedErrors:   s.GetCorrectedErrors(),
				UncorrectedErrors: s.GetUncorrectedErrors(),
				BackspaceCount:    s.GetBackspaceCount(),
				AvgWordLength:     s.GetAvgWordLength(),
			}
			SaveSessionRecord(s.config, record)

			return func() tea.Msg { return SessionCompleteMsg{} }
		} else {
			s.text = s.allChunks[s.chunkIndex]
			s.position = 0
			s.userInput = ""
			s.mistakes = 0
			s.layoutDirty = true
		}
	}

	if s.mode == "quotes" && s.position >= len(s.text) {
		s.chunkIndex++
		s.totalChars += len(s.userInput)
		s.totalMistakes += s.mistakes

		if s.chunkIndex >= len(s.allChunks) {
			s.completed = true
			s.running = false
			s.duration = time.Since(s.startTime)

			record := &SessionRecord{
				Mode:              s.mode,
				Tier:              s.tier,
				TextLength:        len(s.text),
				DurationMs:        s.duration.Milliseconds(),
				WPM:               s.calculateWPM(),
				CPM:               s.calculateCPM(),
				Accuracy:          s.calculateAccuracy(),
				Mistakes:          s.totalMistakes,
				QuoteAuthor:       s.author,
				NetWPM:            CalculateNetWPM(s.totalChars+len(s.userInput), s.GetUncorrectedErrors(), s.duration),
				AdjustedWPM:       CalculateAdjustedWPM(s.GetCorrectChars(), s.GetAvgWordLength(), s.duration),
				CorrectedErrors:   s.GetCorrectedErrors(),
				UncorrectedErrors: s.GetUncorrectedErrors(),
				BackspaceCount:    s.GetBackspaceCount(),
				AvgWordLength:     s.GetAvgWordLength(),
			}
			SaveSessionRecord(s.config, record)

			return func() tea.Msg { return SessionCompleteMsg{} }
		} else {
			s.text = s.allChunks[s.chunkIndex]
			s.position = 0
			s.userInput = ""
			s.mistakes = 0
			s.layoutDirty = true
		}
	}

	if s.mode != "timed" && s.mode != "words" && s.mode != "custom" && s.mode != "quotes" && s.position >= len(s.text) {
		s.completed = true
		s.running = false
		s.duration = time.Since(s.startTime)

		s.totalChars += len(s.userInput)
		s.totalMistakes += s.mistakes

		if s.mode != "challenge" {
			record := &SessionRecord{
				Mode:              s.mode,
				Tier:              s.tier,
				TextLength:        len(s.text),
				DurationMs:        s.duration.Milliseconds(),
				WPM:               s.calculateWPM(),
				CPM:               s.calculateCPM(),
				Accuracy:          s.calculateAccuracy(),
				Mistakes:          s.totalMistakes,
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

		return func() tea.Msg { return SessionCompleteMsg{} }
	}

	return nil
}

func (s *Session) completeSession() tea.Cmd {
	s.completed = true
	s.running = false
	s.duration = time.Since(s.startTime)
	if s.mode != "challenge" {
		record := &SessionRecord{
			Mode:              s.mode,
			Tier:              s.tier,
			TextLength:        len(s.text),
			DurationMs:        s.duration.Milliseconds(),
			WPM:               s.calculateWPM(),
			CPM:               s.calculateCPM(),
			Accuracy:          s.calculateAccuracy(),
			Mistakes:          s.totalMistakes,
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

			record := &SessionRecord{
				Mode:              s.mode,
				Tier:              s.tier,
				TextLength:        len(s.text),
				DurationMs:        s.duration.Milliseconds(),
				WPM:               s.calculateWPM(),
				CPM:               s.calculateCPM(),
				Accuracy:          s.calculateAccuracy(),
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
			s.mistakes = 0

			return func() tea.Msg { return SessionCompleteMsg{} }
		}
		return s.tickTimer()
	}
	return nil
}

func (s *Session) tickTimer() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
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
	wpm := s.calculateWPM()
	accuracy := s.calculateAccuracy()

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

	// Original word-based rendering for non-code modes
	wordStart, wordEnd := s.findCurrentWordBoundaries()

	var rendered strings.Builder
	for i, char := range s.text {
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
	lines := strings.Split(s.text, "\n")

	// Determine language from mode or use default
	language := "go" // default
	if strings.Contains(s.mode, "python") {
		language = "python"
	} else if strings.Contains(s.mode, "javascript") {
		language = "javascript"
	} else if strings.Contains(s.mode, "java") {
		language = "java"
	} else if strings.Contains(s.mode, "cpp") {
		language = "cpp"
	} else if strings.Contains(s.mode, "rust") {
		language = "rust"
	} else if strings.Contains(s.mode, "typescript") {
		language = "typescript"
	}

	// Apply syntax highlighting
	highlighter := syntax.NewHighlighter(s.config)
	highlightedText := highlighter.Highlight(s.text, language)

	highlightedLines := strings.Split(highlightedText, "\n")

	// Calculate line number width
	maxLineNum := len(lines)
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
			lineNum := strconv.Itoa(lineIdx + 1)
			lineNumPadded := fmt.Sprintf("%*s", lineNumWidth, lineNum)
			lineStr.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(s.config.Theme.Colors.TextSecondary)).
				Render(lineNumPadded + " "))
		}

		// Add the highlighted line content with character-level coloring
		if lineIdx < len(highlightedLines) {
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

	// Ensure current line is visible
	if currentLine < s.scrollOffset {
		s.scrollOffset = currentLine
	} else if currentLine >= s.scrollOffset+s.visibleLines {
		s.scrollOffset = currentLine - s.visibleLines + 1
	}

	// Ensure scroll offset is within bounds
	maxScroll := len(lines) - s.visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.scrollOffset > maxScroll {
		s.scrollOffset = maxScroll
	}
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
}

// ScrollUp scrolls up smoothly
func (s *Session) ScrollUp() {
	if s.scrollOffset > 0 {
		// Smooth scrolling - scroll by smaller increments
		scrollAmount := 1
		if s.visibleLines > 10 {
			scrollAmount = 1 // Keep it to 1 line for better control
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
	lines := strings.Split(s.text, "\n")
	maxScroll := len(lines) - s.visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.scrollOffset < maxScroll {
		// Smooth scrolling - scroll by smaller increments
		scrollAmount := 1
		if s.visibleLines > 10 {
			scrollAmount = 1 // Keep it to 1 line for better control
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
	scrollAmount := s.visibleLines - 1 // Keep one line of overlap
	if scrollAmount < 1 {
		scrollAmount = 1
	}
	s.scrollOffset -= scrollAmount
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
	s.layoutDirty = true
}

// ScrollDownPage scrolls down by one page
func (s *Session) ScrollDownPage() {
	lines := strings.Split(s.text, "\n")
	scrollAmount := s.visibleLines - 1 // Keep one line of overlap
	if scrollAmount < 1 {
		scrollAmount = 1
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
		s.visibleLines = textHeight
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

func (s *Session) calculateWPM() float64 {
	return s.CalculateWPM()
}

func (s *Session) calculateCPM() float64 {
	return s.CalculateCPM()
}

func (s *Session) calculateAccuracy() float64 {
	return s.CalculateAccuracy()
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
	s.text = text
	s.ResetForNewText()
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
		WPM:      s.calculateWPM(),
		CPM:      s.calculateCPM(),
		Accuracy: s.calculateAccuracy(),
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
