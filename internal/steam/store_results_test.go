package steam

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseStoreResultsHTML(t *testing.T) {
	html := `
	<a href="https://store.steampowered.com/app/264710/Subnautica/" data-ds-appid="264710" class="search_result_row">
		<span class="title">Subnautica</span>
		<div class="search_released">Jan 23, 2018</div>
		<span class="search_review_summary positive" data-tooltip-html="Overwhelmingly Positive&lt;br&gt;"> </span>
		<div class="discount_pct">-75%</div>
		<div class="discount_original_price">$29.99</div>
		<div class="discount_final_price">$7.49</div>
	</a>`
	got := ParseStoreResultsHTML(html)
	if len(got) != 1 {
		t.Fatalf("len(ParseStoreResultsHTML()) = %d, want 1", len(got))
	}
	item := got[0]
	if item.AppID != 264710 || item.Name != "Subnautica" || item.Release != "Jan 23, 2018" || item.Discount != "-75%" || item.Final != "$7.49" {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestStoreResultJSONOmitsLocalizedReleaseText(t *testing.T) {
	raw, err := json.Marshal(StoreResult{
		AppID:       1,
		Name:        "Game",
		Release:     "2026 年 5 月 27 日",
		ReleaseTime: 1779890400,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, `"release":`) || strings.Contains(text, `"release_date"`) || strings.Contains(text, `"precision"`) || !strings.Contains(text, `"release_time":1779890400`) {
		t.Fatalf("json = %s", text)
	}
}
