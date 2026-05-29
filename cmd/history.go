package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"steam-cli/internal/i18n"
	"steam-cli/internal/itad"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var historyOpts struct {
	days      int
	allStores bool
	sales     bool
	shops     string
}

type historyResult struct {
	AppID   int            `json:"appid"`
	Name    string         `json:"name"`
	Since   string         `json:"since,omitempty"`
	Scope   string         `json:"scope"`
	Entries []historyEntry `json:"entries"`
	Sales   []saleWindow   `json:"sales,omitempty"`
}

type historyEntry struct {
	Date     string `json:"date"`
	At       string `json:"at,omitempty"`
	Unix     int64  `json:"unix,omitempty"`
	Store    string `json:"store"`
	Price    string `json:"price"`
	Original string `json:"original"`
	Discount string `json:"discount"`
}

type saleWindow struct {
	Start        string `json:"start"`
	StartAt      string `json:"start_at"`
	StartUnix    int64  `json:"start_unix"`
	End          string `json:"end,omitempty"`
	EndAt        string `json:"end_at,omitempty"`
	EndUnix      int64  `json:"end_unix,omitempty"`
	Store        string `json:"store"`
	Price        string `json:"price"`
	Original     string `json:"original"`
	Discount     string `json:"discount"`
	Status       string `json:"status"`
	DurationDays int    `json:"duration_days,omitempty"`
}

var historyCmd = &cobra.Command{
	Use:     "history APPID",
	Short:   "Show Steam price history for an app, with optional all-store expansion",
	Example: "  steam-cli history 264710\n  steam-cli history 264710 --days 30\n  steam-cli history 264710 --all-stores",
	Args:    exactArgsWithExample(1, "steam-cli history APPID [--days N] [--all-stores]", "steam-cli history 264710 --days 30"),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		appid, err := parseAppID(args[0])
		if err != nil {
			return nil, err
		}
		var shops []int
		scope := "steam"
		if strings.TrimSpace(historyOpts.shops) != "" {
			shops, err = parseShopIDs(historyOpts.shops)
			if err != nil {
				return nil, err
			}
			scope = "custom"
		} else if !historyOpts.allStores {
			shops = []int{itad.SteamShopID()}
		} else {
			scope = "all"
		}
		client := itadClient()
		game, err := client.LookupByAppID(appid)
		if err != nil {
			return nil, err
		}
		since := ""
		if historyOpts.days > 0 {
			since = time.Now().Add(-time.Duration(historyOpts.days) * 24 * time.Hour).UTC().Format(time.RFC3339)
		}
		entries, err := client.History(game.ID, strings.ToUpper(opts.cc), shops, since)
		if err != nil {
			return nil, err
		}
		out := make([]historyEntry, 0, len(entries))
		for _, entry := range entries {
			ts, err := time.Parse(time.RFC3339, entry.Timestamp)
			if err != nil {
				return nil, err
			}
			discount := "-"
			if entry.Deal.Cut > 0 {
				discount = fmt.Sprintf("-%d%%", entry.Deal.Cut)
			}
			out = append(out, historyEntry{
				Date:     formatITADDate(entry.Timestamp),
				At:       ts.Format(time.RFC3339),
				Unix:     ts.Unix(),
				Store:    entry.Shop.Name,
				Price:    formatITADMoney(&entry.Deal.Price),
				Original: formatITADMoney(&entry.Deal.Regular),
				Discount: discount,
			})
		}
		return historyResult{
			AppID:   appid,
			Name:    game.Title,
			Since:   since,
			Scope:   scope,
			Entries: out,
			Sales:   buildSaleWindows(entries),
		}, nil
	}, func(value any) error {
		data := value.(historyResult)
		if historyOpts.sales {
			return renderSaleHistory(data)
		}
		if opts.quiet {
			for _, entry := range data.Entries {
				fmt.Printf("%s\t%s\t%s\t%s\n", entry.Date, entry.Store, entry.Price, entry.Discount)
			}
			return nil
		}
		fmt.Println(ui.Title.Render(fmt.Sprintf("%s (%d)", data.Name, data.AppID)))
		fmt.Println()
		rows := make([][]string, 0, len(data.Entries))
		for _, entry := range data.Entries {
			rows = append(rows, []string{
				entry.Date,
				entry.Store,
				entry.Price,
				entry.Original,
				entry.Discount,
			})
		}
		if len(rows) == 0 {
			fmt.Println(ui.Muted.Render(i18n.T("message.no_results")))
			return nil
		}
		fmt.Println(ui.Table([]string{i18n.T("table.date"), i18n.T("table.shop"), i18n.T("table.price"), i18n.T("table.original"), i18n.T("table.discount")}, rows))
		if data.Since != "" {
			fmt.Println(ui.Muted.Render(fmt.Sprintf(i18n.T("history.observed_since"), formatITADDate(data.Since))))
		}
		return nil
	}),
}

