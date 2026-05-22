package cmd

import (
	"fmt"
	"strconv"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var similarOpts struct {
	count int
}

var similarCmd = &cobra.Command{
	Use:     "similar APPID",
	Aliases: []string{"recommend", "recommendations"},
	Short:   "Show Steam store recommendations similar to an app",
	Args:    cobra.ExactArgs(1),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		appid, err := parseAppID(args[0])
		if err != nil {
			return nil, err
		}
		return client().Similar(appid, similarOpts.count)
	}, func(value any) error {
		result := value.(*steam.SimilarResult)
		fmt.Println(ui.Title.Render(fmt.Sprintf("Similar to %d", result.AppID)))
		if len(result.Items) == 0 {
			fmt.Println()
			fmt.Println(ui.Muted.Render("No similar items returned."))
			return nil
		}
		rows := make([][]string, 0, len(result.Items))
		for _, item := range result.Items {
			rows = append(rows, []string{
				strconv.Itoa(item.AppID),
				empty(item.Name),
				storePrice(item.BestPurchaseOption),
				storeReviewText(item.Reviews),
				storePlatformsText(item.Platforms),
			})
		}
		fmt.Println()
		fmt.Println(ui.Table([]string{"AppID", "Name", "Price", "Reviews", "Platforms"}, rows))
		return nil
	}),
}

func init() {
	similarCmd.Flags().IntVar(&similarOpts.count, "count", 10, "number of similar games")
}
