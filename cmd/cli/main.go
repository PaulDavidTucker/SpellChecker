package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/PaulDavidTucker/SpellChecker/internal/allowlist"
	"github.com/PaulDavidTucker/SpellChecker/internal/checker"
	"github.com/PaulDavidTucker/SpellChecker/internal/dictionary"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	dictPath          string
	baseAllowlistPath string
	profilesDir       string
	profileID         string
	outputFormat      string
	ignoreTerms       []string
	maxSuggestions    int

	// Version info
	version = "1.0.0"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "spellcheck-cli",
		Short: "A CLI spell checker with profile-based allowlists",
		Long: `SpellCheck CLI provides spell checking capabilities from the command line.
		
It supports checking text from files, stdin, or command line arguments,
with multiple output formats and profile-based allowlists.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate that dictionary file exists
			if _, err := os.Stat(dictPath); os.IsNotExist(err) {
				return fmt.Errorf("dictionary file not found: %s", dictPath)
			}
			return nil
		},
	}

	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(&dictPath, "dict", "dictionaries/en_gb.txt", "Path to dictionary file")
	rootCmd.PersistentFlags().StringVar(&baseAllowlistPath, "base-allowlist", "config/base-allowlist.yaml", "Path to base allowlist config")
	rootCmd.PersistentFlags().StringVar(&profilesDir, "profiles-dir", "config/profiles", "Directory containing profile configs")

	// Add commands
	rootCmd.AddCommand(checkCmd())
	rootCmd.AddCommand(profilesCmd())
	rootCmd.AddCommand(validateCmd())
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(configCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// checkCmd provides the main spell checking functionality
func checkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [text]",
		Short: "Check text for misspellings",
		Long: `Check text for misspellings and repeated words.
		
The text can be provided as an argument, from a file with --file,
or from stdin (if no argument provided and --file not set).

Examples:
  spellcheck-cli check "The goverment announced a new partnership"
  spellcheck-cli check --file article.txt
  echo "The goverment announced" | spellcheck-cli check
  spellcheck-cli check --profile bbc-news "The MPs debated Brexit"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get text from various sources
			text, err := getText(cmd, args)
			if err != nil {
				return err
			}

			if text == "" {
				return fmt.Errorf("no text provided. Use arguments, --file, or pipe text via stdin")
			}

			// Initialize components
			store, err := allowlist.NewStore(baseAllowlistPath, profilesDir)
			if err != nil {
				return fmt.Errorf("failed to load allowlist store: %w", err)
			}

			pool, err := checker.NewPool(dictPath, store)
			if err != nil {
				return fmt.Errorf("failed to build checker pool: %w", err)
			}

			c := pool.Get(profileID)
			result := c.Check(checker.CheckRequest{
				Text:           text,
				ProfileID:      profileID,
				IgnoreTerms:    ignoreTerms,
				MaxSuggestions: maxSuggestions,
			})

			// Output results
			switch outputFormat {
			case "json":
				return outputJSON(result)
			case "simple":
				return outputSimple(result)
			default:
				return outputPretty(result)
			}
		},
	}

	// Command-specific flags
	cmd.Flags().StringVarP(&profileID, "profile", "p", "default", "Profile to use for checking")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "pretty", "Output format: pretty, simple, json")
	cmd.Flags().StringSliceVarP(&ignoreTerms, "ignore", "i", nil, "Terms to ignore (comma-separated)")
	cmd.Flags().IntVarP(&maxSuggestions, "max-suggestions", "n", 3, "Maximum suggestions per misspelling")
	cmd.Flags().StringP("file", "f", "", "Read text from file")

	return cmd
}

// getText retrieves text from arguments, file, or stdin
func getText(cmd *cobra.Command, args []string) (string, error) {
	// Check if --file flag is set
	filePath, _ := cmd.Flags().GetString("file")
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		return string(data), nil
	}

	// Check if text provided as argument
	if len(args) > 0 {
		return args[0], nil
	}

	// Try to read from stdin
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat stdin: %w", err)
	}

	// Check if stdin has data (not a terminal)
	if stat.Mode()&os.ModeCharDevice == 0 {
		data, err := os.ReadFile("/dev/stdin")
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return string(data), nil
	}

	return "", nil
}

