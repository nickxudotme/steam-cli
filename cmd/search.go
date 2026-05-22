package cmd

import (
	"fmt"
	"strconv"

	"steam-cli/internal/i18n"
	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var searchCount int

var searchCmd = &cobra.Command{
	Use:     "search TERM",
	Aliases: []string{"find", "lookup"},
	Short:   "Search Steam store games",
	Example: "  steam-cli search \"elden ring\"\n  steam-cli search portal --count 5\n  steam-cli app 1245620",
	Args:    exactArgsWithExample(1, "steam-cli search TERM [--count N]", "steam-cli search portal --count 5"),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		items, err := client().Search(args[0], searchCount)
		if err != nil {
			return nil, err
		}
		return items, nil
	}, func(value any) error {
		items := value.([]steam.SearchItem)
		if opts.quiet {
			for _, item := range items {
				fmt.Printf("%d\t%s\n", item.ID, item.Name)
			}
			return nil
		}
		rows := make([][]string, 0, len(items))
		if len(items) == 0 {
			fmt.Println(ui.Muted.Render(i18n.T("message.no_results")))
			return nil
		}
		for _, item := range items {
			rows = append(rows, []string{
				strconv.Itoa(item.ID),
				empty(item.Name),
				searchPrice(item.Price),
				discountText(item.Price),
			})
		}
		fmt.Println(ui.Table([]string{i18n.T("table.appid"), i18n.T("table.name"), i18n.T("table.price"), i18n.T("table.discount")}, rows))
		return nil
	}),
}

func init() {
	searchCmd.Flags().IntVar(&searchCount, "count", 10, "number of results")
}

func searchPrice(price *steam.PriceOverview) string {
	if price == nil {
		return "-"
	}
	if price.FinalFormatted != "" {
		return price.FinalFormatted
	}
	return ui.Money(price.Final, price.Currency)
}

func discountText(price *steam.PriceOverview) string {
	if price == nil || price.DiscountPercent == 0 {
		return "-"
	}
	return strconv.Itoa(price.DiscountPercent) + "%"
}
