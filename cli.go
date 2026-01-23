package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CLI flags
var (
	interactive    = flag.Bool("interactive", false, "Enable interactive mode")
	i              = flag.Bool("i", false, "Enable interactive mode (short)")
	jsonOutput     = flag.Bool("json", false, "Output in JSON format")
	noMarkdown     = flag.Bool("no-markdown", false, "Disable markdown rendering")
	markdown       = flag.Bool("markdown", false, "Force enable markdown rendering")
	noSpinner      = flag.Bool("no-spinner", false, "Disable loading spinner")
	showSpinner    = flag.Bool("spinner", false, "Force enable loading spinner")
	noStream       = flag.Bool("no-stream", false, "Disable response streaming")
	stream         = flag.Bool("stream", false, "Force enable response streaming")
	configPath     = flag.String("config", "", "Path to config file")
	generateConfig = flag.Bool("generate-config", false, "Generate default config file and exit")
)

// setupUsage configures the CLI usage/help text
func setupUsage() {
	flag.Usage = func() {
		// Get executable name without full path for cleaner output
		execName := filepath.Base(os.Args[0])
		if runtime.GOOS == "windows" && strings.HasSuffix(execName, ".exe") {
			execName = strings.TrimSuffix(execName, ".exe")
		}

		fmt.Fprintf(os.Stderr, "Usage: %s [options] [prompt] [model]\n", execName)
		fmt.Fprintf(os.Stderr, "\nWindows Automation Assistant - AI-powered Windows task automation\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s \"list files in current directory\"\n", execName)
		fmt.Fprintf(os.Stderr, "  %s -interactive\n", execName)
		fmt.Fprintf(os.Stderr, "  %s --json \"analyze this file\" \"gpt-4.1\"\n", execName)
		fmt.Fprintf(os.Stderr, "  %s --markdown \"create a table of processes\"\n", execName)
		fmt.Fprintf(os.Stderr, "  %s --no-markdown \"simple text only\"\n", execName)
		fmt.Fprintf(os.Stderr, "  %s --no-stream \"wait for full response\"\n", execName)
		fmt.Fprintf(os.Stderr, "  %s --generate-config\n", execName)
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  ASSISTANT_DEBUG=1     Show detailed error information with file/line numbers\n")
		fmt.Fprintf(os.Stderr, "  NO_SPINNER=1          Disable progress spinner animations\n")
		fmt.Fprintf(os.Stderr, "\nFor more information, see README.md\n")
	}
}

// parseFlags parses CLI flags and returns the loaded config
func parseFlags() *Config {
	setupUsage()
	flag.Parse()

	// Handle generate config command
	if *generateConfig {
		if err := generateDefaultConfig(); err != nil {
			handleError(err, "Generating config")
		}
		fmt.Println("Default config file generated successfully")
		os.Exit(0)
	}

	// Load configuration
	config, err := LoadConfig(*configPath)
	if err != nil {
		handleError(err, "Loading configuration")
	}

	// Override config with CLI flags
	if *noMarkdown {
		config.Output.Markdown = false
	} else if *markdown {
		config.Output.Markdown = true
	}

	if *noSpinner {
		config.Output.Spinner = false
	} else if *showSpinner {
		config.Output.Spinner = true
	}

	if *noStream {
		config.Output.Streaming = false
	} else if *stream {
		config.Output.Streaming = true
	}

	// JSON output mode
	if *jsonOutput {
		config.Output.JSON = true
		config.Output.Markdown = false  // Disable markdown for clean JSON
		config.Output.Spinner = false   // Disable spinner for clean JSON
		config.Output.Streaming = false // Disable streaming for clean JSON
	}

	// Override config with environment variables
	if debugEnv := os.Getenv("ASSISTANT_DEBUG"); debugEnv == "1" {
		config.Debug = true
	}
	if os.Getenv("NO_SPINNER") == "1" {
		config.Output.Spinner = false
	}

	// Validate configuration
	if err := ValidateConfig(config); err != nil {
		handleError(err, "Configuration validation")
	}

	return config
}

// isInteractiveMode returns true if interactive mode is requested
func isInteractiveMode() bool {
	return *interactive || *i
}

// getPromptAndModel extracts prompt and model from CLI args
func getPromptAndModel(config *Config) (prompt, model string) {
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	prompt = args[0]
	model = config.Model // default from config
	if len(args) >= 2 {
		model = args[1] // override from command line
	}
	return
}

// generateDefaultConfig creates a default configuration file
func generateDefaultConfig() error {
	config := DefaultConfig()
	return SaveConfig(config, "config.yaml")
}
