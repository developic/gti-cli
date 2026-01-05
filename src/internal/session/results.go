package session

import (
	"time"
)

type Results struct {
	WPM      float64
	CPM      float64
	Accuracy float64
	Mistakes int
	Duration time.Duration
}

func CalculateWPM(totalChars int, duration time.Duration) float64 {
	if duration <= 0 {
		return 0
	}
	minutes := duration.Minutes()
	words := float64(totalChars) / 5.0
	return words / minutes
}

func CalculateNetWPM(totalChars int, uncorrectedErrors int, duration time.Duration) float64 {
	if duration <= 0 {
		return 0
	}
	minutes := duration.Minutes()

	penalizedChars := float64(totalChars - (uncorrectedErrors * 5))
	if penalizedChars < 0 {
		penalizedChars = 0
	}
	words := penalizedChars / 5.0
	return words / minutes
}

func CalculateAdjustedWPM(correctChars int, avgWordLength float64, duration time.Duration) float64 {
	if duration <= 0 || avgWordLength <= 0 {
		return 0
	}
	minutes := duration.Minutes()
	words := float64(correctChars) / avgWordLength
	return words / minutes
}

func CalculateAccuracy(totalChars int, mistakes int) float64 {
	if totalChars == 0 {
		return 100.0
	}
	return float64(totalChars-mistakes) / float64(totalChars) * 100
}

type ResultsCalculator struct{}

func NewResultsCalculator() *ResultsCalculator {
	return &ResultsCalculator{}
}

func (rc *ResultsCalculator) CalculateResults(session *Session, mode string) Results {
	var mistakes int

	switch mode {
	case "challenge":

		mistakes = session.GetTotalMistakes()
		if session.GetMistakes() > 0 {
			mistakes += session.GetMistakes()
		}
	case "practice":

		mistakes = session.GetTotalMistakes()
	default:

		mistakes = session.GetTotalMistakes() + session.GetMistakes()
	}

	totalChars := session.GetTotalChars() + len(session.TypedText())

	wpm := CalculateWPM(totalChars, session.GetDuration())

	cpm := 0.0
	if session.GetDuration() > 0 {
		minutes := session.GetDuration().Minutes()
		cpm = float64(totalChars) / minutes
	}

	accuracy := CalculateAccuracy(totalChars, mistakes)

	return Results{
		WPM:      wpm,
		CPM:      cpm,
		Accuracy: accuracy,
		Mistakes: mistakes,
		Duration: session.GetDuration(),
	}
}
