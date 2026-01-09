package tui

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gti/src/internal/config"
	"gti/src/internal/session"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	minValidDuration   = 15 * time.Second
	minValidTextLength = 60

	recentSessionsCount       = 5
	minSessionsForVariance    = 3
	minSessionsForImprovement = 10

	lowAccuracyThreshold            = 85.0
	highVarianceThreshold           = 25.0
	goodVarianceThreshold           = 12.0
	significantImprovementThreshold = 10.0

	defaultLineWidth           = 79
	achievementBarWidth        = 24
	recentSessionsDisplayLimit = 8
	trendChartSessionCount     = 20
	trendChartBarMaxWidth      = 40

	firstStepsSessions     = 1
	gettingStartedSessions = 10
	dedicatedSessions      = 50
	committedSessions      = 100

	speedThreshold1 = 30.0
	speedThreshold2 = 50.0
	speedThreshold3 = 70.0
	speedThreshold4 = 100.0

	accuracyThreshold1 = 95.0
	accuracyThreshold2 = 98.0
	accuracyThreshold3 = 99.0

	timeThreshold1 = time.Hour
	timeThreshold2 = 24 * time.Hour

	streakThreshold1       = 3
	streakThreshold2       = 7
	streakThreshold3       = 14
	longestStreakThreshold = 30

	consistencyVarianceThreshold = 10.0
)

type StatisticsView string

const (
	ViewAllTime StatisticsView = "all-time"
	ViewWeekly  StatisticsView = "weekly"
	ViewDaily   StatisticsView = "daily"
	ViewSession StatisticsView = "session"
)

type StatisticsModel struct {
	config   *config.Config
	view     StatisticsView
	records  []*session.SessionRecord
	stats    *Statistics
	width    int
	height   int
	quitting bool

	cachedView            StatisticsView
	cachedFilteredRecords []*session.SessionRecord
	cachedFilteredStats   *Statistics

	viewport viewport.Model

	styles statsStyles
}

type Statistics struct {
	TotalSessions   int
	TotalTime       time.Duration
	RawAvgWPM       float64
	RawPeakWPM      float64
	RawAvgAccuracy  float64
	RawBestAccuracy float64
	AvgMistakes     float64
	BackspaceRate   float64

	ValidSessions        []*session.SessionRecord
	NormalizedAvgWPM     float64
	NormalizedPeakWPM    float64
	RecentValidAvgWPM    float64
	RecentValidCountUsed int

	NetAvgWPM            float64
	NetPeakWPM           float64
	AdjustedAvgWPM       float64
	AdjustedPeakWPM      float64
	AvgCorrectedErrors   float64
	AvgUncorrectedErrors float64

	ConsistencyScore float64
	ImprovementRate  float64
	VariancePercent  float64
	OutlierCount     int

	CurrentStreak int
	LongestStreak int
}

type statsStyles struct {
	base      lipgloss.Style
	title     lipgloss.Style
	section   lipgloss.Style
	subtle    lipgloss.Style
	key       lipgloss.Style
	val       lipgloss.Style
	bad       lipgloss.Style
	good      lipgloss.Style
	accent    lipgloss.Style
	viewOn    lipgloss.Style
	viewOff   lipgloss.Style
	box       lipgloss.Style
	footer    lipgloss.Style
	monoWidth int
}

// StyleFactory creates lipgloss styles with configurable parameters
type StyleFactory struct {
	colors config.ThemeColorsConfig
}

// NewStyleFactory creates a new style factory with the given color configuration
func NewStyleFactory(colors config.ThemeColorsConfig) *StyleFactory {
	return &StyleFactory{colors: colors}
}

// CreateStyle creates a lipgloss style with the specified configuration
func (f *StyleFactory) CreateStyle(config StyleConfig) lipgloss.Style {
	style := lipgloss.NewStyle()

	if config.Foreground != "" {
		style = style.Foreground(lipgloss.Color(f.getColor(config.Foreground)))
	}

	if config.Background != "" {
		style = style.Background(lipgloss.Color(f.getColor(config.Background)))
	}

	if config.Bold {
		style = style.Bold(true)
	}

	if config.Border != "" {
		style = style.Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(f.getColor(config.Border)))
	}

	if config.PaddingX > 0 || config.PaddingY > 0 {
		style = style.Padding(config.PaddingY, config.PaddingX)
	}

	return style
}

