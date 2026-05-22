package cmd

import (
	"strconv"
	"strings"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"
)

// parseAppID converts a CLI argument to an integer appid, returning a
// typed *steam.Error so the JSON envelope's error.type is "invalid_input".
//
// Use this from every command that takes APPID. Do NOT replicate
// fmt.Errorf("invalid appid %q") at call sites — error classification
// depends on the typed Error.
func parseAppID(arg string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(arg))
	if err != nil || id <= 0 {
		// Mirrors the previous message so any external scripts grepping
		// stderr keep working, but classification now goes through the
		// typed Error.Code field.
		return 0, &steam.Error{
			Code:    steam.CodeInvalidInput,
			Message: "invalid appid \"" + arg + "\"",
			HintKey: "hint.invalid_appid",
		}
	}
	return id, nil
}

func namedValues(values []steam.NamedValue) string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		if value.Description != "" {
			names = append(names, value.Description)
		}
	}
	return ui.Join(names)
}

func intPtrText(value *int) string {
	if value == nil {
		return "-"
	}
	return strconv.Itoa(*value)
}

func priceText(details *steam.AppDetails) string {
	if details.IsFree {
		return ui.Price(0, 0, 0, "", true)
	}
	if details.PriceOverview == nil {
		return ui.Muted.Render("unavailable in this region or not sold separately")
	}
	price := details.PriceOverview
	return ui.Price(price.Final, price.Initial, price.DiscountPercent, price.Currency, false)
}

func priceDetail(details *steam.AppDetails) string {
	if details.IsFree || details.PriceOverview == nil {
		return priceText(details)
	}
	price := details.PriceOverview
	parts := []string{ui.Price(price.Final, price.Initial, price.DiscountPercent, price.Currency, false)}
	if price.Initial > 0 && price.Initial != price.Final {
		parts = append(parts, "original "+ui.Money(price.Initial, price.Currency))
	}
	return strings.Join(parts, "  ")
}

func truncate(value string, limit int) string {
	value = strings.Join(strings.Fields(stripTags(value)), " ")
	if limit <= 0 || len([]rune(value)) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit-1]) + "..."
}

func stripTags(value string) string {
	var b strings.Builder
	inTag := false
	for _, r := range value {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
