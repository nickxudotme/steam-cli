package cmd

import (
	"fmt"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var localesOpts struct {
	kind  string
	live  bool
	probe bool
	appid int
}

type localesPayload struct {
	Regions     []steam.RegionOption     `json:"regions,omitempty"`
	Languages   []steam.LanguageOption   `json:"languages,omitempty"`
	RegionProbe *steam.RegionProbeResult `json:"region_probe,omitempty"`
	Sources     map[string]string        `json:"sources,omitempty"`
}

var localesCmd = &cobra.Command{
	Use:   "locales",
	Short: "List common --cc regions and --lang Steam languages",
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		if err := validateEnumFlag("type", localesOpts.kind, "all", "regions", "languages"); err != nil {
			return nil, err
		}
		data := steam.LocaleOptionsData()
		payload := localesPayload{
			Sources: map[string]string{
				"regions":   "built-in common Steam price regions",
				"languages": "built-in Steam language codes",
			},
		}
		if localesOpts.live {
			languages, err := client().LiveLanguages()
			if err != nil {
				return nil, err
			}
			data.Languages = languages
			payload.Sources["languages"] = "https://store.steampowered.com/ language menu"
		}
		if localesOpts.probe {
			probe, err := client().ProbeRegions(localesOpts.appid, data.Regions)
			if err != nil {
				return nil, err
			}
			payload.RegionProbe = probe
			payload.Sources["regions"] = probe.Source + " observed by appid"
		}
		switch localesOpts.kind {
		case "all", "":
			payload.Regions = data.Regions
			payload.Languages = data.Languages
			return payload, nil
		case "regions":
			payload.Regions = data.Regions
			return payload, nil
		case "languages":
			payload.Languages = data.Languages
			return payload, nil
		}
		return payload, nil
	}, func(value any) error {
		data, ok := value.(localesPayload)
		if !ok {
			return fmt.Errorf("unexpected locales payload %T", value)
		}
		fmt.Println(ui.Title.Render("Steam CLI locales"))
		if len(data.Regions) > 0 {
			fmt.Println()
			fmt.Println(ui.Section("Regions for --cc"))
			rows := make([][]string, 0, len(data.Regions))
			for _, region := range data.Regions {
				rows = append(rows, []string{region.Code, region.Name})
			}
			fmt.Println(ui.Table([]string{"Code", "Region"}, rows))
			fmt.Println(ui.Muted.Render("Steam generally accepts ISO 3166-1 alpha-2 country codes when that store region is supported."))
		}
		if data.RegionProbe != nil {
			fmt.Println()
			fmt.Println(ui.Section(fmt.Sprintf("Observed region price support for appid %d", data.RegionProbe.AppID)))
			rows := make([][]string, 0, len(data.RegionProbe.Regions))
			for _, region := range data.RegionProbe.Regions {
				status := "no"
				if region.Available {
					status = "yes"
				}
				price := region.FinalFormatted
				if price == "" && region.Currency != "" {
					price = region.Currency
				}
				if price == "" && region.Error != "" {
					price = region.Error
				}
				rows = append(rows, []string{region.Code, region.Name, status, region.Currency, price})
			}
			fmt.Println(ui.Table([]string{"Code", "Region", "Available", "Currency", "Price / Error"}, rows))
			fmt.Println(ui.Muted.Render("This is an observed probe against Steam appdetails, not an official exhaustive region registry."))
		}
		if len(data.Languages) > 0 {
			fmt.Println()
			fmt.Println(ui.Section("Languages for --lang"))
			rows := make([][]string, 0, len(data.Languages))
			for _, language := range data.Languages {
				rows = append(rows, []string{language.Code, language.Name})
			}
			fmt.Println(ui.Table([]string{"Code", "Language"}, rows))
		}
		return nil
	}),
}

func init() {
	localesCmd.Flags().StringVar(&localesOpts.kind, "type", "all", "which locale list to show: all, regions, languages")
	localesCmd.Flags().BoolVar(&localesOpts.live, "live", false, "fetch languages from the Steam Store language menu")
	localesCmd.Flags().BoolVar(&localesOpts.probe, "probe", false, "probe listed regions against Steam appdetails pricing")
	localesCmd.Flags().IntVar(&localesOpts.appid, "appid", 264710, "appid used by --probe")
}
