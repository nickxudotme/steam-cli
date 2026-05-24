package cmd

import (
	"fmt"
	"strings"
	"time"

	"steam-cli/internal/i18n"
	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var dealsOpts struct {
	count  int
	filter string
	any    string
	all    string
}

var dealsCmd = &cobra.Command{
	Use:     "deals",
	Aliases: []string{"sale", "sales", "topsellers"},
	Short:   "Show Steam store lists such as specials, top sellers, new releases, and upcoming games",
	Example: "  steam-cli deals\n  steam-cli deals --filter topsellers --any discounted,preorder --count 10\n  steam-cli deals --filter topsellers --all discounted --count 10",
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		if err := validateEnumFlag("filter", dealsOpts.filter, "specials", "discountedtopsellers", "preordertopsellers", "topsellers", "new", "comingsoon"); err != nil {
			return nil, err
		}
		anyConditions, err := parseStoreResultConditions("any", dealsOpts.any)
		if err != nil {
			return nil, err
		}
		allConditions, err := parseStoreResultConditions("all", dealsOpts.all)
		if err != nil {
			return nil, err
		}
		items, err := client().StoreResultsQuery(steam.StoreResultsQuery{
			Filter: dealsOpts.filter,
			Count:  dealsOpts.count,
			Any:    anyConditions,
			All:    allConditions,
		})
		if err != nil {
			return nil, err
		}
		return items, nil
	}, func(value any) error {
		items := value.([]steam.StoreResult)
		if len(items) == 0 {
			fmt.Println(ui.Muted.Render(i18n.T("message.no_results")))
			return nil
		}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{
				fmt.Sprintf("%d", item.AppID),
				empty(item.Name),
				empty(item.Release),
				truncate(item.Review, 36),
				empty(item.Discount),
				dealDiscountEndText(item.DiscountEnd),
				empty(item.Final),
			})
		}
		fmt.Println(ui.Table([]string{i18n.T("table.appid"), i18n.T("table.name"), i18n.T("table.release"), i18n.T("table.review"), i18n.T("table.discount"), i18n.T("table.discount_ends"), i18n.T("table.price")}, rows))
		return nil
	}),
}

func init() {
	dealsCmd.Flags().IntVar(&dealsOpts.count, "count", 20, "number of items")
	dealsCmd.Flags().StringVar(&dealsOpts.filter, "filter", "specials", "list filter: specials, discountedtopsellers, preordertopsellers, topsellers, new, comingsoon")
	dealsCmd.Flags().StringVar(&dealsOpts.any, "any", "", "comma-separated conditions where any may match: discounted, preorder")
	dealsCmd.Flags().StringVar(&dealsOpts.all, "all", "", "comma-separated conditions where all must match: discounted, preorder")
}

func parseStoreResultConditions(flagName, raw string) ([]steam.StoreResultCondition, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	conditions := make([]steam.StoreResultCondition, 0, len(parts))
	seen := map[steam.StoreResultCondition]bool{}
	for _, part := range parts {
		value := steam.StoreResultCondition(strings.ToLower(strings.TrimSpace(part)))
		if value == "" {
			continue
		}
		switch value {
		case steam.StoreResultConditionDiscounted, steam.StoreResultConditionPreorder:
			if !seen[value] {
				conditions = append(conditions, value)
				seen[value] = true
			}
		default:
			return nil, &steam.Error{
				Code:    steam.CodeInvalidInput,
				HintKey: "hint.invalid_enum",
				Message: fmt.Sprintf("invalid --%s condition %q; expected one of: discounted, preorder", flagName, value),
			}
		}
	}
	return conditions, nil
}

func dealDiscountEndText(value int64) string {
	if value <= 0 {
		return "-"
	}
	return time.Unix(value, 0).Format(localTimeFormat)
}
