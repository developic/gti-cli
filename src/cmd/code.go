package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gti/src/internal/app"
)

var codeLanguage string
var codeCount int
var codeTimed int

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
  -t, --timed <seconds>       Timed mode with duration in seconds`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse language from args or flags
		language := "go" // default
		if len(args) > 0 {
			language = args[0]
		}
		if codeLanguage != "" {
			language = codeLanguage
		}

		// Validate language
		supportedLanguages := []string{"go", "python", "javascript", "java", "cpp", "rust", "typescript"}
		isSupported := false
		for _, lang := range supportedLanguages {
			if language == lang {
				isSupported = true
				break
			}
		}
		if !isSupported {
			return fmt.Errorf("unsupported language '%s'. Supported languages: %s",
				language, strings.Join(supportedLanguages, ", "))
		}

		// Validate count
		if codeCount < 1 {
			codeCount = 1
		}
		if codeCount > 10 {
			codeCount = 10
		}

		// Handle different modes
		if codeTimed > 0 {
			// Timed code practice
			return app.StartCodePracticeTimed(language, codeCount, codeTimed)
		} else {
			// Regular code practice
			return app.StartCodePractice(language, codeCount)
		}
	},
}

func init() {
	codeCmd.Flags().StringVarP(&codeLanguage, "language", "l", "", "programming language")
	codeCmd.Flags().IntVarP(&codeCount, "count", "n", 1, "number of code snippets")
	codeCmd.Flags().IntVarP(&codeTimed, "timed", "t", 0, "timed mode with duration in seconds")
}