// outputJSON outputs results in JSON format
func outputJSON(result checker.CheckResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// outputSimple outputs results in a simple format
func outputSimple(result checker.CheckResult) error {
	if len(result.Misspellings) == 0 && len(result.RepeatedWords) == 0 {
		fmt.Println("✓ No issues found")
		fmt.Printf("Checked %d tokens in %.2fms\n", result.CheckedCount, result.ElapsedMs)
		return nil
	}

	if len(result.Misspellings) > 0 {
		fmt.Printf("✗ Found %d misspelling(s):\n", len(result.Misspellings))
		for _, ms := range result.Misspellings {
			fmt.Printf("  Line %d, Col %d: '%s'\n", ms.Line, ms.Column, ms.Word)
			if len(ms.Suggestions) > 0 {
				fmt.Printf("    Suggestions: ")
				for i, s := range ms.Suggestions {
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Printf("%s", s.Word)
				}
				fmt.Println()
			}
		}
	}

	if len(result.RepeatedWords) > 0 {
		fmt.Printf("⚠ Found %d repeated word(s):\n", len(result.RepeatedWords))
		for _, rw := range result.RepeatedWords {
			fmt.Printf("  Line %d, Col %d: '%s'\n", rw.Line, rw.Column, rw.Word)
		}
	}

	fmt.Printf("\nChecked %d tokens in %.2fms\n", result.CheckedCount, result.ElapsedMs)
	return nil
}

// outputPretty outputs results in a formatted, human-readable way
func outputPretty(result checker.CheckResult) error {
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║           Spell Check Results                          ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	if len(result.Misspellings) == 0 && len(result.RepeatedWords) == 0 {
		fmt.Println("✓ No issues found!")
		fmt.Println()
		fmt.Printf("Tokens checked: %d\n", result.CheckedCount)
		fmt.Printf("Time elapsed: %.2fms\n", result.ElapsedMs)
		return nil
	}

	if len(result.Misspellings) > 0 {
		fmt.Printf("✗ Misspellings Found: %d\n", len(result.Misspellings))
		fmt.Println(strings.Repeat("─", 50))
		for _, ms := range result.Misspellings {
			fmt.Printf("\nWord:    %s\n", ms.Word)
			fmt.Printf("Location: Line %d, Column %d\n", ms.Line, ms.Column)
			if len(ms.Suggestions) > 0 {
				fmt.Println("Suggestions:")
				for i, s := range ms.Suggestions {
					fmt.Printf("  %d. %s (distance: %d)\n", i+1, s.Word, s.EditDistance)
				}
			}
			fmt.Println()
		}
	}

	if len(result.RepeatedWords) > 0 {
		if len(result.Misspellings) > 0 {
			fmt.Println()
		}
		fmt.Printf("⚠ Repeated Words Found: %d\n", len(result.RepeatedWords))
		fmt.Println(strings.Repeat("─", 50))
		for _, rw := range result.RepeatedWords {
			fmt.Printf("\nWord:    %s\n", rw.Word)
			fmt.Printf("Location: Line %d, Column %d\n", rw.Line, rw.Column)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("═", 50))
	fmt.Printf("Total tokens: %d | Checked: %d | Time: %.2fms\n",
		result.TokenCount, result.CheckedCount, result.ElapsedMs)

	return nil
}

// profilesCmd lists available profiles
func profilesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profiles",
		Short: "List available profiles",
		Long:  `List all available profiles that can be used with the --profile flag.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := allowlist.NewStore(baseAllowlistPath, profilesDir)
			if err != nil {
				return fmt.Errorf("failed to load allowlist store: %w", err)
			}

			count := store.ProfileCount()
			fmt.Printf("Available profiles (%d total):\n\n", count)

			// List default profile
			fmt.Println("• default")
			fmt.Println("  Default profile with base allowlist terms")
			fmt.Println()

			// List other profiles
			entries, err := os.ReadDir(profilesDir)
			if err != nil {
				return fmt.Errorf("failed to read profiles directory: %w", err)
			}

			for _, entry := range entries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
					continue
				}

				profileID := entry.Name()[:len(entry.Name())-5] // Remove .yaml
				if profileID == "default" {
					continue
				}

				// Get profile terms count
				terms := store.ProfileTerms(profileID)
				fmt.Printf("• %s\n", profileID)
				if len(terms) > 0 {
					fmt.Printf("  Terms: %d\n", len(terms))
				}
				fmt.Println()
			}

			return nil
		},
	}
}

// validateCmd validates configuration files
func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration files",
		Long: `Validate that the dictionary, base allowlist, and profile configurations are valid.
		
Checks:
  - Dictionary file exists and is readable
  - Base allowlist YAML is valid
  - All profile YAML files are valid
  - Profile IDs match filenames`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Validating configuration...")
			fmt.Println()

			// Validate dictionary
			fmt.Printf("✓ Dictionary: %s\n", dictPath)
			dict, err := dictionary.Load(dictPath)
			if err != nil {
				return fmt.Errorf("✗ Failed to load dictionary: %w", err)
			}
			fmt.Printf("  Loaded %d words\n", dict.WordCount())
			fmt.Println()

			// Validate base allowlist
			fmt.Printf("✓ Base allowlist: %s\n", baseAllowlistPath)
			if _, err := os.Stat(baseAllowlistPath); os.IsNotExist(err) {
				return fmt.Errorf("✗ Base allowlist file not found: %s", baseAllowlistPath)
			}
			fmt.Println()

			// Validate profiles
			fmt.Printf("✓ Profiles directory: %s\n", profilesDir)
			store, err := allowlist.NewStore(baseAllowlistPath, profilesDir)
			if err != nil {
				return fmt.Errorf("✗ Failed to load profiles: %w", err)
			}
			fmt.Printf("  Found %d profiles\n", store.ProfileCount())
			fmt.Println()

			fmt.Println("✓ All configurations are valid!")
			return nil
		},
	}
}

// versionCmd shows version information
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the current version of the spellcheck-cli tool.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("spellcheck-cli version %s\n", version)
			fmt.Printf("  commit: %s\n", commit)
			fmt.Printf("  built:  %s\n", date)
		},
	}
}

// configCmd shows current configuration
func configCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show current configuration",
		Long:  `Display the current configuration settings that will be used.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Current Configuration:")
			fmt.Println()
			fmt.Printf("Dictionary path:     %s\n", dictPath)
			fmt.Printf("Base allowlist:      %s\n", baseAllowlistPath)
			fmt.Printf("Profiles directory:  %s\n", profilesDir)
			fmt.Printf("Default profile:     default\n")
		},
	}
}
