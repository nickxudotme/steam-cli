package cmd

import (
	"fmt"
	"strconv"
	"time"

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
	Use:   "reviews APPID",
	Short: "Show Steam user reviews for an app",
	Args:  cobra.ExactArgs(1),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
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

		fmt.Println(ui.Title.Render(fmt.Sprintf("Reviews for %d", appid)))
		fmt.Println(ui.Table(
			[]string{"Summary", "Positive", "Negative", "Total", "Next Cursor"},
			[][]string{{
				empty(resp.QuerySummary.ReviewScoreDesc),
				strconv.Itoa(resp.QuerySummary.TotalPositive),
				strconv.Itoa(resp.QuerySummary.TotalNegative),
				strconv.Itoa(resp.QuerySummary.TotalReviews),
				empty(resp.Cursor),
			}},
		))
		if len(resp.Reviews) == 0 {
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
		fmt.Println(ui.Table([]string{"Date", "Vote", "Playtime", "Helpful", "Review"}, rows))
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
