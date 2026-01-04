package cmd

import (
	"fmt"
	"os"
	"strconv"
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
  -n <count>             Number of chunks per group for default practice (default: 2)
  -g <count>             Number of groups for default practice (default: 1)
  -c, --custom <file>    Start with custom text file
  --start <num>          Start from paragraph number (for custom mode)
  -t, --timed <time>     Start timed mode with duration
  -s, --shortcuts        Show shortcuts and exit
  -h, --help             Display help information
  -v, --version          Display version information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		custom, _ := cmd.Flags().GetString("custom")
		timed, _ := cmd.Flags().GetString("timed")

		if custom != "" && timed != "" {
			return app.StartCustomTimed(custom, startParagraph, parseDuration(timed))
		}

		if custom != "" {
			return app.StartCustom(custom, startParagraph)
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
			if !internal.IsLanguageSupported(language) {
				fmt.Fprintf(os.Stderr, "Error: Language '%s' is not supported. Use one of: english, spanish, french, german, japanese, russian, italian, portuguese, chinese, arabic, hindi, korean, dutch, swedish, czech, danish, finnish, greek, hebrew, hungarian, norwegian, polish, thai, turkish\n", language)
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

func showShortcuts() error {
	fmt.Println("GTI Keyboard Shortcuts and Controls")
	fmt.Println("===================================")
	fmt.Println()
	fmt.Println("GLOBAL SHORTCUTS:")
	fmt.Println("  Ctrl+C        Force quit application (no confirmation)")
	fmt.Println("  Ctrl+Q        Quit with confirmation dialog")
	fmt.Println("  Esc           Close overlays / Cancel operations")
	fmt.Println()
	fmt.Println("TYPING SESSION CONTROLS:")
	fmt.Println("  Tab/Enter     Submit completed text / Accept results")
	fmt.Println("  Ctrl+R        Restart current session/text")
	fmt.Println("  Backspace     Delete characters (during typing)")
	fmt.Println("  Ctrl+H        Show help overlay (if available)")
	fmt.Println()
	fmt.Println("NAVIGATION:")
	fmt.Println("  Left/Right    Navigate between text segments")
	fmt.Println("  Up/Down       Scroll through content (in menus/views)")
	fmt.Println("  PgUp/PgDn     Page up/down (in statistics view)")
	fmt.Println()
	fmt.Println("STATISTICS VIEW CONTROLS:")
	fmt.Println("  q             Quit statistics view")
	fmt.Println("  s             Switch between time views (session/daily/weekly/all-time)")
	fmt.Println("  h/l           Navigate between views (vim-style)")
	fmt.Println("  e             Export current view data to Downloads folder")
	fmt.Println("  ↑/↓           Scroll through statistics")
	fmt.Println("  PgUp/PgDn     Page scroll in statistics")
	fmt.Println()
	fmt.Println("QUICK START FLAGS:")
	fmt.Println("  -c <file>     Start with custom text file")
	fmt.Println("  -t <time>     Start timed test (e.g., -t 30, -t 2m)")
	fmt.Println("  -s            Show this shortcuts help")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  gti shortcuts    # Show this help")
	fmt.Println("  gti -s           # Same as above (shortcut flag)")
	return nil
}