func init() {
	historyCmd.Flags().IntVar(&historyOpts.days, "days", 90, "load price changes from the last N days; 0 keeps the API default")
	historyCmd.Flags().BoolVar(&historyOpts.allStores, "all-stores", false, "include authorized non-Steam store history in addition to Steam store history")
	historyCmd.Flags().BoolVar(&historyOpts.sales, "sales", false, "show historical discount windows instead of raw price-change points")
	historyCmd.Flags().StringVar(&historyOpts.shops, "shops", "", "comma-separated ITAD shop IDs, for example 61,35")
	_ = historyCmd.Flags().MarkHidden("shops")
}

func renderSaleHistory(data historyResult) error {
	if opts.quiet {
		for _, sale := range data.Sales {
			fmt.Printf("%s\t%s\t%s\t%s\t%s\n", sale.Start, empty(sale.End), sale.Store, sale.Price, sale.Discount)
		}
		return nil
	}
	fmt.Println(ui.Title.Render(fmt.Sprintf("%s (%d)", data.Name, data.AppID)))
	fmt.Println()
	if len(data.Sales) == 0 {
		fmt.Println(ui.Muted.Render(i18n.T("message.no_results")))
		return nil
	}
	rows := make([][]string, 0, len(data.Sales))
	for _, sale := range data.Sales {
		duration := "-"
		if sale.DurationDays > 0 {
			duration = fmt.Sprintf("%dd", sale.DurationDays)
		}
		rows = append(rows, []string{
			sale.Start,
			empty(sale.End),
			sale.Store,
			sale.Price,
			sale.Original,
			sale.Discount,
			sale.Status,
			duration,
		})
	}
	fmt.Println(ui.Table([]string{
		i18n.T("table.start"),
		i18n.T("table.end"),
		i18n.T("table.shop"),
		i18n.T("table.price"),
		i18n.T("table.original"),
		i18n.T("table.discount"),
		i18n.T("table.status"),
		i18n.T("table.duration"),
	}, rows))
	if data.Since != "" {
		fmt.Println(ui.Muted.Render(fmt.Sprintf(i18n.T("history.observed_since"), formatITADDate(data.Since))))
	}
	return nil
}

func buildSaleWindows(entries []itad.HistoryEntry) []saleWindow {
	if len(entries) == 0 {
		return nil
	}
	type parsedEntry struct {
		itad.HistoryEntry
		ts time.Time
	}
	byStore := map[string][]parsedEntry{}
	for _, entry := range entries {
		ts, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil {
			continue
		}
		byStore[entry.Shop.Name] = append(byStore[entry.Shop.Name], parsedEntry{
			HistoryEntry: entry,
			ts:           ts,
		})
	}
	out := []saleWindow{}
	for store, items := range byStore {
		sort.Slice(items, func(i, j int) bool {
			return items[i].ts.After(items[j].ts)
		})
		for i, item := range items {
			if item.Deal.Cut <= 0 {
				continue
			}
			sale := saleWindow{
				Start:     item.ts.Format(time.DateOnly),
				StartAt:   item.ts.Format(time.RFC3339),
				StartUnix: item.ts.Unix(),
				Store:     store,
				Price:     formatITADMoney(&item.Deal.Price),
				Original:  formatITADMoney(&item.Deal.Regular),
				Discount:  fmt.Sprintf("-%d%%", item.Deal.Cut),
				Status:    i18n.T("history.status_finished"),
			}
			if i == 0 {
				sale.Status = i18n.T("history.status_active")
			} else {
				end := items[i-1].ts
				sale.End = end.Format(time.DateOnly)
				sale.EndAt = end.Format(time.RFC3339)
				sale.EndUnix = end.Unix()
				if end.After(item.ts) {
					sale.DurationDays = int(end.Sub(item.ts).Hours() / 24)
					if sale.DurationDays == 0 {
						sale.DurationDays = 1
					}
				}
			}
			out = append(out, sale)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Start == out[j].Start {
			return out[i].Store < out[j].Store
		}
		return out[i].Start > out[j].Start
	})
	return out
}
