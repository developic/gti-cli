package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gti/src/internal"
	"gti/src/internal/app"
	"gti/src/internal/config"
)

var cfgFile string
var chunksPerGroup int
var defaultGroups int
var language string
var startParagraph int

var rootCmd = &cobra.Command{
	Use:   "gti",
	Short: "Terminal-based typing speed and practice application",
	Long: `GTI is a fast typing test application for the terminal.

USAGE
  gti [OPTIONS] [COMMAND]

QUICK START
  gti                    Start practice mode (2 chunks)
  gti -n 10              Start practice with 10 chunks per group
  gti -g 3               Start practice with 3 groups (6 chunks)
  gti -t 30              Start 30-second timed test
  gti -c file.txt        Practice with custom text
  gti statistics         View typing statistics

COMMANDS
  quote                  Start with random quotes
  challenge              Progressive challenge with levels
  code                   Practice typing with code snippets
  statistics             View detailed typing statistics
  theme <command>        Manage color themes
  config <command>       View and manage configuration
  version                Display version information

OPTIONS
  -n <count>             Number of chunks per group (default: 2)
  -g <count>             Number of groups (default: 1)
  -c, --custom <file>    Start with custom text file
  --start <num>          Start from paragraph number
  -t, --timed <time>     Start timed mode with duration
  -s, --shortcuts        Show shortcuts and exit
  -h, --help             Display help information
  -v, --version          Display version information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		custom, _ := cmd.Flags().GetString("custom")
		timed, _ := cmd.Flags().GetString("timed")

		if custom != "" {
			seconds := 0
			if timed != "" {
				seconds = parseDuration(timed)
			}
			return startCustomFile(custom, startParagraph, seconds)
		}
		if timed != "" {
			return app.StartTimed(parseDuration(timed))
		}
		if shortcuts, _ := cmd.Flags().GetBool("shortcuts"); shortcuts {
			return showShortcuts()
		}

		totalChunks := defaultGroups * chunksPerGroup
		// Handle language selection and save preference if changed
		if language != "" {
			if err := internal.ValidateLanguage(language); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s. Run 'gti --help' to see available languages.\n", err.Error())
				os.Exit(1)
			}

			cfg := config.GetConfig()
			if cfg.Language.Default != language {
				cfg.Language.Default = language
				if err := config.SaveConfig(); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to save language preference: %v\n", err)
				} else {
					fmt.Printf("Default language set to: %s\n", language)
				}
			}
			return app.StartPracticeWithChunksAndLanguage(totalChunks, language)
		}
		return app.StartPracticeWithChunks(totalChunks)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// init sets up the root command flags and subcommands
	cobra.OnInitialize(initConfig)

	rootCmd.CompletionOptions.DisableDefaultCmd = false

	rootCmd.SetHelpTemplate(`{{.Long}}
`)

	rootCmd.Flags().IntVarP(&chunksPerGroup, "chunks", "n", 2, "number of chunks per group for default practice")
	rootCmd.Flags().IntVarP(&defaultGroups, "groups", "g", 1, "number of groups for default practice")
	rootCmd.Flags().StringP("custom", "c", "", "start with custom text file")
	rootCmd.Flags().IntVar(&startParagraph, "start", 1, "start from paragraph number (for custom mode)")
	rootCmd.Flags().StringP("timed", "t", "", "start timed mode with duration (e.g., 30, 10s, 5m)")
	rootCmd.Flags().StringVarP(&language, "language", "l", "", "language for word generation (english, spanish, french, german, japanese, etc.)")
	rootCmd.Flags().BoolP("shortcuts", "s", false, "show shortcuts and exit")

	rootCmd.AddCommand(quoteCmd)
	rootCmd.AddCommand(challengeCmd)
	rootCmd.AddCommand(codeCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(themeCmd)
	rootCmd.AddCommand(statisticsCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	config.InitConfig(cfgFile)
}

func parseDuration(durationStr string) int {
	// Parse duration string (e.g., "30s", "5m", or plain number) into seconds, defaulting to 60 if invalid
	if duration, err := time.ParseDuration(durationStr); err == nil {
		return int(duration.Seconds())
	}
	if s, err := strconv.Atoi(durationStr); err == nil {
		return s
	}

	return 60
}

// isCodeFile checks if the given file path has a code file extension
func isCodeFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	codeExtensions := map[string]bool{
		".go":   true,
		".py":   true,
		".js":   true,
		".java": true,
		".cpp":  true,
		".rs":   true,
		".ts":   true,
	}
	return codeExtensions[ext]
}

// startCustomFile starts a typing session with a custom file, automatically detecting if it's code
func startCustomFile(file string, start int, seconds int) error {
	if isCodeFile(file) {
		return app.StartApp(app.AppOptions{
			Mode:    "custom-code",
			File:    file,
			Start:   start,
			Seconds: seconds,
		})
	}
	if seconds > 0 {
		return app.StartCustomTimed(file, start, seconds)
	}
	return app.StartCustom(file, start)
}

func showShortcuts() error {
	shortcuts := []struct{ key, desc string }{
		{"GLOBAL SHORTCUTS", ""},
		{"Ctrl+C", "Force quit application"},
		{"Ctrl+Q", "Quit with confirmation"},
		{"Esc", "Close overlays / Cancel"},
		{"", ""},
		{"TYPING SESSION CONTROLS", ""},
		{"Tab/Enter", "Submit text / Accept results"},
		{"Ctrl+R", "Restart current session"},
		{"Backspace", "Delete characters"},
		{"Ctrl+H", "Show help overlay"},
		{"", ""},
		{"NAVIGATION", ""},
		{"Left/Right", "Navigate text segments"},
		{"Up/Down", "Scroll content"},
		{"PgUp/PgDn", "Page scroll"},
		{"", ""},
		{"QUICK START FLAGS", ""},
		{"-c <file>", "Start with custom text"},
		{"-t <time>", "Start timed test"},
		{"-s", "Show this help"},
	}

	fmt.Println("GTI Keyboard Shortcuts")
	fmt.Println("======================")
	fmt.Println()

	for _, s := range shortcuts {
		if s.desc == "" {
			if s.key != "" {
				fmt.Printf("%s:\n", s.key)
			} else {
				fmt.Println()
			}
		} else {
			fmt.Printf("  %-12s %s\n", s.key, s.desc)
		}
	}

	fmt.Println("\nExamples:")
	fmt.Println("  gti -s           # Show shortcuts")
	fmt.Println("  gti -c file.txt  # Custom text")
	fmt.Println("  gti -t 30        # 30-second test")

	return nil
}