// StyleConfig defines the configuration for creating a lipgloss style
type StyleConfig struct {
	Foreground string
	Background string
	Bold       bool
	Border     string
	PaddingX   int
	PaddingY   int
}

// getColor maps color names to actual color values from the theme
func (f *StyleFactory) getColor(name string) string {
	switch name {
	case "accent":
		return f.colors.Accent
	case "textPrimary":
		return f.colors.TextPrimary
	case "textSecondary":
		return f.colors.TextSecondary
	case "correct":
		return f.colors.Correct
	case "incorrect":
		return f.colors.Incorrect
	case "background":
		return f.colors.Background
	case "border":
		return f.colors.Border
	default:
		return f.colors.TextPrimary
	}
}

func NewStatisticsModel(cfg *config.Config) StatisticsModel {
	records, _ := session.LoadSessionRecords(cfg)

	m := StatisticsModel{
		config:  cfg,
		view:    ViewAllTime,
		records: records,
		stats:   calculateStatistics(records),
	}
	m.styles = newStatsStyles(cfg)

	m.viewport = viewport.New(80, 20)
	m.viewport.SetContent(m.renderScrollableContent())

	return m
}

func newStatsStyles(cfg *config.Config) statsStyles {
	factory := NewStyleFactory(cfg.Theme.Colors)

	return statsStyles{
		base:    lipgloss.NewStyle(),
		title:   factory.CreateStyle(StyleConfig{Foreground: "accent", Bold: true}),
		section: factory.CreateStyle(StyleConfig{Foreground: "accent", Bold: true}),
		subtle:  factory.CreateStyle(StyleConfig{Foreground: "textSecondary"}),
		key:     factory.CreateStyle(StyleConfig{Foreground: "textPrimary"}),
		val:     factory.CreateStyle(StyleConfig{Foreground: "textPrimary"}),
		good:    factory.CreateStyle(StyleConfig{Foreground: "correct"}),
		bad:     factory.CreateStyle(StyleConfig{Foreground: "incorrect"}),
		accent:  factory.CreateStyle(StyleConfig{Foreground: "accent", Bold: true}),
		viewOn:  factory.CreateStyle(StyleConfig{Foreground: "background", Background: "accent", Bold: true, PaddingX: 1}),
		viewOff: factory.CreateStyle(StyleConfig{Foreground: "textSecondary", Background: "border", PaddingX: 1}),
		box:     factory.CreateStyle(StyleConfig{Border: "border", PaddingX: 1}),
		footer:  factory.CreateStyle(StyleConfig{Foreground: "textSecondary"}),
	}
}

func centerText(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	left := (width - len(text)) / 2
	right := width - len(text) - left
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

func (m StatisticsModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m StatisticsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 4
		footerHeight := 1
		viewportHeight := m.height - headerHeight - footerHeight
		if viewportHeight < 10 {
			viewportHeight = 10
		}

		m.viewport.Width = m.width
		m.viewport.Height = viewportHeight

		return m, nil
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m StatisticsModel) View() string {
	if m.width < 80 || m.height < 20 {
		return "Terminal too small. Resize to at least 80x20.\nPress Ctrl+C to quit."
	}

	s := m.styles

	width := 79
	if m.width > width {
		width = m.width
	}

	header := centerText("🚀 GTI TYPING STATISTICS 🚀", width) + "\n"
	header += strings.Repeat("─", width) + "\n"
	header += m.renderViewSelector() + "\n\n"

	viewportContent := m.viewport.View()

	footer := "\n" + s.footer.Render("[q] Quit   [s] Switch View   [h/l] Navigate   [e] Export   [↑/↓] Scroll   [PgUp/PgDn] Page")

	content := header + viewportContent + footer

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, content)
}

func (m *StatisticsModel) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "s":
		m.switchView()
		return m, nil
	case "h":
		m.previousView()
		return m, nil
	case "l":
		m.nextView()
		return m, nil
	case "e":
		m.exportStatistics()
		return m, nil
	case "up", "k":
		m.viewport.LineUp(1)
		return m, nil
	case "down", "j":
		m.viewport.LineDown(1)
		return m, nil
	case "pgup":
		m.viewport.HalfViewUp()
		return m, nil
	case "pgdown":
		m.viewport.HalfViewDown()
		return m, nil
	}
	return m, nil
}

