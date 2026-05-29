package cmd

import (
	"fmt"
	"strings"

	"steam-cli/internal/i18n"
	"steam-cli/internal/itad"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var storesSearch string

var storesCmd = &cobra.Command{
	Use:     "stores",
	Aliases: []string{"shops"},
	Short:   "List shop mappings used by the advanced pricing backend",
	Hidden:  true,
	Args:    cobra.NoArgs,
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		shops, err := itadClient().Shops()
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(storesSearch) == "" {
			return shops, nil
		}
		filtered := make([]itad.Shop, 0, len(shops))
		needle := strings.ToLower(strings.TrimSpace(storesSearch))
		for _, shop := range shops {
			if strings.Contains(strings.ToLower(shop.Title), needle) {
				filtered = append(filtered, shop)
			}
		}
		return filtered, nil
	}, func(value any) error {
		shops := value.([]itad.Shop)
		if opts.quiet {
			for _, shop := range shops {
				fmt.Printf("%d\t%s\n", shop.ID, shop.Title)
			}
			return nil
		}
		if len(shops) == 0 {
			fmt.Println(ui.Muted.Render(i18n.T("message.no_results")))
			return nil
		}
		rows := make([][]string, 0, len(shops))
		for _, shop := range shops {
			rows = append(rows, []string{
				fmt.Sprintf("%d", shop.ID),
				shop.Title,
				fmt.Sprintf("%d", shop.Deals),
				fmt.Sprintf("%d", shop.Games),
				empty(formatITADDate(ptrString(shop.Update))),
			})
		}
		fmt.Println(ui.Table([]string{"ID", i18n.T("table.shop"), i18n.T("table.deals"), i18n.T("table.games"), i18n.T("table.date")}, rows))
		return nil
	}),
}

func init() {
	storesCmd.Flags().StringVar(&storesSearch, "search", "", "filter shops by case-insensitive title match")
}
