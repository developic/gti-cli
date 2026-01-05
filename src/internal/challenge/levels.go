package challenge

import "math"

type ChallengeLevel struct {
	Name        string  `toml:"name"`
	TimeSeconds int     `toml:"time_seconds"`
	MinAccuracy float64 `toml:"min_accuracy"`
	MaxMistakes int     `toml:"max_mistakes"`
	MinChars    int     `toml:"min_chars"`
	MinWords    int     `toml:"min_words"`
	IsBoss      bool    `toml:"is_boss"`
}

// Difficulty tiers with meaningful names and descriptions
type DifficultyTier struct {
	Name         string
	Description  string
	StartLevel   int
	EndLevel     int
	BaseAccuracy float64
	TimeBase     int
	CharBase     int
	MistakeBase  int
}

var difficultyTiers = []DifficultyTier{
	{
		Name:         "Beginner",
		Description:  "Getting Started",
		StartLevel:   1,
		EndLevel:     10,
		BaseAccuracy: 90.0,
		TimeBase:     30,
		CharBase:     15,
		MistakeBase:  100,
	},
	{
		Name:         "Apprentice",
		Description:  "Building Skills",
		StartLevel:   11,
		EndLevel:     25,
		BaseAccuracy: 92.0,
		TimeBase:     35,
		CharBase:     50,
		MistakeBase:  50,
	},
	{
		Name:         "Intermediate",
		Description:  "Finding Rhythm",
		StartLevel:   26,
		EndLevel:     50,
		BaseAccuracy: 94.0,
		TimeBase:     40,
		CharBase:     100,
		MistakeBase:  25,
	},
	{
		Name:         "Advanced",
		Description:  "Precision Work",
		StartLevel:   51,
		EndLevel:     75,
		BaseAccuracy: 96.0,
		TimeBase:     45,
		CharBase:     200,
		MistakeBase:  15,
	},
	{
		Name:         "Expert",
		Description:  "Master Level",
		StartLevel:   76,
		EndLevel:     95,
		BaseAccuracy: 97.5,
		TimeBase:     50,
		CharBase:     300,
		MistakeBase:  8,
	},
	{
		Name:         "Legendary",
		Description:  "Typing Legend",
		StartLevel:   96,
		EndLevel:     100,
		BaseAccuracy: 98.5,
		TimeBase:     60,
		CharBase:     400,
		MistakeBase:  3,
	},
}

// Boss level names with variety
var bossNames = []string{
	"Speed Demon",
	"Accuracy Ace",
	"Time Master",
	"Word Warrior",
	"Precision Pro",
	"Ultimate Test",
	"Legendary Challenge",
	"Master's Trial",
}

// Generate level name based on tier and position
func generateLevelName(levelNum int, isBoss bool, tier DifficultyTier) string {
	if isBoss {
		bossIndex := (levelNum - 1) % len(bossNames)
		return tier.Name + " Boss - " + bossNames[bossIndex]
	}

	// Varied level names within each tier
	levelNames := []string{
		"Foundation", "Building Blocks", "First Steps", "Getting Comfortable",
		"Settling In", "Finding Flow", "Building Confidence", "Growing Stronger",
		"Pushing Limits", "Tier Complete",
	}

	localLevel := levelNum - tier.StartLevel
	if localLevel < len(levelNames) {
		return tier.Name + " - " + levelNames[localLevel]
	}

	return tier.Name + " - Level " + string(rune('A'+localLevel-len(levelNames)))
}

func GetBuiltInLevels() []ChallengeLevel {
	var levels []ChallengeLevel

	for _, tier := range difficultyTiers {
		levelsInTier := tier.EndLevel - tier.StartLevel + 1
		bossInterval := 5 // Boss every 5 levels

		for i := 0; i < levelsInTier; i++ {
			levelNum := tier.StartLevel + i
			isBoss := (i+1)%bossInterval == 0 || levelNum == 100

			// Calculate progressive difficulty within tier
			progress := float64(i) / float64(levelsInTier-1)

			// Non-linear progression curves for more realistic difficulty
			accuracy := tier.BaseAccuracy + progress*progress*4.0 // Quadratic increase
			timeSeconds := tier.TimeBase + int(progress*progress*15.0) // Quadratic time increase
			charCount := tier.CharBase + int(progress*progress*200.0)   // Quadratic length increase
			maxMistakes := int(float64(tier.MistakeBase) * (1.0 - progress*progress*0.8)) // Quadratic decrease

			// Ensure minimums
			if accuracy < 85.0 {
				accuracy = 85.0
			}
			if timeSeconds < 20 {
				timeSeconds = 20
			}
			if charCount < 10 {
				charCount = 10
			}
			if maxMistakes < 1 {
				maxMistakes = 1
			}

			// Special handling for final level
			if levelNum == 100 {
				accuracy = 99.5
				timeSeconds = 120
				charCount = 2000
				maxMistakes = 5
			}

			// Estimate word count (roughly 5 chars per word)
			minWords := int(math.Ceil(float64(charCount) / 5.5))

			level := ChallengeLevel{
				Name:        generateLevelName(levelNum, isBoss, tier),
				TimeSeconds: timeSeconds,
				MinAccuracy: math.Round(accuracy*10) / 10, // Round to 1 decimal
				MaxMistakes: maxMistakes,
				MinChars:    charCount,
				MinWords:    minWords,
				IsBoss:      isBoss,
			}

			levels = append(levels, level)
		}
	}

	return levels
}
