package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"steam-cli/internal/itad"
	"steam-cli/internal/steam"
)

func itadClient() *itad.Client {
	return itad.NewClient(opts.itadKey, time.Duration(opts.timeout)*time.Second)
}

func loadITADSummary(appid int) (*itad.Summary, error) {
	client := itadClient()
	return client.SummaryByAppID(appid, strings.ToUpper(opts.cc))
}

type priceInsights struct {
	BestDeal      *insightDeal         `json:"best_deal,omitempty"`
	HistoryLow    *historyLowWindows   `json:"history_low,omitempty"`
	BestEverDeal  *insightHistoricDeal `json:"best_ever_deal,omitempty"`
	SteamStoreLow *insightHistoricDeal `json:"steam_store_low,omitempty"`
	BundleCount   int                  `json:"bundle_count"`
	PricePage     string               `json:"price_page,omitempty"`
	DealURL       string               `json:"deal_url,omitempty"`
	Bundles       []insightBundle      `json:"bundles,omitempty"`
}

type insightDeal struct {
	Store    string `json:"store"`
	Price    string `json:"price"`
	Original string `json:"original,omitempty"`
	Discount string `json:"discount,omitempty"`
	Date     string `json:"date,omitempty"`
	URL      string `json:"url,omitempty"`
}

type insightHistoricDeal struct {
	Store    string `json:"store"`
	Price    string `json:"price"`
	Original string `json:"original,omitempty"`
	Discount string `json:"discount,omitempty"`
	Date     string `json:"date,omitempty"`
}

type historyLowWindows struct {
	All string `json:"all,omitempty"`
	Y1  string `json:"y1,omitempty"`
	M3  string `json:"m3,omitempty"`
}

type insightBundle struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Price  string `json:"price,omitempty"`
	Expiry string `json:"expiry,omitempty"`
}

func buildPriceInsights(summary *itad.Summary) *priceInsights {
	if summary == nil {
		return nil
	}
	out := &priceInsights{
		BestDeal:      toInsightDeal(itadCurrentDeal(summary)),
		HistoryLow:    toHistoryLowWindows(summary),
		BestEverDeal:  toInsightHistoricDeal(itadLowestDeal(summary)),
		SteamStoreLow: toInsightHistoricDeal(itadSteamLowDeal(summary)),
		BundleCount:   len(activeITADBundles(summary.Bundles)),
		PricePage:     itadGameURL(summary),
		DealURL:       itadCurrentDealURL(summary),
		Bundles:       toInsightBundles(activeITADBundles(summary.Bundles)),
	}
	if out.BestDeal == nil && out.HistoryLow == nil && out.BestEverDeal == nil && out.SteamStoreLow == nil && out.BundleCount == 0 && out.PricePage == "" && out.DealURL == "" && len(out.Bundles) == 0 {
		return nil
	}
	return out
}

func toInsightDeal(value *itad.Deal) *insightDeal {
	if value == nil {
		return nil
	}
	out := &insightDeal{
		Store: value.Shop.Name,
		Price: formatITADMoney(&value.Price),
		Date:  formatITADDate(value.Timestamp),
		URL:   value.URL,
	}
	if value.Regular.AmountInt > 0 {
		out.Original = formatITADMoney(&value.Regular)
	}
	if value.Cut > 0 {
		out.Discount = fmt.Sprintf("-%d%%", value.Cut)
	}
	return out
}

func toInsightHistoricDeal(value *itad.HistoricDeal) *insightHistoricDeal {
	if value == nil {
		return nil
	}
	out := &insightHistoricDeal{
		Store: value.Shop.Name,
		Price: formatITADMoney(&value.Price),
		Date:  formatITADDate(value.Timestamp),
	}
	if value.Regular.AmountInt > 0 {
		out.Original = formatITADMoney(&value.Regular)
	}
	if value.Cut > 0 {
		out.Discount = fmt.Sprintf("-%d%%", value.Cut)
	}
	return out
}

