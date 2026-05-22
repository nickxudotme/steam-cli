package cmd

import (
	"fmt"
	"strconv"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var dlcCmd = &cobra.Command{
	Use:   "dlc APPID",
	Short: "List DLC for a Steam app with current store data",
	Args:  cobra.ExactArgs(1),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		appid, err := parseAppID(args[0])
		if err != nil {
			return nil, err
		}
		return client().DLC(appid)
	}, func(value any) error {
		result := value.(*steam.DLCResult)
		fmt.Println(ui.Title.Render(fmt.Sprintf("%s DLC (%d)", result.Name, result.AppID)))
		if len(result.Items) == 0 {
			fmt.Println()
			fmt.Println(ui.Muted.Render("No public DLC found."))
			return nil
		}
		rows := make([][]string, 0, len(result.Items))
		for _, item := range result.Items {
			rows = append(rows, []string{
				strconv.Itoa(item.AppID),
				empty(item.Name),
				storeDate(releaseDate(item.Release)),
				storePrice(item.BestPurchaseOption),
				storeReviewText(item.Reviews),
			})
		}
		fmt.Println()
		fmt.Println(ui.Table([]string{"AppID", "Name", "Release", "Price", "Reviews"}, rows))
		return nil
	}),
}

func releaseDate(release *steam.StoreRelease) int64 {
	if release == nil {
		return 0
	}
	if release.SteamReleaseDate > 0 {
		return release.SteamReleaseDate
	}
	if release.OriginalSteamReleaseDate > 0 {
		return release.OriginalSteamReleaseDate
	}
	return release.OriginalReleaseDate
}
