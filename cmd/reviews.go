package cmd

import (
	"fmt"
	"strconv"
	"time"

	"steam-cli/internal/i18n"
	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var reviewOpts struct {
	count        int
	filter       string
	reviewType   string
	purchaseType string
	cursor       string
}

var reviewsCmd = &cobra.Command{
	Use:     "reviews APPID",
	Short:   "Show Steam user reviews for an app",
	Example: "  steam-cli search portal\n  steam-cli reviews 620 --count 5\n  steam-cli reviews 620 --filter all --type positive",
	Args:    exactArgsWithExample(1, "steam-cli reviews APPID [--count N]", "steam-cli reviews 620 --count 5"),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		if err := validateEnumFlag("filter", reviewOpts.filter, "recent", "updated", "all"); err != nil {
			return nil, err
		}
		if err := validateEnumFlag("type", reviewOpts.reviewType, "all", "positive", "negative"); err != nil {
			return nil, err
		}
		if err := validateEnumFlag("purchase", reviewOpts.purchaseType, "all", "steam", "non_steam_purchase"); err != nil {
			return nil, err
		}
		appid, err := parseAppID(args[0])
		if err != nil {
			return nil, err
		}
		resp, err := client().Reviews(appid, steam.ReviewQuery{
			Count:        reviewOpts.count,
			Language:     opts.lang,
			Filter:       reviewOpts.filter,
			ReviewType:   reviewOpts.reviewType,
			PurchaseType: reviewOpts.purchaseType,
			Cursor:       reviewOpts.cursor,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"appid": appid, "response": resp}, nil
	}, func(value any) error {
		data := value.(map[string]any)
		appid := data["appid"].(int)
		resp := data["response"].(*steam.ReviewResponse)

		fmt.Println(ui.Title.Render(fmt.Sprintf(i18n.T("title.reviews_for"), appid)))
		fmt.Println(ui.Table(
			[]string{i18n.T("table.summary"), i18n.T("table.positive"), i18n.T("table.negative"), i18n.T("table.total"), i18n.T("table.next_cursor")},
			[][]string{{
				empty(resp.QuerySummary.ReviewScoreDesc),
				strconv.Itoa(resp.QuerySummary.TotalPositive),
				strconv.Itoa(resp.QuerySummary.TotalNegative),
				strconv.Itoa(resp.QuerySummary.TotalReviews),
				empty(resp.Cursor),
			}},
		))
		if len(resp.Reviews) == 0 {
			fmt.Println()
			fmt.Println(ui.Muted.Render(i18n.T("message.no_results")))
			return nil
		}
		rows := make([][]string, 0, len(resp.Reviews))
		for _, review := range resp.Reviews {
			rows = append(rows, []string{
				time.Unix(review.TimestampCreated, 0).Format(time.DateOnly),
				thumb(review.VotedUp),
				minutesText(review.Author.PlaytimeAtReview),
				strconv.Itoa(review.VotesUp),
				truncate(review.Review, 120),
			})
		}
		fmt.Println()
		fmt.Println(ui.Table([]string{i18n.T("table.date"), i18n.T("table.vote"), i18n.T("table.playtime"), i18n.T("table.helpful"), i18n.T("table.review")}, rows))
		return nil
	}),
}

func init() {
	reviewsCmd.Flags().IntVar(&reviewOpts.count, "count", 5, "number of reviews")
	reviewsCmd.Flags().StringVar(&reviewOpts.filter, "filter", "recent", "review filter: recent, updated, all")
	reviewsCmd.Flags().StringVar(&reviewOpts.reviewType, "type", "all", "review type: all, positive, negative")
	reviewsCmd.Flags().StringVar(&reviewOpts.purchaseType, "purchase", "all", "purchase type: all, steam, non_steam_purchase")
	reviewsCmd.Flags().StringVar(&reviewOpts.cursor, "cursor", "", "pagination cursor from previous response")
}

func thumb(votedUp bool) string {
	if votedUp {
		return "up"
	}
	return "down"
}

func minutesText(minutes int) string {
	if minutes <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1fh", float64(minutes)/60)
}
