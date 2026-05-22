package cmd

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"steam-cli/internal/i18n"
	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check Steam CLI network access, locale settings, and core data sources",
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		return runDoctor(), nil
	}, func(value any) error {
		data := value.(*doctorResult)
		if opts.quiet {
			if data.OK {
				fmt.Println(i18n.T("doctor.status.ok"))
			} else {
				fmt.Println(i18n.T("doctor.status.failed"))
			}
			return nil
		}
		fmt.Println(ui.Title.Render(i18n.T("doctor.title")))
		fmt.Println()
		rows := make([][]string, 0, len(data.Checks))
		for _, check := range data.Checks {
			status := i18n.T("doctor.status.ok")
			if !check.OK {
				status = i18n.T("doctor.status.failed")
			}
			rows = append(rows, []string{check.Name, status, fmt.Sprintf("%d", check.Status), check.Message})
		}
		fmt.Println(ui.Table([]string{i18n.T("doctor.header.check"), i18n.T("doctor.header.status"), i18n.T("doctor.header.http"), i18n.T("doctor.header.message")}, rows))
		fmt.Println()
		fmt.Println(ui.KeyValue(i18n.T("label.cc"), data.CC))
		fmt.Println(ui.KeyValue(i18n.T("label.lang"), data.Lang))
		fmt.Println(ui.KeyValue(i18n.T("label.ui_lang"), data.UILang))
		fmt.Println(ui.KeyValue(i18n.T("label.observed"), data.ObservedAt))
		return nil
	}),
}

type doctorResult struct {
	OK         bool          `json:"ok"`
	CC         string        `json:"cc"`
	Lang       string        `json:"lang"`
	UILang     string        `json:"ui_lang"`
	ObservedAt string        `json:"observed_at"`
	Checks     []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	OK      bool   `json:"ok"`
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

func runDoctor() *doctorResult {
	cc := strings.ToUpper(opts.cc)
	result := &doctorResult{
		OK:         true,
		CC:         cc,
		Lang:       opts.lang,
		UILang:     opts.uiLang,
		ObservedAt: time.Now().UTC().Format(time.RFC3339),
	}
	checks := doctorChecks(cc, opts.lang)

	// Reuse the same retry/UA-equipped Client so doctor sees the same network
	// behavior as real commands. We disable cache to avoid masking failures.
	c := steam.NewClient(cc, opts.lang, time.Duration(opts.timeout)*time.Second)
	c.Cache = steam.NewCache()
	attachRetryLogger(c)

	for _, raw := range checks {
		check := doctorCheck{Name: raw.Name, URL: raw.URL}
		// Strip query so c.GetText doesn't double-encode.
		base, params := splitURL(raw.URL)
		_, err := c.GetText(base, params)
		if err != nil {
			check.OK = false
			result.OK = false
			check.Message = classifyError(err).Hint
			if check.Message == "" {
				check.Message = err.Error()
			}
			if he, ok := steam.HTTPErrorFromAny(err); ok {
				check.Status = he.Status
			}
			result.Checks = append(result.Checks, check)
			continue
		}
		check.Status = 200
		check.OK = true
		check.Message = i18n.T("doctor.message.reachable")
		result.Checks = append(result.Checks, check)
	}
	return result
}

func doctorChecks(cc, lang string) []struct {
	Name string
	URL  string
} {
	return []struct {
		Name string
		URL  string
	}{
		{Name: "Steam Store", URL: "https://store.steampowered.com/?l=" + lang},
		{Name: "appdetails", URL: fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=264710&cc=%s&l=%s", cc, lang)},
		{Name: "Steam Web API", URL: "https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/?appid=264710"},
		{Name: "Steam Community", URL: "https://steamcommunity.com/profiles/76561198115468824/?xml=1"},
		{Name: "Steamworks Events", URL: "https://partner.steamgames.com/doc/marketing/upcoming_events?l=" + lang},
	}
}

func splitURL(raw string) (string, url.Values) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw, nil
	}
	values, _ := url.ParseQuery(parsed.RawQuery)
	parsed.RawQuery = ""
	return parsed.String(), values
}
