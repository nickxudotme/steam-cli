package steam

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestEventsUsesClientLanguage(t *testing.T) {
	c := NewClient("US", "schinese", time.Second)
	c.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.Query().Get("l"); got != "schinese" {
			t.Fatalf("language query = %q, want schinese", got)
		}
		body := `<div class="documentation_bbcode">
			<h2><a name="1"></a>季节性特卖</h2>
			<h2><a name="2"></a>2026 年季节性特卖</h2>
			<h2><a name="3"></a>夏日特卖 | 2026 年 6 月 25 日 - 7 月 9 日（PT）</h2>
			<table><tr><th>活动日期</th><th>主题</th></tr></table>
			<h2><a name="9"></a>新品节</h2>
		</div><div id="hashLocationHighlight"></div>`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	events, err := c.Events(EventQuery{PastDays: 365, FutureDays: 365})
	if err != nil {
		t.Fatalf("Events returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Events returned %d events, want 1: %#v", len(events), events)
	}
	if events[0].Name != "夏日特卖" {
		t.Fatalf("event name = %q, want localized name", events[0].Name)
	}
}

func TestEventsFallsBackToEnglishWhenLocalizedPageHasNoEvents(t *testing.T) {
	c := NewClient("US", "german", time.Second)
	queries := []string{}
	c.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		lang := req.URL.Query().Get("l")
		queries = append(queries, lang)
		body := `<div class="documentation_bbcode">
			<p>No localized documentation for this language.</p>
		</div><div id="hashLocationHighlight"></div>`
		if lang == "english" {
			body = `<div class="documentation_bbcode">
				<h2><a name="1"></a>Seasonal Sales</h2>
				<h2><a name="2"></a>2026 Seasonal Sales</h2>
				<h2><a name="3"></a>Summer Sale | June 25 - July 9, 2026</h2>
				<table><tr><th>Event Dates</th><th>Theme</th></tr></table>
				<h2><a name="9"></a>Next Fest</h2>
			</div><div id="hashLocationHighlight"></div>`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	events, err := c.Events(EventQuery{PastDays: 365, FutureDays: 365})
	if err != nil {
		t.Fatalf("Events returned error: %v", err)
	}
	if got, want := strings.Join(queries, ","), "german,english"; got != want {
		t.Fatalf("queries = %q, want %q", got, want)
	}
	if len(events) != 1 || events[0].Name != "Summer Sale" {
		t.Fatalf("fallback events = %#v, want English Summer Sale", events)
	}
}

func TestParseSteamworksEventsLocalizedChinese(t *testing.T) {
	raw := `<div class="documentation_bbcode">
		<h2 class="bb_section"><a name="1"></a>季节性特卖</h2>
		<h2 class="bb_subsection"><a name="2"></a>2026 年季节性特卖</h2>
		<h2 class="bb_subsection"><a name="3"></a>春季特卖 | 2026 年 3 月 19 日 - 26 日（已结束）</h2>
		<h2 class="bb_subsection"><a name="4"></a>夏日特卖 | 2026 年 6 月 25 日 - 7 月 9 日（PT）</h2>
		<h2 class="bb_section"><a name="7"></a>主题特卖活动（游戏节）</h2>
		<h2 class="bb_subsection"><a name="8"></a><strong>2026 年各游戏节</strong></h2>
		<table>
			<tr><th>活动日期</th><th>主题</th><th>注册</th><th>参加资格及备注</th></tr>
			<tr><td>4 月 9 日<br>4 月 13 日（PT）</td><td>隐藏物品游戏节</td><td>-</td><td>此游戏节已结束。</td></tr>
			<tr><td>5 月 18 日<br>5 月 25 日（PT）</td><td>海洋游戏节</td><td><a href="https://partner.steamgames.com/optin/sale/sale_ocean_2026">注册详情</a></td><td>关于大海的游戏，无论是在海面之上，还是深潜海底。 <a href="https://partner.steamgames.com/doc/marketing/upcoming_events/themed_sales/ocean_2026" class="bb_doclink">更多信息</a>。</td></tr>
		</table>
		<h2 class="bb_section"><a name="9"></a>新品节</h2>
		<h2 class="bb_subsection"><a name="12"></a><strong>新品节</strong> | 2026 年 6 月 15 日 - 6 月 22 日（PT）</h2>
	</div><div id="hashLocationHighlight"></div>`

	events := ParseSteamworksEvents(raw)
	if len(events) != 5 {
		t.Fatalf("len(ParseSteamworksEvents) = %d, want 5: %#v", len(events), events)
	}

	assertEvent := func(index int, name, startDate, endDate, category string) {
		t.Helper()
		event := events[index]
		if event.Name != name || event.StartDate != startDate || event.EndDate != endDate || event.Category != category {
			t.Fatalf("events[%d] = %#v, want name=%q start=%q end=%q category=%q", index, event, name, startDate, endDate, category)
		}
	}
	assertEvent(0, "春季特卖", "2026-03-19", "2026-03-26", "seasonal")
	assertEvent(1, "夏日特卖", "2026-06-25", "2026-07-09", "seasonal")
	assertEvent(2, "隐藏物品游戏节", "2026-04-09", "2026-04-13", "fest")
	assertEvent(3, "海洋游戏节", "2026-05-18", "2026-05-25", "fest")
	assertEvent(4, "新品节", "2026-06-15", "2026-06-22", "next_fest")

	if events[2].Description != "此游戏节已结束。" {
		t.Fatalf("ended fest description = %q", events[2].Description)
	}
	if events[3].Description != "关于大海的游戏，无论是在海面之上，还是深潜海底。" {
		t.Fatalf("localized description = %q", events[3].Description)
	}
}

func TestParseSteamworksEventsLocalizedEnglishFests(t *testing.T) {
	raw := `<div data-panel="doc" class="partner documentation_bbcode">
		<h2 class="bb_section"><a name="1"></a>Seasonal Sales</h2>
		<h2 class="bb_subsection"><a name="2"></a>2026 Seasonal Sales</h2>
		<h2 class="bb_subsection"><a name="3"></a>Summer Sale | June 25 - July 9, 2026</h2>
		<h2 class="bb_section"><a name="7"></a>Themed Sale Events (Fests)</h2>
		<h2 class="bb_subsection"><a name="8"></a><strong>2026 Fests</strong></h2>
		<table class="eventTable">
			<tr class="header"><th>Event Dates</th><th>Theme</th><th>Registration</th><th>Eligibility &amp; Notes</th></tr>
			<tr data-event="ocean"><td class="date">May 18<br>May 25</td><td>Ocean Fest</td><td><a href="https://partner.steamgames.com/optin/sale/sale_ocean_2026">Registration details</a></td><td>Games about the ocean, whether above water or below. <a href="https://partner.steamgames.com/doc/marketing/upcoming_events/themed_sales/ocean_2026" class="bb_doclink">More info</a>.</td></tr>
			<tr data-event="bullet"><td class="date">June 8<br />June 15</td><td>Bullet Fest</td><td><a href="https://partner.steamgames.com/optin/sale/sale_bullet_2026">Registration details</a></td><td>Games where the screen is a chaos of bullets. <a href="https://partner.steamgames.com/doc/marketing/upcoming_events/themed_sales/bullet_2026" class="bb_doclink">More info</a>.</td></tr>
		</table>
		<h2 class="bb_section"><a name="9"></a>Next Fest</h2>
		<h2 class="bb_subsection"><a name="12"></a><strong>Next Fest</strong> | June 15 - June 22, 2026</h2>
	</div><div class="marker" id="hashLocationHighlight"></div>`

	events := ParseSteamworksEvents(raw)
	if len(events) != 4 {
		t.Fatalf("len(ParseSteamworksEvents) = %d, want 4: %#v", len(events), events)
	}

	assertEvent := func(index int, name, startDate, endDate, category string) {
		t.Helper()
		event := events[index]
		if event.Name != name || event.StartDate != startDate || event.EndDate != endDate || event.Category != category {
			t.Fatalf("events[%d] = %#v, want name=%q start=%q end=%q category=%q", index, event, name, startDate, endDate, category)
		}
	}
	assertEvent(0, "Summer Sale", "2026-06-25", "2026-07-09", "seasonal")
	assertEvent(1, "Ocean Fest", "2026-05-18", "2026-05-25", "fest")
	assertEvent(2, "Bullet Fest", "2026-06-08", "2026-06-15", "fest")
	assertEvent(3, "Steam Next Fest", "2026-06-15", "2026-06-22", "next_fest")

	if events[1].Description != "Games about the ocean, whether above water or below." {
		t.Fatalf("english description = %q", events[1].Description)
	}
}

func TestParseSteamStoreSalePage(t *testing.T) {
	raw := `<html><head>
		<meta property="og:title" content="2026 年农历新年">
	</head><body>
		<div id="application_config"
			data-partnereventstore="[{&quot;event_name&quot;:&quot;Lunar New Year 2026&quot;,&quot;rtime32_start_time&quot;:1770920340,&quot;rtime32_end_time&quot;:1772128800}]"
			data-groupvanityinfo="[{&quot;group_name&quot;:&quot;蒸汽平台促销&quot;}]"></div>
	</body></html>`

	events := ParseSteamStoreSalePage(raw, "https://store.steampowered.com/sale/lny2026")
	if len(events) != 1 {
		t.Fatalf("len(ParseSteamStoreSalePage) = %d, want 1: %#v", len(events), events)
	}

	event := events[0]
	if event.Name != "2026 年农历新年" {
		t.Fatalf("name = %q, want localized title", event.Name)
	}
	if event.StartDate != "2026-02-12" || event.EndDate != "2026-02-26" {
		t.Fatalf("dates = %s - %s, want PT sale dates", event.StartDate, event.EndDate)
	}
	if event.Source != "steam_store" || event.Category != "store_sale" || event.Timezone != "PT" {
		t.Fatalf("source/category/timezone = %#v", event)
	}
	if event.Description != "Steam Store sale page presented by 蒸汽平台促销." {
		t.Fatalf("description = %q", event.Description)
	}
}

func TestParseSteamStoreSaleURLs(t *testing.T) {
	raw := `<div
		data-ch_spotlights_data="[{&quot;url&quot;:&quot;https:\/\/store.steampowered.com\/sale\/lny2026?snr=1&quot;},{&quot;url&quot;:&quot;https:\/\/store.steampowered.com\/app\/620&quot;}]">
		<a href="/sale/skulls2026">Skulls</a>
		<a href="https://store.steampowered.com/sale/lny2026?again=1">Duplicate</a>
	</div>`

	urls := ParseSteamStoreSaleURLs(raw, "http://example.test")
	want := []string{
		"http://example.test/sale/lny2026",
		"http://example.test/sale/skulls2026",
	}
	if strings.Join(urls, ",") != strings.Join(want, ",") {
		t.Fatalf("urls = %#v, want %#v", urls, want)
	}
}
