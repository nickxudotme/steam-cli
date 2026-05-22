package cmd

import (
	"fmt"
	"strconv"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var mediaOpts struct {
	probe bool
}

var mediaCmd = &cobra.Command{
	Use:     "media APPID",
	Aliases: []string{"images", "assets"},
	Short:   "Show Steam app images, screenshots, trailers, and media assets",
	Args:    cobra.ExactArgs(1),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		appid, err := parseAppID(args[0])
		if err != nil {
			return nil, err
		}
		media, err := client().Media(appid, mediaOpts.probe)
		if err != nil {
			return nil, err
		}
		return media, nil
	}, func(value any) error {
		media := value.(*steam.Media)
		fmt.Println(ui.Title.Render(fmt.Sprintf("%s (%d)", media.Name, media.AppID)))
		if media.HeaderImage != "" {
			fmt.Println()
			fmt.Println(ui.Accent.Render("Header image: ") + media.HeaderImage)
		}

		fmt.Println()
		fmt.Println(ui.Accent.Render("CDN assets"))
		assetRows := make([][]string, 0, len(media.CDNAssets))
		for _, asset := range media.CDNAssets {
			status := "-"
			if mediaOpts.probe {
				status = fmt.Sprintf("%d", asset.Status)
			}
			assetRows = append(assetRows, []string{asset.Kind, asset.Name, status, asset.URL})
		}
		fmt.Println(ui.Table([]string{"Kind", "Name", "Status", "URL"}, assetRows))

		if len(media.Screenshots) > 0 {
			fmt.Println()
			fmt.Println(ui.Accent.Render(fmt.Sprintf("Screenshots (%d)", len(media.Screenshots))))
			rows := make([][]string, 0, len(media.Screenshots))
			for _, shot := range media.Screenshots {
				rows = append(rows, []string{strconv.Itoa(shot.ID), shot.PathFull})
			}
			fmt.Println(ui.Table([]string{"ID", "Full URL"}, rows))
		}

		if len(media.Movies) > 0 {
			fmt.Println()
			fmt.Println(ui.Accent.Render(fmt.Sprintf("Movie thumbnails (%d)", len(media.Movies))))
			rows := make([][]string, 0, len(media.Movies))
			for _, movie := range media.Movies {
				rows = append(rows, []string{strconv.Itoa(movie.ID), truncate(movie.Name, 40), movie.Thumbnail})
			}
			fmt.Println(ui.Table([]string{"ID", "Name", "Thumbnail"}, rows))
		}

		if len(media.AchievementIcons) > 0 {
			fmt.Println()
			fmt.Println(ui.Accent.Render(fmt.Sprintf("Achievement icons (%d)", len(media.AchievementIcons))))
			rows := make([][]string, 0, len(media.AchievementIcons))
			for _, achievement := range media.AchievementIcons {
				rows = append(rows, []string{truncate(achievement.LocalizedName, 32), achievement.Path})
			}
			fmt.Println(ui.Table([]string{"Achievement", "Icon"}, rows))
		}
		return nil
	}),
}

func init() {
	mediaCmd.Flags().BoolVar(&mediaOpts.probe, "probe", false, "HEAD probe fixed CDN assets and include status/availability")
}
