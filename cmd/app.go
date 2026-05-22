package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"steam-cli/internal/i18n"
	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var appNewsCount int

var appCmd = &cobra.Command{
	Use:     "app APPID",
	Aliases: []string{"game", "info"},
	Short:   "Show public info for a Steam app",
	Example: "  steam-cli search \"subnautica\"\n  steam-cli app 264710\n  steam-cli app 264710 --news 0 --json",
	Args:    exactArgsWithExample(1, "steam-cli app APPID", "steam-cli app 264710"),
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
			[]string{i18n.T("table.type"), i18n.T("table.release"), i18n.T("table.free"), i18n.T("table.players_now"), "Metacritic", "Deck"},
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
		fmt.Println(ui.KeyValue(i18n.T("table.price"), priceDetail(details)))
		if text := discountEndText(bundle.StoreItem); text != "" {
			fmt.Println(ui.KeyValue(i18n.T("table.discount_ends"), text))
		}
		if details.ShortDescription != "" {
			fmt.Println()
			fmt.Println(ui.KeyValue(i18n.T("table.summary"), details.ShortDescription))
		}
		fmt.Println()
		fmt.Println(ui.Table(
			[]string{i18n.T("table.store_reviews"), i18n.T("table.app_reviews"), i18n.T("table.positive"), i18n.T("table.negative"), i18n.T("table.total"), i18n.T("table.recommendations"), i18n.T("table.achievements")},
			[][]string{{
				storeReviewText(storeReviews(bundle.StoreItem)),
				reviewScoreText(bundle.Reviews),
				reviewCountText(bundle.Reviews, func(r *steam.ReviewSummary) int { return r.TotalPositive }),
				reviewCountText(bundle.Reviews, func(r *steam.ReviewSummary) int { return r.TotalNegative }),
				reviewCountText(bundle.Reviews, func(r *steam.ReviewSummary) int { return r.TotalReviews }),
				recommendationsText(details.Recommendations),
				achievementsText(details.Achievements),
			}},
		))
		fmt.Println()
		fmt.Println(ui.Section(i18n.T("section.store_profile")))
		fmt.Println(ui.KeyValue(i18n.T("label.developers"), ui.Join(details.Developers)))
		fmt.Println(ui.KeyValue(i18n.T("label.publishers"), ui.Join(details.Publishers)))
		fmt.Println(ui.KeyValue(i18n.T("label.genres"), namedValues(details.Genres)))
		fmt.Println(ui.KeyValue(i18n.T("label.categories"), namedValues(details.Categories)))
		fmt.Println(ui.KeyValue(i18n.T("label.platforms"), platformsText(details.Platforms)))
		fmt.Println(ui.KeyValue(i18n.T("label.controller"), empty(details.ControllerSupport)))
		fmt.Println(ui.KeyValue(i18n.T("label.required_age"), empty(string(details.RequiredAge))))
		fmt.Println(ui.KeyValue(i18n.T("label.languages"), languageText(details, bundle.StoreItem)))
		if bundle.StoreItem != nil {
			if tags := storeTagsText(bundle.StoreItem.Tags, 8); tags != "-" {
				fmt.Println(ui.KeyValue(i18n.T("label.store_tags"), tags))
			}
			if bundle.StoreItem.GameRating != nil && bundle.StoreItem.GameRating.Rating != "" {
				fmt.Println(ui.KeyValue(i18n.T("label.rating"), ratingText(bundle.StoreItem.GameRating)))
			}
		}
		if len(details.DLC) > 0 {
			fmt.Println(ui.KeyValue(i18n.T("label.dlc_appids"), intSliceText(details.DLC)))
		}
		if bundle.StoreItem != nil && len(bundle.StoreItem.PurchaseOptions) > 0 {
			fmt.Println()
			fmt.Println(ui.Section(i18n.T("section.purchase_options")))
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
			fmt.Println(ui.Table([]string{"ID", i18n.T("table.option"), i18n.T("table.original"), i18n.T("table.final"), i18n.T("table.discount"), i18n.T("table.ends")}, rows))
		}
		if len(details.Screenshots) > 0 || len(details.Movies) > 0 {
			fmt.Println()
			fmt.Println(ui.Section(i18n.T("section.media")))
			fmt.Printf("%s: %d\n", i18n.T("label.screenshots"), len(details.Screenshots))
			if len(details.Screenshots) > 0 {
				fmt.Println(i18n.T("label.first_screenshot") + ": " + details.Screenshots[0].PathFull)
			}
			fmt.Printf("%s: %d\n", i18n.T("label.movies"), len(details.Movies))
			if len(details.Movies) > 0 {
				fmt.Println(i18n.T("label.first_movie") + ": " + details.Movies[0].Name)
			}
		}
		if details.SupportInfo.URL != "" || details.SupportInfo.Email != "" {
			fmt.Println()
			fmt.Println(ui.Section(i18n.T("section.support")))
			if details.SupportInfo.URL != "" {
				fmt.Println(i18n.T("label.url") + ": " + details.SupportInfo.URL)
			}
			if details.SupportInfo.Email != "" {
				fmt.Println(i18n.T("label.email") + ": " + details.SupportInfo.Email)
			}
		}
		if bundle.StoreItem != nil && len(bundle.StoreItem.Links) > 0 {
			fmt.Println()
			fmt.Println(ui.Section(i18n.T("section.official_links")))
			fmt.Println(ui.Table([]string{i18n.T("table.type"), "URL"}, storeLinkRows(bundle.StoreItem.Links)))
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
			fmt.Println(ui.Section(i18n.T("section.news")))
			fmt.Println(ui.Table([]string{i18n.T("table.date"), i18n.T("table.title"), i18n.T("table.feed")}, rows))
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

func reviewScoreText(summary *steam.ReviewSummary) string {
	if summary == nil {
		return "-"
	}
	return empty(summary.ReviewScoreDesc)
}

func reviewCountText(summary *steam.ReviewSummary, pick func(*steam.ReviewSummary) int) string {
	if summary == nil {
		return "-"
	}
	return strconv.Itoa(pick(summary))
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