func toHistoryLowWindows(summary *itad.Summary) *historyLowWindows {
	if summary == nil || summary.HistoryLow == nil {
		return nil
	}
	out := &historyLowWindows{}
	if value := summary.HistoryLow.HistoryLow.All; value != nil {
		out.All = formatITADMoney(value)
	}
	if value := summary.HistoryLow.HistoryLow.Y1; value != nil {
		out.Y1 = formatITADMoney(value)
	}
	if value := summary.HistoryLow.HistoryLow.M3; value != nil {
		out.M3 = formatITADMoney(value)
	}
	if out.All == "" && out.Y1 == "" && out.M3 == "" {
		return nil
	}
	return out
}

func toInsightBundles(bundles []itad.Bundle) []insightBundle {
	if len(bundles) == 0 {
		return nil
	}
	out := make([]insightBundle, 0, len(bundles))
	for _, bundle := range bundles {
		item := insightBundle{
			ID:     bundle.ID,
			Title:  bundle.Title,
			Expiry: formatITADDate(ptrString(bundle.Expiry)),
		}
		if len(bundle.Tiers) > 0 {
			item.Price = formatITADMoney(&bundle.Tiers[0].Price)
		}
		out = append(out, item)
	}
	return out
}

func formatInsightDeal(value *insightDeal) string {
	if value == nil {
		return "-"
	}
	parts := []string{value.Price, value.Store}
	if value.Discount != "" {
		parts = append(parts, value.Discount)
	}
	if value.Date != "" {
		parts = append(parts, value.Date)
	}
	return strings.Join(parts, "  ")
}

func formatInsightHistoricDeal(value *insightHistoricDeal) string {
	if value == nil {
		return "-"
	}
	parts := []string{value.Price, value.Store}
	if value.Discount != "" {
		parts = append(parts, value.Discount)
	}
	if value.Date != "" {
		parts = append(parts, value.Date)
	}
	return strings.Join(parts, "  ")
}

func historyLowText(value *historyLowWindows) string {
	if value == nil {
		return "-"
	}
	parts := []string{}
	if value.All != "" {
		parts = append(parts, "all "+value.All)
	}
	if value.Y1 != "" {
		parts = append(parts, "1y "+value.Y1)
	}
	if value.M3 != "" {
		parts = append(parts, "3m "+value.M3)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "  ")
}

func bundleCountText(value *priceInsights) string {
	if value == nil {
		return "-"
	}
	return strconv.Itoa(value.BundleCount)
}

func insightBundleRows(bundles []insightBundle) [][]string {
	rows := make([][]string, 0, len(bundles))
	for _, bundle := range bundles {
		rows = append(rows, []string{
			strconv.Itoa(bundle.ID),
			empty(bundle.Title),
			empty(bundle.Price),
			empty(bundle.Expiry),
		})
	}
	return rows
}

func formatITADMoney(value *itad.Money) string {
	if value == nil {
		return "-"
	}
	if symbol, ok := currencySymbol(strings.ToUpper(value.Currency)); ok {
		return fmt.Sprintf("%s%.2f", symbol, value.Amount)
	}
	return fmt.Sprintf("%.2f %s", value.Amount, value.Currency)
}

func currencySymbol(currency string) (string, bool) {
	symbols := map[string]string{
		"AUD": "A$",
		"BRL": "R$",
		"CAD": "C$",
		"CHF": "CHF ",
		"CLP": "CLP$",
		"CNY": "¥",
		"EUR": "€",
		"GBP": "£",
		"HKD": "HK$",
		"JPY": "¥",
		"KRW": "₩",
		"MXN": "Mex$",
		"NOK": "kr ",
		"NZD": "NZ$",
		"PLN": "zł ",
		"RUB": "₽",
		"SGD": "S$",
		"TWD": "NT$",
		"USD": "$",
	}
	symbol, ok := symbols[currency]
	return symbol, ok
}