func (m *StatisticsModel) exportStatistics() {
	stats := m.getFilteredStats()
	records := m.getFilteredRecords()

	exportData := map[string]interface{}{
		"exported_at":   time.Now().Format(time.RFC3339),
		"view":          string(m.view),
		"statistics":    stats,
		"session_count": len(records),
		"sessions":      records,
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("gti_statistics_%s_%s.json", m.view, timestamp)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	exportDir := filepath.Join(homeDir, "Downloads")
	if err := config.EnsureDir(exportDir); err != nil {
		exportDir = homeDir
	}

	filePath := filepath.Join(exportDir, filename)

	if err := config.SaveJSONData(filePath, exportData); err != nil {
		return
	}

}

func (m *StatisticsModel) switchView() {
	switch m.view {
	case ViewAllTime:
		m.view = ViewWeekly
	case ViewWeekly:
		m.view = ViewDaily
	case ViewDaily:
		m.view = ViewSession
	case ViewSession:
		m.view = ViewAllTime
	}

	if m.cachedView != m.view {
		m.cachedView = m.view
		m.cachedFilteredRecords = m.getFilteredRecords()
		m.cachedFilteredStats = calculateStatistics(m.cachedFilteredRecords)
		m.viewport.SetContent(m.renderScrollableContent())
		m.viewport.GotoTop()
	}
}

func (m *StatisticsModel) nextView() { m.switchView() }

func (m *StatisticsModel) previousView() {
	switch m.view {
	case ViewAllTime:
		m.view = ViewSession
	case ViewSession:
		m.view = ViewDaily
	case ViewDaily:
		m.view = ViewWeekly
	case ViewWeekly:
		m.view = ViewAllTime
	}

	if m.cachedView != m.view {
		m.cachedView = m.view
		m.cachedFilteredRecords = m.getFilteredRecords()
		m.cachedFilteredStats = calculateStatistics(m.cachedFilteredRecords)
		m.viewport.SetContent(m.renderScrollableContent())
		m.viewport.GotoTop()
	}
}

func (m StatisticsModel) getFilteredRecords() []*session.SessionRecord {
	now := time.Now()

	switch m.view {
	case ViewSession:
		if len(m.records) > 0 {
			return []*session.SessionRecord{m.records[0]}
		}
		return []*session.SessionRecord{}

	case ViewDaily:
		var daily []*session.SessionRecord
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		for _, r := range m.records {
			if !r.Timestamp.Before(today) {
				daily = append(daily, r)
			}
		}
		return daily

	case ViewWeekly:
		var weekly []*session.SessionRecord
		daysSinceMonday := int(now.Weekday() - time.Monday)
		if daysSinceMonday < 0 {
			daysSinceMonday += 7
		}
		monday := now.AddDate(0, 0, -daysSinceMonday)
		monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())

		for _, r := range m.records {
			if !r.Timestamp.Before(monday) {
				weekly = append(weekly, r)
			}
		}
		return weekly

	case ViewAllTime:
		fallthrough
	default:
		return m.records
	}
}

func (m StatisticsModel) getFilteredStats() *Statistics {
	return calculateStatistics(m.getFilteredRecords())
}

func (m StatisticsModel) renderScrollableContent() string {
	var b strings.Builder

	var filteredStats *Statistics
	var filteredRecords []*session.SessionRecord

	if m.cachedView == m.view && m.cachedFilteredStats != nil {
		filteredStats = m.cachedFilteredStats
		filteredRecords = m.cachedFilteredRecords
	} else {
		filteredStats = m.getFilteredStats()
		filteredRecords = m.getFilteredRecords()
	}

	b.WriteString(m.renderStatisticsSummaryWithStats(filteredStats))

	b.WriteString(m.renderPerformanceAnalysisWithStats(filteredStats))

	b.WriteString(m.renderAchievements())

	b.WriteString(m.renderRecentSessionsWithRecords(filteredRecords))

	if len(filteredStats.ValidSessions) >= 5 {
		b.WriteString(m.renderTrendChartWithStats(filteredStats))
	}

	return b.String()
}

