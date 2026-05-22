package cmd

import (
	"fmt"
	"time"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var newsCount int

var newsCmd = &cobra.Command{
	Use:   "news APPID",
	Short: "Show Steam news and announcements for an app",
	Args:  cobra.ExactArgs(1),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		appid, err := parseAppID(args[0])
		if err != nil {
			return nil, err
		}
		items, err := client().News(appid, newsCount)
		if err != nil {
			return nil, err
		}
		return items, nil
	}, func(value any) error {
		items := value.([]steam.NewsItem)
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			feed := item.FeedLabel
			if feed == "" {
				feed = item.FeedName
			}
			rows = append(rows, []string{
				time.Unix(item.Date, 0).Format(time.DateOnly),
				empty(item.Title),
				empty(feed),
				truncate(item.Contents, 120),
				empty(item.URL),
			})
		}
		fmt.Println(ui.Table([]string{"Date", "Title", "Feed", "Summary", "URL"}, rows))
		return nil
	}),
}

func init() {
	newsCmd.Flags().IntVar(&newsCount, "count", 10, "number of news items")
}