func formatITADDeal(value *itad.Deal) string {
	if value == nil {
		return "-"
	}
	parts := []string{
		formatITADMoney(&value.Price),
		value.Shop.Name,
	}
	if value.Cut > 0 {
		parts = append(parts, fmt.Sprintf("-%d%%", value.Cut))
	}
	if ts := formatITADDate(value.Timestamp); ts != "" {
		parts = append(parts, ts)
	}
	return strings.Join(parts, "  ")
}

func formatITADHistoricDeal(value *itad.HistoricDeal) string {
	if value == nil {
		return "-"
	}
	parts := []string{
		formatITADMoney(&value.Price),
		value.Shop.Name,
	}
	if value.Cut > 0 {
		parts = append(parts, fmt.Sprintf("-%d%%", value.Cut))
	}
	if ts := formatITADDate(value.Timestamp); ts != "" {
		parts = append(parts, ts)
	}
	return strings.Join(parts, "  ")
}

func formatITADDate(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return parsed.Format(time.DateOnly)
}

func itadHistoryLowText(summary *itad.Summary) string {
	if summary == nil || summary.HistoryLow == nil {
		return "-"
	}
	parts := []string{}
	if value := summary.HistoryLow.HistoryLow.All; value != nil {
		parts = append(parts, "all "+formatITADMoney(value))
	}
	if value := summary.HistoryLow.HistoryLow.Y1; value != nil {
		parts = append(parts, "1y "+formatITADMoney(value))
	}
	if value := summary.HistoryLow.HistoryLow.M3; value != nil {
		parts = append(parts, "3m "+formatITADMoney(value))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "  ")
}

func itadBundlesText(summary *itad.Summary) string {
	if summary == nil {
		return "-"
	}
	return strconv.Itoa(len(activeITADBundles(summary.Bundles)))
}

func itadCurrentDeal(summary *itad.Summary) *itad.Deal {
	if summary == nil || summary.Overview == nil {
		return nil
	}
	return summary.Overview.Current
}

func itadLowestDeal(summary *itad.Summary) *itad.HistoricDeal {
	if summary == nil || summary.Overview == nil {
		return nil
	}
	return summary.Overview.Lowest
}

func itadSteamLowDeal(summary *itad.Summary) *itad.HistoricDeal {
	if summary == nil || summary.SteamLow == nil {
		return nil
	}
	return summary.SteamLow.Low
}

func itadGameURL(summary *itad.Summary) string {
	if summary == nil || summary.Overview == nil {
		return ""
	}
	return summary.Overview.URLs.Game
}

func itadCurrentDealURL(summary *itad.Summary) string {
	deal := itadCurrentDeal(summary)
	if deal == nil {
		return ""
	}
	return deal.URL
}

func itadBundleRows(bundles []itad.Bundle) [][]string {
	active := activeITADBundles(bundles)
	rows := make([][]string, 0, len(active))
	for _, bundle := range active {
		price := "-"
		if len(bundle.Tiers) > 0 {
			price = formatITADMoney(&bundle.Tiers[0].Price)
		}
		expires := empty(formatITADDate(ptrString(bundle.Expiry)))
		rows = append(rows, []string{
			strconv.Itoa(bundle.ID),
			empty(bundle.Title),
			price,
			expires,
		})
	}
	return rows
}

func activeITADBundles(bundles []itad.Bundle) []itad.Bundle {
	if len(bundles) == 0 {
		return nil
	}
	now := time.Now()
	out := make([]itad.Bundle, 0, len(bundles))
	for _, bundle := range bundles {
		if bundle.Expiry == nil || *bundle.Expiry == "" {
			out = append(out, bundle)
			continue
		}
		expiry, err := time.Parse(time.RFC3339, *bundle.Expiry)
		if err != nil || expiry.After(now) {
			out = append(out, bundle)
		}
	}
	return out
}

func ptrString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func parseShopIDs(raw string) ([]int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	seen := map[int]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil || id <= 0 {
			return nil, &steam.Error{
				Code:    steam.CodeInvalidInput,
				Message: fmt.Sprintf("invalid --shops value %q; expected comma-separated numeric shop IDs", raw),
			}
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, nil
}
