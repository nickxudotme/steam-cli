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

var priceOpts struct {
	compare string
}

var priceCmd = &cobra.Command{
	Use:     "price APPID",
	Aliases: []string{"prices"},
	Short:   "Show price for a Steam app",
	Args:    cobra.ExactArgs(1),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		appid, err := parseAppID(args[0])
		if err != nil {
			return nil, err
		}
		if priceOpts.compare != "" {
			return loadPriceComparison(appid, priceOpts.compare)
		}
		details, err := client().AppDetails(appid)
		if err != nil {
			return nil, err
		}
		storeItem, _ := client().StoreItem(appid)
		return priceResult{
			AppID:         appid,
			Name:          details.Name,
			IsFree:        details.IsFree,
			PriceOverview: details.PriceOverview,
			StoreItem:     storeItem,
			details:       details,
		}, nil
	}, func(value any) error {
		if data, ok := value.(*priceComparisonResult); ok {
			renderPriceComparison(data)
			return nil
		}
		data := value.(priceResult)
		details := data.details
		if opts.quiet {
			if details.PriceOverview != nil {
				fmt.Println(details.PriceOverview.FinalFormatted)
			} else {
				fmt.Println(priceText(details))
			}
			return nil
		}
		fmt.Println(ui.Title.Render(fmt.Sprintf("%s (%d)", details.Name, data.AppID)))
		fmt.Println()
		fmt.Println(ui.Accent.Render("Price: ") + priceDetail(details))
		if text := discountEndText(data.StoreItem); text != "" {
			fmt.Println(ui.Accent.Render(i18n.T("price.discount_ends")) + text)
		}
		return nil
	}),
}

func init() {
	priceCmd.Flags().StringVar(&priceOpts.compare, "compare", "", "comma-separated country/region codes to compare, for example CN,US,JP")
}

type priceResult struct {
	AppID         int                  `json:"appid"`
	Name          string               `json:"name"`
	IsFree        bool                 `json:"is_free"`
	PriceOverview *steam.PriceOverview `json:"price_overview,omitempty"`
	StoreItem     *steam.StoreItem     `json:"store_item,omitempty"`
	details       *steam.AppDetails
}

type comparedPrice struct {
	CC              string               `json:"cc"`
	Name            string               `json:"name,omitempty"`
	Available       bool                 `json:"available"`
	PriceOverview   *steam.PriceOverview `json:"price_overview,omitempty"`
	DiscountEnd     int64                `json:"discount_end,omitempty"`
	DiscountEndText string               `json:"discount_end_text,omitempty"`
	Error           string               `json:"error,omitempty"`
}

type priceComparisonResult struct {
	AppID      int             `json:"appid"`
	Name       string          `json:"name"`
	ObservedAt string          `json:"observed_at"`
	Source     string          `json:"source"`
	Confidence string          `json:"confidence"`
	Prices     []comparedPrice `json:"prices"`
}

func loadPriceComparison(appid int, compare string) (*priceComparisonResult, error) {
	baseClient := client()
	baseDetails, err := baseClient.AppDetails(appid)
	if err != nil {
		return nil, err
	}
	result := &priceComparisonResult{
		AppID:      appid,
		Name:       baseDetails.Name,
		ObservedAt: time.Now().UTC().Format(time.RFC3339),
		Source:     "https://store.steampowered.com/api/appdetails + IStoreBrowseService/GetItems/v1",
		Confidence: "official_api_observed",
	}
	for i, cc := range splitCodes(compare) {
		if i > 0 {
			time.Sleep(150 * time.Millisecond)
		}
		regionClient := steam.NewClient(strings.ToUpper(cc), opts.lang, time.Duration(opts.timeout)*time.Second)
		item := comparedPrice{CC: strings.ToUpper(cc)}
		details, err := regionClient.AppDetails(appid)
		if err != nil {
			item.Error = err.Error()
			result.Prices = append(result.Prices, item)
			continue
		}
		item.Name = details.Name
		item.Available = true
		item.PriceOverview = details.PriceOverview
		if storeItem, err := regionClient.StoreItem(appid); err == nil {
			item.DiscountEnd = latestDiscountEndFromStoreItem(storeItem)
			if item.DiscountEnd > 0 {
				item.DiscountEndText = formatDiscountEnd(item.DiscountEnd)
			}
		}
		result.Prices = append(result.Prices, item)
	}
	return result, nil
}

