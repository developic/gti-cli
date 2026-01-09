package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gti/src/internal"
	"gti/src/internal/app"
)

var codeLanguage string
var codeCount int
var codeTimed string
var codeCustom string
var codeStart int

var codeCmd = &cobra.Command{
	Use:   "code [language]",
	Short: "Practice typing with code snippets",
	Long: `Practice typing with actual code snippets from various programming languages.

Supported languages: go, python, javascript, java, cpp, rust, typescript

EXAMPLES:
  gti code                    # Practice Go code (default)
  gti code python             # Practice Python code
  gti code javascript -n 3    # Practice 3 JavaScript snippets
  gti code -t 60              # Timed code practice (60 seconds)
  gti code java               # Practice Java code

OPTIONS:
  -l, --language <lang>       Programming language (go, python, javascript, etc.)
  -n, --count <num>           Number of code snippets (default: 1)
  -c, --custom <file>         Practice with custom code file (.py, .go, .js, etc.)
  --start <num>               Start from paragraph number (for custom files)
  -t, --timed <duration>      Timed mode with duration (e.g., 30, 10s, 5m)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if custom file is specified
		if codeCustom != "" {
			// Handle custom code file - use custom-code mode for proper code rendering
			if codeTimed != "" {
				// Timed custom code file
				timedSeconds := parseDuration(codeTimed)
				return app.StartApp(app.AppOptions{
					Mode:    "custom-code",
					File:    codeCustom,
					Start:   codeStart,
					Seconds: timedSeconds,
				})
			} else {
				// Regular custom code file
				return app.StartApp(app.AppOptions{
					Mode:  "custom-code",
					File:  codeCustom,
					Start: codeStart,
				})
			}
		}

		// parse language from args or flags
		language := "go" // default
		if len(args) > 0 {
			language = args[0]
		}
		if codeLanguage != "" {
			language = codeLanguage
		}

		// Validate language
		if err := internal.ValidateCodeLanguage(language); err != nil {
			supportedLanguages := internal.GetSupportedCodeLanguages()
			return fmt.Errorf("%s. Supported languages: %s", err.Error(), strings.Join(supportedLanguages, ", "))
		}

		// Validate count
		if codeCount < 1 {
			codeCount = 1
		}
		if codeCount > 10 {
			codeCount = 10
		}

		// Handle different nodes
		if codeTimed != "" {
			// Timed 
			timedSeconds := parseDuration(codeTimed)
			return app.StartCodePracticeTimed(language, codeCount, timedSeconds)
		} else {
			return app.StartCodePractice(language, codeCount)
		}
	},
}

func init() {
	codeCmd.Flags().StringVarP(&codeLanguage, "language", "l", "", "programming language")
	codeCmd.Flags().IntVarP(&codeCount, "count", "n", 1, "number of code snippets")
	codeCmd.Flags().StringVarP(&codeCustom, "custom", "c", "", "practice with custom code file (.py, .go, .js, etc.)")
	codeCmd.Flags().IntVar(&codeStart, "start", 1, "start from paragraph number (for custom files)")
	codeCmd.Flags().StringVarP(&codeTimed, "timed", "t", "", "timed mode with duration (e.g., 30, 10s, 5m)")
}