func (m StatisticsModel) renderStatistics() string {
	s := m.styles
	var b strings.Builder

	titleLine := s.title.Render("GTI TYPING STATISTICS")
	b.WriteString(s.box.Render(titleLine))
	b.WriteString("\n\n")

	b.WriteString(s.section.Render("PERFORMANCE OVERVIEW"))
	b.WriteString("\n")
	b.WriteString(s.subtle.Render(strings.Repeat("─", 79)))
	b.WriteString("\n")

	b.WriteString(m.renderViewSelector())
	b.WriteString("\n\n")

	filteredStats := m.getFilteredStats()
	filteredRecords := m.getFilteredRecords()

	b.WriteString(m.renderStatisticsSummaryWithStats(filteredStats))
	b.WriteString(m.renderPerformanceAnalysisWithStats(filteredStats))
	b.WriteString(m.renderAchievements())
	b.WriteString(m.renderRecentSessionsWithRecords(filteredRecords))

	if len(filteredStats.ValidSessions) >= 5 {
		b.WriteString(m.renderTrendChartWithStats(filteredStats))
	}

	b.WriteString("\n")
	b.WriteString(s.footer.Render("[Enter] Exit   [Tab] View   [Left/Right] Navigate   [Ctrl+E] Export   [Ctrl+R] Reset"))
	b.WriteString("\n")

	return b.String()
}

func (m StatisticsModel) renderViewSelector() string {
	s := m.styles

	type item struct {
		view  StatisticsView
		label string
	}
	items := []item{
		{ViewSession, "SESSION"},
		{ViewDaily, "DAILY"},
		{ViewWeekly, "WEEKLY"},
		{ViewAllTime, "ALL-TIME"},
	}

	var parts []string
	for _, it := range items {
		if m.view == it.view {
			parts = append(parts, s.viewOn.Render("> "+it.label))
		} else {
			parts = append(parts, s.viewOff.Render("  "+it.label))
		}
	}
	return strings.Join(parts, " ")
}

func (m StatisticsModel) renderStatisticsSummaryWithStats(stats *Statistics) string {
	s := m.styles
	var b strings.Builder

	b.WriteString(s.section.Render("STATISTICS SUMMARY"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 79))
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("%s %s    %s %s\n",
		s.key.Render("Total sessions:"),
		s.val.Render(fmt.Sprintf("%d", stats.TotalSessions)),
		s.key.Render("Total time:"),
		s.val.Render(formatDuration(stats.TotalTime)),
	))

	if stats.OutlierCount > 0 {
		b.WriteString(fmt.Sprintf("%s %s    %s %s\n",
			s.key.Render("Valid sessions:"),
			s.val.Render(fmt.Sprintf("%d", len(stats.ValidSessions))),
			s.key.Render("Excluded:"),
			s.val.Render(fmt.Sprintf("%d (short/low-text)", stats.OutlierCount)),
		))
	}
	b.WriteString("\n")

	if len(stats.ValidSessions) > 0 {
		b.WriteString(fmt.Sprintf("%s (>=%.0fs and >=%d chars):", s.key.Render("Normalized WPM"), minValidDuration.Seconds(), minValidTextLength))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  ├─ %s %s\n", s.key.Render("Average:"), s.val.Render(fmt.Sprintf("%.1f wpm", stats.NormalizedAvgWPM))))
		b.WriteString(fmt.Sprintf("  ├─ %s %s\n", s.key.Render("Peak:"), s.val.Render(fmt.Sprintf("%.1f wpm", stats.NormalizedPeakWPM))))

		recent := fmt.Sprintf("%.1f wpm", stats.RecentValidAvgWPM)
		if stats.ImprovementRate != 0 {
			if stats.ImprovementRate > 0 {
				recent += fmt.Sprintf(" (+%.1f%%)", stats.ImprovementRate)
			} else {
				recent += fmt.Sprintf(" (%.1f%%)", stats.ImprovementRate)
			}
		}
		b.WriteString(fmt.Sprintf("  └─ %s %s\n", s.key.Render("Recent avg:"), s.val.Render(recent)))

		if stats.RecentValidCountUsed >= minSessionsForVariance && stats.VariancePercent > 0 {
			varStyle := s.good
			if stats.VariancePercent > highVarianceThreshold {
				varStyle = s.bad
			}
			b.WriteString(fmt.Sprintf("     %s %s\n",
				s.key.Render("Consistency (variance):"),
				varStyle.Render(fmt.Sprintf("±%.1f%%", stats.VariancePercent)),
			))
		}
		b.WriteString("\n")
	} else {
		b.WriteString(s.subtle.Render("No valid sessions in this view (need >=15s and >=60 chars)."))
		b.WriteString("\n\n")
	}

	b.WriteString(fmt.Sprintf("%s %s\n",
		s.key.Render("Raw peak WPM:"),
		s.val.Render(fmt.Sprintf("%.1f (includes short sessions)", stats.RawPeakWPM)),
	))
	b.WriteString("\n")

	b.WriteString(s.key.Render("Accuracy:"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  ├─ %s %s\n", s.key.Render("Average:"), s.val.Render(fmt.Sprintf("%.1f%%", stats.RawAvgAccuracy))))
	b.WriteString(fmt.Sprintf("  └─ %s %s\n", s.key.Render("Best:"), s.val.Render(fmt.Sprintf("%.1f%%", stats.RawBestAccuracy))))
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("%s %s\n", s.key.Render("Avg mistakes:"), s.val.Render(fmt.Sprintf("%.1f per session", stats.AvgMistakes))))
	if stats.BackspaceRate > 0 {
		b.WriteString(fmt.Sprintf("%s %s\n", s.key.Render("Backspace rate:"), s.val.Render(fmt.Sprintf("%.1f per session", stats.BackspaceRate))))
	} else {
		b.WriteString(fmt.Sprintf("%s %s\n", s.key.Render("Backspace rate:"), s.subtle.Render("n/a")))
	}
	b.WriteString("\n")

	if stats.CurrentStreak > 0 || stats.LongestStreak > 0 {
		b.WriteString(s.key.Render("Streaks (consecutive days with practice):"))
		b.WriteString("\n")
		currentStreakStr := fmt.Sprintf("%d days", stats.CurrentStreak)
		if stats.CurrentStreak > 0 {
			currentStreakStr = s.good.Render(currentStreakStr)
		} else {
			currentStreakStr = s.subtle.Render(currentStreakStr)
		}
		b.WriteString(fmt.Sprintf("  ├─ %s %s\n", s.key.Render("Current:"), currentStreakStr))
		b.WriteString(fmt.Sprintf("  └─ %s %s\n", s.key.Render("Longest:"), s.val.Render(fmt.Sprintf("%d days", stats.LongestStreak))))
		b.WriteString("\n")
	}

	return b.String()
}

