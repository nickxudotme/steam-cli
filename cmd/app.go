package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var appNewsCount int

var appCmd = &cobra.Command{
	Use:     "app APPID",
	Aliases: []string{"game", "info"},
	Short:   "Show public info for a Steam app",
	Args:    cobra.ExactArgs(1),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		appid, err := parseAppID(args[0])
		if err != nil {
			return nil, err
		}

		bundle, err := client().AppBundle(appid, appNewsCount)
		if err != nil {
			return nil, err
		}
		return bundle, nil
	}, func(value any) error {
		bundle := value.(*steam.AppBundle)
		appid := bundle.AppID

		details := bundle.Details
		fmt.Println(ui.Title.Render(fmt.Sprintf("%s (%d)", details.Name, appid)))
		fmt.Println()
		fmt.Println(ui.Table(
			[]string{"Type", "Release", "Free", "Players Now", "Metacritic", "Deck"},
			[][]string{{
				empty(details.Type),
				empty(details.ReleaseDate.Date),
				yesNo(details.IsFree),
				intPtrText(bundle.CurrentPlayers),
				scoreText(details.Metacritic),
				deckText(bundle.StoreItem),
			}},
		))
		fmt.Println()
		fmt.Println(ui.KeyValue("Price", priceDetail(details)))
		if text := discountEndText(bundle.StoreItem); text != "" {
			fmt.Println(ui.KeyValue("Discount ends", text))
		}
		if details.ShortDescription != "" {
			fmt.Println()
			fmt.Println(ui.KeyValue("Summary", details.ShortDescription))
		}
		fmt.Println()
		fmt.Println(ui.Table(
			[]string{"Store Reviews", "App Reviews", "Positive", "Negative", "Total", "Recommendations", "Achievements"},
			[][]string{{
				storeReviewText(storeReviews(bundle.StoreItem)),
				empty(bundle.Reviews.ReviewScoreDesc),
				strconv.Itoa(bundle.Reviews.TotalPositive),
				strconv.Itoa(bundle.Reviews.TotalNegative),
				strconv.Itoa(bundle.Reviews.TotalReviews),
				recommendationsText(details.Recommendations),
				achievementsText(details.Achievements),
			}},
		))
		fmt.Println()
		fmt.Println(ui.Section("Store profile"))
		fmt.Println(ui.KeyValue("Developers", ui.Join(details.Developers)))
		fmt.Println(ui.KeyValue("Publishers", ui.Join(details.Publishers)))
		fmt.Println(ui.KeyValue("Genres", namedValues(details.Genres)))
		fmt.Println(ui.KeyValue("Categories", namedValues(details.Categories)))
		fmt.Println(ui.KeyValue("Platforms", platformsText(details.Platforms)))
		fmt.Println(ui.KeyValue("Controller", empty(details.ControllerSupport)))
		fmt.Println(ui.KeyValue("Required age", empty(string(details.RequiredAge))))
		fmt.Println(ui.KeyValue("Languages", languageText(details, bundle.StoreItem)))
		if bundle.StoreItem != nil {
			if tags := storeTagsText(bundle.StoreItem.Tags, 8); tags != "-" {
				fmt.Println(ui.KeyValue("Store tags", tags))
			}
			if bundle.StoreItem.GameRating != nil && bundle.StoreItem.GameRating.Rating != "" {
				fmt.Println(ui.KeyValue("Rating", ratingText(bundle.StoreItem.GameRating)))
			}
		}
		if len(details.DLC) > 0 {
			fmt.Println(ui.KeyValue("DLC AppIDs", intSliceText(details.DLC)))
		}
		if bundle.StoreItem != nil && len(bundle.StoreItem.PurchaseOptions) > 0 {
			fmt.Println()
			fmt.Println(ui.Section("Purchase options"))
			rows := make([][]string, 0, len(bundle.StoreItem.PurchaseOptions))
			for _, option := range bundle.StoreItem.PurchaseOptions {
				rows = append(rows, []string{
					purchaseOptionID(option),
					empty(option.PurchaseOptionName),
					empty(option.FormattedOriginalPrice),
					empty(option.FormattedFinalPrice),
					discountPctText(option.DiscountPct),
					purchaseOptionDiscountEndText(option),
				})
			}
			fmt.Println(ui.Table([]string{"ID", "Option", "Original", "Final", "Discount", "Ends"}, rows))
		}
		if len(details.Screenshots) > 0 || len(details.Movies) > 0 {
			fmt.Println()
			fmt.Println(ui.Section("Media"))
			fmt.Printf("Screenshots: %d\n", len(details.Screenshots))
			if len(details.Screenshots) > 0 {
				fmt.Println("First screenshot: " + details.Screenshots[0].PathFull)
			}
			fmt.Printf("Movies: %d\n", len(details.Movies))
			if len(details.Movies) > 0 {
				fmt.Println("First movie: " + details.Movies[0].Name)
			}
		}
		if details.SupportInfo.URL != "" || details.SupportInfo.Email != "" {
			fmt.Println()
			fmt.Println(ui.Section("Support"))
			if details.SupportInfo.URL != "" {
				fmt.Println("URL: " + details.SupportInfo.URL)
			}
			if details.SupportInfo.Email != "" {
				fmt.Println("Email: " + details.SupportInfo.Email)
			}
		}
		if bundle.StoreItem != nil && len(bundle.StoreItem.Links) > 0 {
			fmt.Println()
			fmt.Println(ui.Section("Official links"))
			fmt.Println(ui.Table([]string{"Type", "URL"}, storeLinkRows(bundle.StoreItem.Links)))
		}

		if len(bundle.News) > 0 {
			rows := make([][]string, 0, len(bundle.News))
			for _, item := range bundle.News {
				feed := item.FeedLabel
				if feed == "" {
					feed = item.FeedName
				}
				rows = append(rows, []string{
					time.Unix(item.Date, 0).Format(time.DateOnly),
					empty(item.Title),
					empty(feed),
				})
			}
			fmt.Println()
			fmt.Println(ui.Section("News"))
			fmt.Println(ui.Table([]string{"Date", "Title", "Feed"}, rows))
		}
		if len(bundle.Warnings) > 0 {
			fmt.Println()
			fmt.Println(ui.Muted.Render("Warnings (sibling lookups failed; primary data still rendered above):"))
			for _, w := range bundle.Warnings {
				fmt.Println(ui.Muted.Render("  - " + w))
			}
		}
		return nil
	}),
}

