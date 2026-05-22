package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"
)

func storePrice(option *steam.PurchaseOption) string {
	if option == nil {
		return "-"
	}
	if option.FormattedFinalPrice != "" {
		if option.DiscountPct > 0 && option.FormattedOriginalPrice != "" {
			return fmt.Sprintf("%s %s", option.FormattedFinalPrice, ui.Warn.Render("-"+strconv.Itoa(option.DiscountPct)+"%"))
		}
		return option.FormattedFinalPrice
	}
	if option.FinalPriceInCents == 0 && option.OriginalPriceInCents == 0 {
		return "-"
	}
	return fmt.Sprintf("%.2f", float64(option.FinalPriceInCents)/100)
}

func storeReviewText(reviews *steam.StoreReviews) string {
	if reviews == nil || reviews.SummaryFiltered == nil {
		return "-"
	}
	summary := reviews.SummaryFiltered
	if summary.ReviewScoreLabel == "" {
		return fmt.Sprintf("%d%% positive (%d)", summary.PercentPositive, summary.ReviewCount)
	}
	return fmt.Sprintf("%s, %d%% of %d", summary.ReviewScoreLabel, summary.PercentPositive, summary.ReviewCount)
}

func storePlatformsText(platforms *steam.StorePlatforms) string {
	if platforms == nil {
		return "-"
	}
	values := []string{}
	if platforms.Windows {
		values = append(values, "windows")
	}
	if platforms.Mac {
		values = append(values, "mac")
	}
	if platforms.Linux {
		values = append(values, "linux")
	}
	if platforms.VRSupport != nil && platforms.VRSupport.VRHMD {
		values = append(values, "vr")
	}
	return ui.Join(values)
}

func storeTagsText(tags []steam.StoreTag, limit int) string {
	if len(tags) == 0 {
		return "-"
	}
	if limit <= 0 || limit > len(tags) {
		limit = len(tags)
	}
	parts := make([]string, 0, limit)
	for _, tag := range tags[:limit] {
		parts = append(parts, fmt.Sprintf("%d(%d)", tag.TagID, tag.Weight))
	}
	return strings.Join(parts, ", ")
}

func storeLinkRows(links []steam.StoreLink) [][]string {
	rows := make([][]string, 0, len(links))
	for _, link := range links {
		rows = append(rows, []string{storeLinkType(link.LinkType), link.URL})
	}
	return rows
}

func storeLinkType(value int) string {
	names := map[int]string{
		1:  "YouTube",
		2:  "Facebook",
		3:  "X",
		4:  "Twitch",
		5:  "Discord",
		10: "Reddit",
		11: "Instagram",
		14: "TikTok",
		20: "Bluesky",
		22: "Threads",
	}
	if name, ok := names[value]; ok {
		return name
	}
	return strconv.Itoa(value)
}

func storeLanguageSummary(languages []steam.StoreLanguage) string {
	if len(languages) == 0 {
		return "-"
	}
	fullAudio := 0
	subtitles := 0
	for _, language := range languages {
		if language.FullAudio {
			fullAudio++
		}
		if language.Subtitles {
			subtitles++
		}
	}
	return fmt.Sprintf("%d supported, %d full audio, %d subtitles", len(languages), fullAudio, subtitles)
}

func storeDate(unix int64) string {
	if unix <= 0 {
		return "-"
	}
	return time.Unix(unix, 0).Format(time.DateOnly)
}

func purchaseOptionID(option steam.PurchaseOption) string {
	if option.PackageID > 0 {
		return "sub " + strconv.Itoa(option.PackageID)
	}
	if option.BundleID > 0 {
		return "bundle " + strconv.Itoa(option.BundleID)
	}
	return "-"
}
