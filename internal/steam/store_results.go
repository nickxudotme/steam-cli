package steam

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

func ParseStoreResultsHTML(raw string) []StoreResult {
	raw = html.UnescapeString(raw)
	rowRe := regexp.MustCompile(`(?is)<a\s+href="([^"]+)"[^>]*data-ds-appid="(\d+)"[^>]*class="search_result_row.*?</a>`)
	rows := rowRe.FindAllStringSubmatch(raw, -1)

	results := make([]StoreResult, 0, len(rows))
	for _, row := range rows {
		block := row[0]
		appid, _ := strconv.Atoi(row[2])
		release := firstMatch(block, `(?is)<div class="search_released[^"]*">\s*(.*?)\s*</div>`)
		results = append(results, StoreResult{
			AppID:    appid,
			URL:      row[1],
			Name:     firstMatch(block, `(?is)<span class="title">(.*?)</span>`),
			Release:  release,
			Review:   cleanTooltip(firstMatch(block, `(?is)data-tooltip-html="(.*?)"`)),
			Discount: firstMatch(block, `(?is)<div class="discount_pct">(.*?)</div>`),
			Original: firstMatch(block, `(?is)<div class="discount_original_price">(.*?)</div>`),
			Final:    firstMatch(block, `(?is)<div class="discount_final_price">(.*?)</div>`),
		})
	}
	return results
}

func firstMatch(value, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(value)
	if len(match) < 2 {
		return "-"
	}
	return cleanText(match[1])
}

func cleanTooltip(value string) string {
	value = strings.ReplaceAll(value, "&lt;br&gt;", " | ")
	value = strings.ReplaceAll(value, "<br>", " | ")
	parts := strings.Split(cleanText(value), "|")
	if len(parts) == 0 {
		return "-"
	}
	return strings.TrimSpace(parts[0])
}