func init() {
	appCmd.Flags().IntVar(&appNewsCount, "news", 3, "number of news items to include")
}

func empty(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func scoreText(value *steam.Metacritic) string {
	if value == nil || value.Score == 0 {
		return "-"
	}
	return strconv.Itoa(value.Score)
}

func storeReviews(item *steam.StoreItem) *steam.StoreReviews {
	if item == nil {
		return nil
	}
	return item.Reviews
}

func deckText(item *steam.StoreItem) string {
	if item == nil || item.Platforms == nil || item.Platforms.SteamDeckCompatCategory == 0 {
		return "-"
	}
	switch item.Platforms.SteamDeckCompatCategory {
	case 1:
		return "unsupported"
	case 2:
		return "playable"
	case 3:
		return "verified"
	default:
		return strconv.Itoa(item.Platforms.SteamDeckCompatCategory)
	}
}

func ratingText(rating *steam.GameRating) string {
	parts := []string{}
	if rating.Type != "" {
		parts = append(parts, strings.ToUpper(rating.Type))
	}
	if rating.Rating != "" {
		parts = append(parts, rating.Rating)
	}
	if len(rating.Descriptors) > 0 {
		parts = append(parts, strings.Join(rating.Descriptors, ", "))
	}
	return strings.Join(parts, " - ")
}

func languageText(details *steam.AppDetails, item *steam.StoreItem) string {
	if item != nil && len(item.SupportedLanguages) > 0 {
		return storeLanguageSummary(item.SupportedLanguages)
	}
	return compactLanguages(details.SupportedLanguages)
}

func recommendationsText(value *steam.Recommendations) string {
	if value == nil || value.Total == 0 {
		return "-"
	}
	return strconv.Itoa(value.Total)
}

func achievementsText(value *steam.AppAchievements) string {
	if value == nil || value.Total == 0 {
		return "-"
	}
	return strconv.Itoa(value.Total)
}

func platformsText(platforms map[string]bool) string {
	values := []string{}
	for _, name := range []string{"windows", "mac", "linux"} {
		if platforms[name] {
			values = append(values, name)
		}
	}
	return ui.Join(values)
}

func intSliceText(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ", ")
}

func compactLanguages(value string) string {
	value = strings.ReplaceAll(value, "<br>", " ")
	value = strings.ReplaceAll(value, "<strong>*</strong>", "*")
	value = strings.ReplaceAll(value, "<strong>", "")
	value = strings.ReplaceAll(value, "</strong>", "")
	return empty(strings.Join(strings.Fields(value), " "))
}
