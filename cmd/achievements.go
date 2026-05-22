package cmd

import (
	"fmt"
	"sort"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var achievementCount int

var achievementsCmd = &cobra.Command{
	Use:   "achievements APPID",
	Short: "Show global achievement unlock percentages for an app",
	Args:  cobra.ExactArgs(1),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		appid, err := parseAppID(args[0])
		if err != nil {
			return nil, err
		}
		items, err := client().GlobalAchievements(appid)
		if err != nil {
			return nil, err
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].Percent > items[j].Percent
		})
		if achievementCount > 0 && achievementCount < len(items) {
			items = items[:achievementCount]
		}
		return items, nil
	}, func(value any) error {
		items := value.([]steam.GlobalAchievement)
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{item.Name, fmt.Sprintf("%.1f%%", item.Percent)})
		}
		fmt.Println(ui.Table([]string{"Achievement", "Global Unlock"}, rows))
		return nil
	}),
}

func init() {
	achievementsCmd.Flags().IntVar(&achievementCount, "count", 20, "number of achievements")
}
