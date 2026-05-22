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