func (m StatisticsModel) renderPerformanceAnalysisWithStats(stats *Statistics) string {
	s := m.styles
	var b strings.Builder

	b.WriteString(s.section.Render("PERFORMANCE ANALYSIS"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 79))
	b.WriteString("\n")

	var insights []string

	if stats.RawAvgAccuracy > 0 && stats.RawAvgAccuracy < lowAccuracyThreshold {
		insights = append(insights, s.bad.Render(fmt.Sprintf("! Accuracy below %.0f%% (slow down and focus on clean hits)", lowAccuracyThreshold)))
	}

	if stats.VariancePercent > highVarianceThreshold {
		insights = append(insights, s.bad.Render(fmt.Sprintf("! High WPM variance (±%.1f%%): stabilize pace and rhythm", stats.VariancePercent)))
	} else if stats.VariancePercent > 0 && stats.VariancePercent < goodVarianceThreshold {
		insights = append(insights, s.good.Render(fmt.Sprintf("+ Strong consistency (±%.1f%%): keep the same warmup routine", stats.VariancePercent)))
	}

	if stats.OutlierCount > 0 && stats.TotalSessions > 0 && stats.OutlierCount > stats.TotalSessions/3 {
		insights = append(insights, s.bad.Render("! Many short/low-text sessions: add longer runs for better signal"))
	}

	if len(stats.ValidSessions) == 0 && stats.TotalSessions > 0 {
		insights = append(insights, s.bad.Render("! No valid sessions here: aim for >=15s and >=60 chars per session"))
	}

	if stats.ImprovementRate > significantImprovementThreshold {
		insights = append(insights, s.good.Render("+ Recent improvement is strong: increase difficulty gradually"))
	} else if stats.ImprovementRate < -significantImprovementThreshold {
		insights = append(insights, s.bad.Render("! Recent decline: reduce speed targets and reset technique"))
	}

	if len(insights) == 0 {
		insights = append(insights, s.good.Render("+ Metrics look healthy: keep practicing consistently"))
	}

	for _, in := range insights {
		b.WriteString(in)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	return b.String()
}

func (m StatisticsModel) renderAchievements() string {
	s := m.styles
	var b strings.Builder

	b.WriteString(s.section.Render("ACHIEVEMENTS (ALL-TIME)"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 79))
	b.WriteString("\n")

	type ach struct {
		condition   bool
		mark        string
		title       string
		description string
	}

	achievements := []ach{
		{m.stats.TotalSessions >= 1, "[*]", "First Steps", "Complete your first session"},
		{m.stats.TotalSessions >= 10, "[+]", "Getting Started", "Complete 10 sessions"},
		{m.stats.TotalSessions >= 50, "[#]", "Dedicated", "Complete 50 sessions"},
		{m.stats.TotalSessions >= 100, "[#]", "Committed", "Complete 100 sessions"},

		{m.stats.NormalizedPeakWPM >= 30, "[>]", "Speed I", "Reach 30 WPM (normalized)"},
		{m.stats.NormalizedPeakWPM >= 50, "[>]", "Speed II", "Reach 50 WPM (normalized)"},
		{m.stats.NormalizedPeakWPM >= 70, "[>]", "Speed III", "Reach 70 WPM (normalized)"},
		{m.stats.NormalizedPeakWPM >= 100, "[>]", "Speed IV", "Reach 100 WPM (normalized)"},

		{m.stats.RawBestAccuracy >= 95, "[!]", "Accuracy I", "Hit 95% best accuracy"},
		{m.stats.RawBestAccuracy >= 98, "[!]", "Accuracy II", "Hit 98% best accuracy"},
		{m.stats.RawBestAccuracy >= 99, "[!]", "Accuracy III", "Hit 99% best accuracy"},

		{m.stats.TotalTime >= time.Hour, "[=]", "Time I", "Accumulate 1 hour total typing"},
		{m.stats.TotalTime >= 24*time.Hour, "[=]", "Time II", "Accumulate 24 hours total typing"},

		{m.stats.CurrentStreak >= 3, "[🔥]", "Streak I", "Maintain a 3-day practice streak"},
		{m.stats.CurrentStreak >= 7, "[🔥]", "Streak II", "Maintain a 7-day practice streak"},
		{m.stats.CurrentStreak >= 14, "[🔥]", "Streak III", "Maintain a 14-day practice streak"},
		{m.stats.LongestStreak >= 30, "[🔥]", "Dedication", "Achieve a 30-day practice streak"},

		{m.stats.VariancePercent > 0 && m.stats.VariancePercent < 10, "[~]", "Consistent", "Maintain <10% WPM variance (recent)"},
	}

	unlocked := 0
	total := len(achievements)

	nextTitle := ""
	for _, a := range achievements {
		if !a.condition {
			nextTitle = a.title
			break
		}
	}

	for _, a := range achievements {
		if a.condition {
			unlocked++
			b.WriteString(fmt.Sprintf("  %s %s - %s\n",
				s.good.Render(a.mark),
				s.val.Render(a.title),
				s.subtle.Render(a.description),
			))
		}
	}

	barWidth := 24
	filled := 0
	if total > 0 {
		filled = int(math.Round((float64(unlocked) / float64(total)) * float64(barWidth)))
		if filled < 0 {
			filled = 0
		}
		if filled > barWidth {
			filled = barWidth
		}
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s %s %s\n",
		s.key.Render("Progress:"),
		s.val.Render(bar),
		s.val.Render(fmt.Sprintf("%d/%d", unlocked, total)),
	))
	if nextTitle != "" {
		b.WriteString(fmt.Sprintf("%s %s\n", s.key.Render("Next:"), s.accent.Render(nextTitle)))
	}
	b.WriteString("\n")

	return b.String()
}

