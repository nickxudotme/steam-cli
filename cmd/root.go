package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"steam-cli/internal/i18n"
	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

const version = steam.Version

var opts struct {
	cc      string
	lang    string
	timeout int
	json    bool
	quiet   bool
	noColor bool
	verbose bool
	uiLang  string
	rateMs  int
}

var currentCommand string

func Execute() {
	opts.uiLang = initialUILang(os.Args[1:])
	i18n.Set(opts.uiLang)
	localizeCommands()
	normalizeOptions()
	if opts.noColor {
		ui.DisableColor()
	}
	if err := rootCmd.Execute(); err != nil {
		if opts.json {
			classified := classifyError(err)
			if jsonErr := printJSON(jsonEnvelope{
				OK:          false,
				Command:     errorCommandPath(),
				Schema:      commandSchema(errorCommandPath()),
				GeneratedAt: time.Now().UTC().Format(time.RFC3339),
				Meta:        responseMeta(),
				Error:       &classified,
			}); jsonErr != nil {
				fmt.Fprintln(os.Stderr, "steam-cli:", jsonErr)
			}
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "steam-cli:", err)
		os.Exit(1)
	}
}

func initialUILang(args []string) string {
	for i, arg := range args {
		if arg == "--ui-lang" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "--ui-lang=") {
			return strings.TrimPrefix(arg, "--ui-lang=")
		}
	}
	return "auto"
}

func allCommands() []*cobra.Command {
	return []*cobra.Command{
		rootCmd,
		searchCmd,
		appCmd,
		priceCmd,
		mediaCmd,
		dlcCmd,
		similarCmd,
		dealsCmd,
		reviewsCmd,
		newsCmd,
		achievementsCmd,
		eventsCmd,
		userCmd,
		wishlistCmd,
		localesCmd,
		doctorCmd,
	}
}

func normalizeOptions() {
	// Resolve auto-detect for --cc / --lang from system locale on first
	// call. Subsequent invocations from runCommand() are no-ops because
	// opts.cc/lang are no longer "auto" / "" by then.
	if opts.cc == "" || strings.EqualFold(strings.TrimSpace(opts.cc), "auto") {
		opts.cc = autoDetectCC()
	}
	if opts.lang == "" || strings.EqualFold(strings.TrimSpace(opts.lang), "auto") {
		opts.lang = autoDetectLang()
	}

	switch strings.ToUpper(strings.TrimSpace(opts.cc)) {
	case "UK":
		opts.cc = "GB"
	}
	switch strings.ToLower(strings.TrimSpace(opts.lang)) {
	case "chinese", "simplified-chinese", "simplified_chinese", "zh-cn", "zh_cn":
		opts.lang = "schinese"
	case "traditional-chinese", "traditional_chinese", "zh-tw", "zh_tw":
		opts.lang = "tchinese"
	case "korean":
		opts.lang = "koreana"
	case "portuguese-br", "pt-br", "pt_br":
		opts.lang = "brazilian"
	}
	lang := i18n.Set(opts.uiLang)
	opts.uiLang = string(lang)
}

// autoDetectCC resolves the --cc flag from the system locale. Falls back to
// "US" when the locale carries no usable region.
func autoDetectCC() string {
	if cc := i18n.SteamCCFor(i18n.DetectSystemLocale()); cc != "" {
		return cc
	}
	return "US"
}

// autoDetectLang resolves the --lang flag from the system locale. Falls back
// to "english" when no Steam-supported language matches.
func autoDetectLang() string {
	if lang := i18n.SteamLangFor(i18n.DetectSystemLocale()); lang != "" {
		return lang
	}
	return "english"
}

var rootCmd = &cobra.Command{
	Use:                        "steam-cli",
	Short:                      i18n.T("root.short"),
	Version:                    version,
	SilenceUsage:               true,
	SilenceErrors:              true,
	SuggestionsMinimumDistance: 2,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
}