func renderPriceComparison(data *priceComparisonResult) {
	if opts.quiet {
		for _, price := range data.Prices {
			fmt.Printf("%s\t%s\n", price.CC, comparedPriceText(price))
		}
		return
	}
	fmt.Println(ui.Title.Render(fmt.Sprintf("%s (%d)", data.Name, data.AppID)))
	fmt.Println()
	rows := make([][]string, 0, len(data.Prices))
	for _, price := range data.Prices {
		rows = append(rows, []string{
			price.CC,
			yesNo(price.Available),
			comparedPriceText(price),
			comparedDiscountText(price),
			empty(price.DiscountEndText),
		})
	}
	fmt.Println(ui.Table([]string{"CC", i18n.T("table.available"), i18n.T("table.price"), i18n.T("table.discount"), i18n.T("table.discount_ends")}, rows))
	fmt.Println(ui.Muted.Render(fmt.Sprintf(i18n.T("price.observed_from"), data.ObservedAt, data.Source)))
}

func comparedPriceText(price comparedPrice) string {
	if price.Error != "" {
		return price.Error
	}
	if price.PriceOverview == nil {
		return "-"
	}
	if price.PriceOverview.FinalFormatted != "" {
		return price.PriceOverview.FinalFormatted
	}
	return ui.Money(price.PriceOverview.Final, price.PriceOverview.Currency)
}

func comparedDiscountText(price comparedPrice) string {
	if price.PriceOverview == nil || price.PriceOverview.DiscountPercent <= 0 {
		return "-"
	}
	return fmt.Sprintf("-%d%%", price.PriceOverview.DiscountPercent)
}

func splitCodes(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		code := strings.ToUpper(strings.TrimSpace(part))
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		out = append(out, code)
	}
	return out
}

func discountEndText(item *steam.StoreItem) string {
	if item == nil || item.BestPurchaseOption == nil {
		return ""
	}
	latest := latestDiscountEnd(item.BestPurchaseOption.ActiveDiscounts)
	if latest == 0 {
		return ""
	}
	return formatDiscountEnd(latest)
}

func latestDiscountEndFromStoreItem(item *steam.StoreItem) int64 {
	if item == nil || item.BestPurchaseOption == nil {
		return 0
	}
	return latestDiscountEnd(item.BestPurchaseOption.ActiveDiscounts)
}

func purchaseOptionDiscountEndText(option steam.PurchaseOption) string {
	latest := latestDiscountEnd(option.ActiveDiscounts)
	if latest == 0 {
		return "-"
	}
	return time.Unix(latest, 0).Format("2006-01-02 15:04")
}

func discountPctText(value int) string {
	if value <= 0 {
		return "-"
	}
	return fmt.Sprintf("-%d%%", value)
}

func latestDiscountEnd(discounts []steam.ActiveDiscount) int64 {
	var latest int64
	for _, discount := range discounts {
		if discount.DiscountEndDate > latest {
			latest = discount.DiscountEndDate
		}
	}
	return latest
}

func formatDiscountEnd(latest int64) string {
	local := time.Unix(latest, 0).Format("2006-01-02 15:04 MST")
	utc := time.Unix(latest, 0).UTC().Format("2006-01-02 15:04 UTC")
	ptLocation, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return fmt.Sprintf("%s (%s)", local, utc)
	}
	pt := time.Unix(latest, 0).In(ptLocation).Format("2006-01-02 15:04 MST")
	return fmt.Sprintf("%s (%s, %s)", local, utc, pt)
}