func (m StatisticsModel) renderRecentSessionsWithRecords(records []*session.SessionRecord) string {
	s := m.styles
	var b strings.Builder

	b.WriteString(s.section.Render("RECENT SESSIONS"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 79))
	b.WriteString("\n")

	if len(records) == 0 {
		b.WriteString(s.subtle.Render("No sessions in this view."))
		b.WriteString("\n\n")
		return b.String()
	}

	limit := 8
	if len(records) < limit {
		limit = len(records)
	}

	for i := 0; i < limit; i++ {
		r := records[i]
		dur := time.Duration(r.DurationMs) * time.Millisecond

		isValid := dur >= 15*time.Second && r.TextLength >= 60
		validMark := s.bad.Render("x")
		wpmStr := "—"
		if isValid {
			validMark = s.good.Render("v")
			wpmStr = fmt.Sprintf("%.1f", r.WPM)
		}

		accStr := fmt.Sprintf("%.1f%%", r.Accuracy)
		line := fmt.Sprintf(
			"%2d. [%s] %s | wpm %6s | acc %6s | %6s | mode %-10s",
			i+1,
			validMark,
			r.Timestamp.Format("2006-01-02 15:04"),
			wpmStr,
			accStr,
			formatDuration(dur),
			r.Mode,
		)
		b.WriteString(line)
		b.WriteString("\n")

		if r.Tier != "" {
			b.WriteString(fmt.Sprintf("     %s %s\n", s.subtle.Render("tier:"), s.val.Render(r.Tier)))
		}
		if r.QuoteAuthor != "" {
			b.WriteString(fmt.Sprintf("     %s %s\n", s.subtle.Render("author:"), s.val.Render(r.QuoteAuthor)))
		}
	}

	b.WriteString("\n")
	return b.String()
}

