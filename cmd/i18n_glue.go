package cmd

import (
	"strconv"

	"steam-cli/internal/i18n"

	"github.com/spf13/cobra"
)

// localizeCommands rewrites Cobra Short / flag usage strings using the
// currently-active i18n language. Called from Execute() after --ui-lang has
// been resolved (auto-detect or explicit value).
func localizeCommands() {
	rootCmd.Short = i18n.T("root.short")
	rootCmd.SetHelpCommand(localizedHelpCommand())
	usageTemplate := localizedUsageTemplate()
	for _, command := range allCommands() {
		command.InitDefaultHelpFlag()
		command.InitDefaultVersionFlag()
		command.SetUsageTemplate(usageTemplate)
		setFlagUsage(command, "help", i18n.T("flag.help"))
	}
	setFlagUsage(rootCmd, "cc", i18n.T("flag.cc"))
	setFlagUsage(rootCmd, "lang", i18n.T("flag.lang"))
	setFlagUsage(rootCmd, "ui-lang", i18n.T("flag.ui_lang"))
	setFlagUsage(rootCmd, "timeout", i18n.T("flag.timeout"))
	setFlagUsage(rootCmd, "json", i18n.T("flag.json"))
	setFlagUsage(rootCmd, "quiet", i18n.T("flag.quiet"))
	setFlagUsage(rootCmd, "no-color", i18n.T("flag.no_color"))
	setFlagUsage(rootCmd, "verbose", i18n.T("flag.verbose"))
	setFlagUsage(rootCmd, "rate-ms", i18n.T("flag.rate_ms"))
	setFlagUsage(rootCmd, "version", i18n.T("flag.version"))

	searchCmd.Short = i18n.T("search.short")
	setFlagUsage(searchCmd, "count", i18n.T("search.flag.count"))
	appCmd.Short = i18n.T("app.short")
	setFlagUsage(appCmd, "news", i18n.T("app.flag.news"))
	priceCmd.Short = i18n.T("price.short")
	setFlagUsage(priceCmd, "compare", i18n.T("price.flag.compare"))
	mediaCmd.Short = i18n.T("media.short")
	setFlagUsage(mediaCmd, "probe", i18n.T("media.flag.probe"))
	dlcCmd.Short = i18n.T("dlc.short")
	similarCmd.Short = i18n.T("similar.short")
	setFlagUsage(similarCmd, "count", i18n.T("similar.flag.count"))
	dealsCmd.Short = i18n.T("deals.short")
	setFlagUsage(dealsCmd, "count", i18n.T("deals.flag.count"))
	setFlagUsage(dealsCmd, "filter", i18n.T("deals.flag.filter"))
	reviewsCmd.Short = i18n.T("reviews.short")
	setFlagUsage(reviewsCmd, "count", i18n.T("reviews.flag.count"))
	setFlagUsage(reviewsCmd, "filter", i18n.T("reviews.flag.filter"))
	setFlagUsage(reviewsCmd, "type", i18n.T("reviews.flag.type"))
	setFlagUsage(reviewsCmd, "purchase", i18n.T("reviews.flag.purchase"))
	setFlagUsage(reviewsCmd, "cursor", i18n.T("reviews.flag.cursor"))
	newsCmd.Short = i18n.T("news.short")
	setFlagUsage(newsCmd, "count", i18n.T("news.flag.count"))
	achievementsCmd.Short = i18n.T("achievements.short")
	setFlagUsage(achievementsCmd, "count", i18n.T("achievements.flag.count"))
	eventsCmd.Short = i18n.T("events.short")
	setFlagUsage(eventsCmd, "past-days", i18n.T("events.flag.past_days"))
	setFlagUsage(eventsCmd, "future-days", i18n.T("events.flag.future_days"))
	userCmd.Short = i18n.T("user.short")
	wishlistCmd.Short = i18n.T("wishlist.short")
	setFlagUsage(wishlistCmd, "count", i18n.T("wishlist.flag.count"))
	setFlagUsage(wishlistCmd, "offset", i18n.T("wishlist.flag.offset"))
	setFlagUsage(wishlistCmd, "no-details", i18n.T("wishlist.flag.no_details"))
	localesCmd.Short = i18n.T("locales.short")
	setFlagUsage(localesCmd, "type", i18n.T("locales.flag.type"))
	setFlagUsage(localesCmd, "live", i18n.T("locales.flag.live"))
	setFlagUsage(localesCmd, "probe", i18n.T("locales.flag.probe"))
	setFlagUsage(localesCmd, "appid", i18n.T("locales.flag.appid"))
	doctorCmd.Short = i18n.T("doctor.short")
}

func localizedHelpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: i18n.T("help.short"),
		Run: func(cmd *cobra.Command, args []string) {
			target := rootCmd
			if len(args) > 0 {
				if found, _, err := rootCmd.Find(args); err == nil && found != nil {
					target = found
				}
			}
			_ = target.Help()
		},
	}
}

func localizedUsageTemplate() string {
	usage := strconv.Quote(i18n.T("help.usage"))
	aliases := strconv.Quote(i18n.T("help.aliases"))
	examples := strconv.Quote(i18n.T("help.examples"))
	available := strconv.Quote(i18n.T("help.available_commands"))
	flags := strconv.Quote(i18n.T("help.flags"))
	globalFlags := strconv.Quote(i18n.T("help.global_flags"))
	moreInfo := strconv.Quote(i18n.T("help.more_info"))
	return `{{` + usage + `}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

{{` + aliases + `}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{` + examples + `}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

{{` + available + `}}{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{` + flags + `}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{` + globalFlags + `}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

{{printf ` + moreInfo + ` .CommandPath}}{{end}}
`
}

func setFlagUsage(cmd *cobra.Command, name string, usage string) {
	if flag := cmd.Flags().Lookup(name); flag != nil {
		flag.Usage = usage
		return
	}
	if flag := cmd.PersistentFlags().Lookup(name); flag != nil {
		flag.Usage = usage
	}
}
