package cmd

import (
	"fmt"
	"time"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var wishlistOpts struct {
	count     int
	offset    int
	noDetails bool
}

var wishlistCmd = &cobra.Command{
	Use:   "wishlist STEAMID64|VANITY|PROFILE_URL",
	Short: "Show a public Steam user's wishlist",
	Args:  cobra.ExactArgs(1),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		list, err := client().WishlistWithDetails(args[0], wishlistOpts.offset, wishlistOpts.count, !wishlistOpts.noDetails)
		if err != nil {
			return nil, err
		}
		return list, nil
	}, func(value any) error {
		list := value.(*steam.Wishlist)

		fmt.Println(ui.Title.Render(fmt.Sprintf("Wishlist %s", list.SteamID64)))
		fmt.Println(ui.Muted.Render(fmt.Sprintf("Showing %d of %d items from offset %d", list.Count, list.Total, list.Offset)))
		fmt.Println()

		rows := make([][]string, 0, len(list.Items))
		for _, item := range list.Items {
			rows = append(rows, []string{
				fmt.Sprintf("%d", item.AppID),
				wishlistName(item),
				wishlistRelease(item.Details),
				wishlistPrice(item.Details),
				wishlistDiscount(item.Details),
				formatWishlistDate(item.DateAdded),
			})
		}
		fmt.Println(ui.Table(
			[]string{"AppID", "Name", "Release", "Price", "Discount", "Added"},
			rows,
		))
		return nil
	}),
}

func init() {
	wishlistCmd.Flags().IntVar(&wishlistOpts.count, "count", 20, "number of wishlist items to display; 0 shows all")
	wishlistCmd.Flags().IntVar(&wishlistOpts.offset, "offset", 0, "start offset in the wishlist")
	wishlistCmd.Flags().BoolVar(&wishlistOpts.noDetails, "no-details", false, "skip appdetails lookups and show only wishlist appids")
}

func wishlistName(item steam.WishlistItem) string {
	if item.Details != nil && item.Details.Name != "" {
		return truncate(item.Details.Name, 42)
	}
	if item.Error != "" {
		return ui.Muted.Render("details unavailable")
	}
	return "-"
}

func wishlistRelease(details *steam.AppDetails) string {
	if details == nil {
		return "-"
	}
	if details.ReleaseDate.ComingSoon {
		return "Coming soon"
	}
	return empty(details.ReleaseDate.Date)
}

func wishlistPrice(details *steam.AppDetails) string {
	if details == nil {
		return "-"
	}
	return priceText(details)
}

func wishlistDiscount(details *steam.AppDetails) string {
	if details == nil || details.PriceOverview == nil || details.PriceOverview.DiscountPercent == 0 {
		return "-"
	}
	return fmt.Sprintf("-%d%%", details.PriceOverview.DiscountPercent)
}

func formatWishlistDate(unix int64) string {
	if unix <= 0 {
		return "-"
	}
	return time.Unix(unix, 0).Format("2006-01-02")
}