func (m StatisticsModel) renderTrendChartWithStats(stats *Statistics) string {
	s := m.styles
	var b strings.Builder

	b.WriteString(s.section.Render("WPM TREND (NORMALIZED)"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 79))
	b.WriteString("\n")

	wpms := make([]float64, 0, len(stats.ValidSessions))
	for _, r := range stats.ValidSessions {
		wpms = append(wpms, r.WPM)
	}
	if len(wpms) == 0 {
		b.WriteString(s.subtle.Render("No valid sessions to chart."))
		b.WriteString("\n\n")
		return b.String()
	}

	sorted := make([]float64, len(wpms))
	copy(sorted, wpms)
	sort.Float64s(sorted)

	p95 := int(math.Floor(float64(len(sorted)-1) * 0.95))
	if p95 < 0 {
		p95 = 0
	}
	if p95 >= len(sorted) {
		p95 = len(sorted) - 1
	}
	maxScale := sorted[p95]
	if maxScale <= 0 {
		maxScale = 1
	}

	outliers := 0
	for _, w := range wpms {
		if w > maxScale {
			outliers++
		}
	}

	count := 20
	if len(stats.ValidSessions) < count {
		count = len(stats.ValidSessions)
	}

	b.WriteString(fmt.Sprintf("%s %s\n", s.key.Render("Scale:"), s.val.Render(fmt.Sprintf("0 to %.1f WPM (p95)", maxScale))))
	if outliers > 0 {
		b.WriteString(s.subtle.Render(fmt.Sprintf("Note: %d session(s) above p95 marked as outliers.", outliers)))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	const barMax = 40
	for i := 0; i < count; i++ {
		w := stats.ValidSessions[i].WPM

		label := fmt.Sprintf("%2d", count-i)

		if w > maxScale {
			b.WriteString(fmt.Sprintf("%s | %s %.1f wpm\n", label, s.bad.Render("[outlier]"), w))
			continue
		}

		barLen := int(math.Round((w / maxScale) * float64(barMax)))
		if barLen < 1 {
			barLen = 1
		}
		if barLen > barMax {
			barLen = barMax
		}

		bar := strings.Repeat("█", barLen)
		b.WriteString(fmt.Sprintf("%s | %-40s %.1f wpm\n", label, bar, w))
	}

	b.WriteString("\n")
	return b.String()
}

func calculateStatistics(records []*session.SessionRecord) *Statistics {
	stats := &Statistics{}
	totalSessions := len(records)
	if totalSessions == 0 {
		return stats
	}

	calculateBasicStats(records, stats)

	valid := filterValidSessions(records)
	stats.ValidSessions = valid
	stats.OutlierCount = totalSessions - len(valid)

	if len(valid) == 0 {
		return stats
	}

	calculateNormalizedStats(valid, stats)

	calculateRecentPerformance(valid, stats)

	calculateImprovementRate(valid, stats)

	stats.CurrentStreak, stats.LongestStreak = session.CalculateStreaks(valid)

	return stats
}

func calculateBasicStats(records []*session.SessionRecord, stats *Statistics) {
	totalSessions := len(records)
	var totalWPM, totalAccuracy float64
	var totalMistakes int
	var totalDurationMs int64
	var totalBackspaces int
	var totalCorrectedErrors, totalUncorrectedErrors int

	for _, r := range records {
		totalWPM += r.WPM
		totalAccuracy += r.Accuracy
		totalMistakes += r.Mistakes
		totalDurationMs += r.DurationMs
		totalBackspaces += r.BackspaceCount
		totalCorrectedErrors += r.CorrectedErrors
		totalUncorrectedErrors += r.UncorrectedErrors

		if r.WPM > stats.RawPeakWPM {
			stats.RawPeakWPM = r.WPM
		}
		if r.Accuracy > stats.RawBestAccuracy {
			stats.RawBestAccuracy = r.Accuracy
		}
	}

	stats.TotalSessions = totalSessions
	stats.TotalTime = time.Duration(totalDurationMs) * time.Millisecond
	stats.RawAvgWPM = totalWPM / float64(totalSessions)
	stats.RawAvgAccuracy = totalAccuracy / float64(totalSessions)
	stats.AvgMistakes = float64(totalMistakes) / float64(totalSessions)
	stats.BackspaceRate = float64(totalBackspaces) / float64(totalSessions)
	stats.AvgCorrectedErrors = float64(totalCorrectedErrors) / float64(totalSessions)
	stats.AvgUncorrectedErrors = float64(totalUncorrectedErrors) / float64(totalSessions)
}

func filterValidSessions(records []*session.SessionRecord) []*session.SessionRecord {
	valid := make([]*session.SessionRecord, 0, len(records))
	for _, r := range records {
		d := time.Duration(r.DurationMs) * time.Millisecond
		if d >= minValidDuration && r.TextLength >= minValidTextLength {
			valid = append(valid, r)
		}
	}
	return valid
}

func calculateNormalizedStats(valid []*session.SessionRecord, stats *Statistics) {
	var sumValid, sumNetWPM, sumAdjustedWPM float64
	var maxValid, maxNetWPM, maxAdjustedWPM float64

	for _, r := range valid {
		sumValid += r.WPM
		sumNetWPM += r.NetWPM
		sumAdjustedWPM += r.AdjustedWPM

		if r.WPM > maxValid {
			maxValid = r.WPM
		}
		if r.NetWPM > maxNetWPM {
			maxNetWPM = r.NetWPM
		}
		if r.AdjustedWPM > maxAdjustedWPM {
			maxAdjustedWPM = r.AdjustedWPM
		}
	}

	stats.NormalizedAvgWPM = sumValid / float64(len(valid))
	stats.NormalizedPeakWPM = maxValid
	stats.NetAvgWPM = sumNetWPM / float64(len(valid))
	stats.NetPeakWPM = maxNetWPM
	stats.AdjustedAvgWPM = sumAdjustedWPM / float64(len(valid))
	stats.AdjustedPeakWPM = maxAdjustedWPM
}

func calculateRecentPerformance(valid []*session.SessionRecord, stats *Statistics) {
	recentN := recentSessionsCount
	if len(valid) < recentN {
		recentN = len(valid)
	}
	stats.RecentValidCountUsed = recentN

	var recentSum float64
	for i := 0; i < recentN; i++ {
		recentSum += valid[i].WPM
	}
	stats.RecentValidAvgWPM = recentSum / float64(recentN)

	if recentN >= minSessionsForVariance && stats.RecentValidAvgWPM > 0 {
		var variance float64
		for i := 0; i < recentN; i++ {
			diff := valid[i].WPM - stats.RecentValidAvgWPM
			variance += diff * diff
		}
		variance /= float64(recentN)
		stdDev := math.Sqrt(variance)
		stats.ConsistencyScore = (stdDev / stats.RecentValidAvgWPM) * 100
		stats.VariancePercent = stats.ConsistencyScore
	}
}

func calculateImprovementRate(valid []*session.SessionRecord, stats *Statistics) {
	if len(valid) >= minSessionsForImprovement {
		half := len(valid) / 2

		var newerSum, olderSum float64
		for i := 0; i < half; i++ {
			newerSum += valid[i].WPM
			olderSum += valid[len(valid)-1-i].WPM
		}

		newerAvg := newerSum / float64(half)
		olderAvg := olderSum / float64(half)
		if olderAvg > 0 {
			stats.ImprovementRate = ((newerAvg - olderAvg) / olderAvg) * 100
		}
	}
}



func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Round(time.Second).Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%02dm", h, m)
}
