package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"gti/src/internal/config"
	"gti/src/internal/session"
	"gti/src/internal/tui"
)

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

	ConsistencyScore float64
	ImprovementRate  float64
	VariancePercent  float64
	OutlierCount     int

	CurrentStreak int
	LongestStreak int
}

type statisticsCmdFlags struct {
	view   string
	export bool
	json   bool
}

var statsFlags statisticsCmdFlags

var statisticsCmd = &cobra.Command{
	Use:   "statistics [flags]",
	Short: "View detailed typing statistics and session history",
	Long: `Display comprehensive typing performance metrics and historical data.

The statistics view provides insights into your typing progress, including:
- Performance metrics (WPM, accuracy, consistency)
- Session history with detailed breakdowns
- Achievement tracking and progress indicators
- Trend analysis and improvement insights

VIEWS:
  session    Current session statistics
  daily      Today's typing sessions
  weekly     This week's typing sessions
  all-time   Complete typing history (default)

EXAMPLES:
  gti statistics                    # View all-time statistics
  gti statistics --view daily      # View today's performance
  gti statistics --export          # Export data to Downloads folder
  gti statistics --json            # Output machine-readable JSON

CONTROLS:
  q         Quit statistics view
  s         Switch between time views
  h/l       Navigate between views (vim-style)
  e         Export current view data
  ↑/↓       Scroll through statistics
  PgUp/PgDn Page scroll`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.GetConfig()

		if statsFlags.view != "" {
			validViews := map[string]bool{
				"session": true, "daily": true, "weekly": true, "all-time": true,
			}
			if !validViews[statsFlags.view] {
				return fmt.Errorf("invalid view '%s'. Valid options: session, daily, weekly, all-time", statsFlags.view)
			}
		}

		if statsFlags.json {
			return exportStatisticsJSON(cfg, statsFlags.view)
		}

		model := tui.NewStatisticsModel(cfg)

		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("failed to run statistics interface: %w", err)
		}

		return nil
	},
}

func exportStatisticsJSON(cfg *config.Config, viewFilter string) error {
	records, err := session.LoadSessionRecords(cfg)
	if err != nil {
		return fmt.Errorf("failed to load session records: %w", err)
	}

	var filteredRecords []*session.SessionRecord
	now := time.Now()

	switch viewFilter {
	case "session":
		if len(records) > 0 {
			filteredRecords = []*session.SessionRecord{records[0]}
		}
	case "daily":
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		for _, r := range records {
			if !r.Timestamp.Before(today) {
				filteredRecords = append(filteredRecords, r)
			}
		}
	case "weekly":
		daysSinceMonday := int(now.Weekday() - time.Monday)
		if daysSinceMonday < 0 {
			daysSinceMonday += 7
		}
		monday := now.AddDate(0, 0, -daysSinceMonday)
		monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
		for _, r := range records {
			if !r.Timestamp.Before(monday) {
				filteredRecords = append(filteredRecords, r)
			}
		}
	default:
		filteredRecords = records
	}

	stats := calculateStatistics(filteredRecords)

	exportData := map[string]interface{}{
		"view":       viewFilter,
		"generated":  now.Format(time.RFC3339),
		"statistics": stats,
		"sessions":   filteredRecords,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(exportData)
}

func init() {
	statisticsCmd.Flags().StringVar(&statsFlags.view, "view", "", "statistics view (session, daily, weekly, all-time)")
	statisticsCmd.Flags().BoolVar(&statsFlags.export, "export", false, "export current view data to Downloads folder")
	statisticsCmd.Flags().BoolVar(&statsFlags.json, "json", false, "output statistics in JSON format")
}

func calculateStatistics(records []*session.SessionRecord) *Statistics {
	stats := &Statistics{}
	totalSessions := len(records)
	if totalSessions == 0 {
		return stats
	}

	var totalWPM, totalAccuracy float64
	var totalMistakes int
	var totalDurationMs int64

	for _, r := range records {
		totalWPM += r.WPM
		totalAccuracy += r.Accuracy
		totalMistakes += r.Mistakes
		totalDurationMs += r.DurationMs

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
	stats.BackspaceRate = 0

	valid := make([]*session.SessionRecord, 0, totalSessions)
	for _, r := range records {
		d := time.Duration(r.DurationMs) * time.Millisecond
		if d >= 15*time.Second && r.TextLength >= 60 {
			valid = append(valid, r)
		}
	}
	stats.ValidSessions = valid
	stats.OutlierCount = totalSessions - len(valid)

	if len(valid) == 0 {
		return stats
	}

	var sumValid float64
	maxValid := 0.0
	for _, r := range valid {
		sumValid += r.WPM
		if r.WPM > maxValid {
			maxValid = r.WPM
		}
	}
	stats.NormalizedAvgWPM = sumValid / float64(len(valid))
	stats.NormalizedPeakWPM = maxValid

	recentN := 5
	if len(valid) < recentN {
		recentN = len(valid)
	}
	stats.RecentValidCountUsed = recentN

	var recentSum float64
	for i := 0; i < recentN; i++ {
		recentSum += valid[i].WPM
	}
	stats.RecentValidAvgWPM = recentSum / float64(recentN)

	if recentN >= 3 && stats.RecentValidAvgWPM > 0 {
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

	if len(valid) >= 10 {
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

	stats.CurrentStreak, stats.LongestStreak = session.CalculateStreaks(valid)

	return stats
}