func init() {
	cobra.EnableCommandSorting = false

	rootCmd.PersistentFlags().StringVar(&opts.cc, "cc", "auto", "country/region code for prices; \"auto\" detects from system locale (default), run steam-cli locales --type regions for explicit values")
	rootCmd.PersistentFlags().StringVar(&opts.lang, "lang", "auto", "Steam content language; \"auto\" detects from system locale (default), run steam-cli locales --type languages for values")
	rootCmd.PersistentFlags().IntVar(&opts.timeout, "timeout", 12, "request timeout in seconds")
	rootCmd.PersistentFlags().BoolVar(&opts.json, "json", false, "print JSON envelope")
	rootCmd.PersistentFlags().BoolVar(&opts.quiet, "quiet", false, "print only the most important fields for supported commands")
	rootCmd.PersistentFlags().BoolVar(&opts.noColor, "no-color", false, "disable ANSI color in terminal output")
	rootCmd.PersistentFlags().BoolVar(&opts.verbose, "verbose", false, "print retry and diagnostic messages to stderr")
	rootCmd.PersistentFlags().StringVar(&opts.uiLang, "ui-lang", "auto", "Steam CLI interface language: auto, en, zh-CN")
	rootCmd.PersistentFlags().IntVar(&opts.rateMs, "rate-ms", 0, "minimum milliseconds between requests from a single client; default lets the client be polite without throttling")

	rootCmd.AddCommand(searchCmd, appCmd, priceCmd, mediaCmd, dlcCmd, similarCmd, dealsCmd, reviewsCmd, newsCmd, achievementsCmd, eventsCmd, userCmd, wishlistCmd, localesCmd, doctorCmd)
}

// sharedCache is a process-wide steam.Cache reused by every Client returned
// from client(). Multiple Clients with different --cc values will all hit
// the same map, so cross-region price comparison can re-use cached entries.
var sharedCache = steam.NewCache()

func client() *steam.Client {
	c := steam.NewClient(strings.ToUpper(opts.cc), opts.lang, time.Duration(opts.timeout)*time.Second)
	c.Cache = sharedCache
	if opts.rateMs > 0 {
		c.MinInterval = time.Duration(opts.rateMs) * time.Millisecond
	}
	attachRetryLogger(c)
	return c
}

const longRetryNotice = 2 * time.Second

func attachRetryLogger(c *steam.Client) {
	if opts.json {
		return
	}
	c.RetryLogger = func(event steam.RetryEvent) {
		if !opts.verbose && event.Delay < longRetryNotice {
			return
		}
		fmt.Fprintln(os.Stderr, retryNotice(event))
	}
}

func retryNotice(event steam.RetryEvent) string {
	target := endpointHost(event.URL)
	prefix := "steam-cli:"
	if event.Status == 429 {
		return fmt.Sprintf("%s rate limited by %s, retrying in %s (attempt %d/%d)", prefix, target, shortDuration(event.Delay), event.Attempt+1, event.MaxAttempts)
	}
	if event.Status > 0 {
		return fmt.Sprintf("%s %s returned HTTP %d, retrying in %s (attempt %d/%d)", prefix, target, event.Status, shortDuration(event.Delay), event.Attempt+1, event.MaxAttempts)
	}
	if event.Err != nil {
		return fmt.Sprintf("%s request to %s failed, retrying in %s (attempt %d/%d): %v", prefix, target, shortDuration(event.Delay), event.Attempt+1, event.MaxAttempts, event.Err)
	}
	return fmt.Sprintf("%s retrying %s in %s (attempt %d/%d)", prefix, target, shortDuration(event.Delay), event.Attempt+1, event.MaxAttempts)
}

func endpointHost(raw string) string {
	if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		return parsed.Host
	}
	return raw
}

func shortDuration(d time.Duration) string {
	if d >= time.Second {
		return d.Round(100 * time.Millisecond).String()
	}
	return d.Round(time.Millisecond).String()
}

type commandLoader func(cmd *cobra.Command, args []string) (any, error)
type commandRenderer func(value any) error

func runCommand(load commandLoader, render commandRenderer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		normalizeOptions()
		if opts.noColor {
			ui.DisableColor()
		}
		currentCommand = cmd.CommandPath()
		value, err := load(cmd, args)
		if err != nil {
			return err
		}
		if opts.json {
			return printJSON(jsonEnvelope{
				OK:          true,
				Command:     cmd.CommandPath(),
				Schema:      commandSchema(cmd.CommandPath()),
				GeneratedAt: time.Now().UTC().Format(time.RFC3339),
				Meta:        responseMetaFor(cmd.CommandPath()),
				Data:        value,
			})
		}
		return render(value)
	}
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
