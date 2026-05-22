package cmd

import (
	"fmt"

	"steam-cli/internal/i18n"
	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var dealsOpts struct {
	count  int
	filter string
}

var dealsCmd = &cobra.Command{
	Use:     "deals",
	Aliases: []string{"sale", "sales", "topsellers"},
	Short:   "Show Steam store lists such as specials, top sellers, new releases, and upcoming games",
	Example: "  steam-cli deals\n  steam-cli deals --filter topsellers --count 10\n  steam-cli deals --filter comingsoon",
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		if err := validateEnumFlag("filter", dealsOpts.filter, "specials", "topsellers", "new", "comingsoon"); err != nil {
			return nil, err
		}
		items, err := client().StoreResults(dealsOpts.filter, dealsOpts.count)
		if err != nil {
			return nil, err
		}
		return items, nil
	}, func(value any) error {
		items := value.([]steam.StoreResult)
		if len(items) == 0 {
			fmt.Println(ui.Muted.Render(i18n.T("message.no_results")))
			return nil
		}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{
				fmt.Sprintf("%d", item.AppID),
				empty(item.Name),
				empty(item.Release),
				truncate(item.Review, 36),
				empty(item.Discount),
				empty(item.Final),
			})
		}
		fmt.Println(ui.Table([]string{i18n.T("table.appid"), i18n.T("table.name"), i18n.T("table.release"), i18n.T("table.review"), i18n.T("table.discount"), i18n.T("table.price")}, rows))
		return nil
	}),
}

func init() {
	dealsCmd.Flags().IntVar(&dealsOpts.count, "count", 20, "number of items")
	dealsCmd.Flags().StringVar(&dealsOpts.filter, "filter", "specials", "list filter: specials, topsellers, new, comingsoon")
}
