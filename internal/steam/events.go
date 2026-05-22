package steam

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const SteamworksUpcomingEvents = "https://partner.steamgames.com/doc/marketing/upcoming_events"

type Event struct {
	Name            string `json:"name"`
	StartDate       string `json:"start_date"`
	EndDate         string `json:"end_date"`
	Status          string `json:"status"`
	Source          string `json:"source"`
	Category        string `json:"category"`
	Timezone        string `json:"timezone"`
	Description     string `json:"description,omitempty"`
	Notes           string `json:"notes,omitempty"`
	RegistrationURL string `json:"registration_url,omitempty"`
	InfoURL         string `json:"info_url,omitempty"`
}

type EventQuery struct {
	PastDays   int
	FutureDays int
}

var months = map[string]time.Month{
	"Jan": time.January, "January": time.January,
	"Feb": time.February, "February": time.February,
	"Mar": time.March, "March": time.March,
	"Apr": time.April, "April": time.April,
	"May": time.May,
	"Jun": time.June, "June": time.June,
	"Jul": time.July, "July": time.July,
	"Aug": time.August, "August": time.August,
	"Sep": time.September, "Sept": time.September, "September": time.September,
	"Oct": time.October, "October": time.October,
	"Nov": time.November, "November": time.November,
	"Dec": time.December, "December": time.December,
}

func (c *Client) Events(query EventQuery) ([]Event, error) {
	body, err := c.GetText(SteamworksUpcomingEvents, url.Values{"l": {"english"}})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch live Steamworks upcoming events: %w", err)
	}
	parsed := ParseSteamworksEvents(body)
	if len(parsed) == 0 {
		return nil, fmt.Errorf("no live Steamworks events could be parsed from %s", SteamworksUpcomingEvents)
	}
	return FilterEvents(parsed, query), nil
}

func FilterEvents(events []Event, query EventQuery) []Event {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	minDay := today.AddDate(0, 0, -query.PastDays)
	maxDay := today.AddDate(0, 0, query.FutureDays)

	filtered := make([]Event, 0, len(events))
	for _, event := range events {
		start, err := parseISODate(event.StartDate)
		if err != nil {
			continue
		}
		end, err := parseISODate(event.EndDate)
		if err != nil {
			end = start
		}
		if end.Before(minDay) || start.After(maxDay) {
			continue
		}
		event.Status = eventStatus(today, start, end)
		filtered = append(filtered, event)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartDate < filtered[j].StartDate
	})
	return filtered
}

func ParseSteamworksEvents(raw string) []Event {
	raw = html.UnescapeString(raw)
	doc := documentationSection(raw)

	events := []Event{}
	events = append(events, parseHeadingEvents(doc, "seasonal")...)
	events = append(events, parseFestTable(doc)...)
	events = append(events, parseHeadingEvents(doc, "next_fest")...)
	return dedupeEvents(events)
}

func parseHeadingEvents(raw, category string) []Event {
	headingRe := regexp.MustCompile(`(?is)<h2[^>]*>(.*?)</h2>`)
	matches := headingRe.FindAllStringSubmatch(raw, -1)

	events := make([]Event, 0, len(matches))
	for _, match := range matches {
		heading := cleanText(match[1])
		if !strings.Contains(heading, "|") {
			continue
		}
		parts := strings.SplitN(heading, "|", 2)
		name := cleanText(parts[0])
		if category == "seasonal" && !strings.Contains(name, "Sale") {
			continue
		}
		if category == "next_fest" && !strings.Contains(name, "Next Fest") {
			continue
		}
		dateRange := strings.TrimSpace(strings.TrimSuffix(parts[1], "(ENDED)"))
		start, end, ok := parseWrittenDateRange(dateRange)
		if !ok {
			continue
		}
		if category == "next_fest" && name == "Next Fest" {
			name = "Steam Next Fest"
		}
		event := eventFromDates(name, start, end, category)
		event.Description = categoryDescription(category)
		event.InfoURL = headingInfoURL(raw, match[0])
		events = append(events, event)
	}
	return events
}

func parseFestTable(raw string) []Event {
	tableRe := regexp.MustCompile(`(?is)<strong>\s*2026 Fests\s*</strong>.*?<table>(?P<table>.*?)</table>`)
	tableMatch := tableRe.FindStringSubmatch(raw)
	if len(tableMatch) < 2 {
		return nil
	}

	rowRe := regexp.MustCompile(`(?is)<tr>(.*?)</tr>`)
	cellRe := regexp.MustCompile(`(?is)<td>(.*?)</td>`)
	rows := rowRe.FindAllStringSubmatch(tableMatch[1], -1)

	events := make([]Event, 0, len(rows))
	for _, row := range rows {
		cells := cellRe.FindAllStringSubmatch(row[1], -1)
		if len(cells) < 2 {
			continue
		}
		start, end, ok := parseFestDateRange(cells[0][1], 2026)
		if !ok {
			continue
		}
		event := eventFromDates(cleanText(cells[1][1]), start, end, "fest")
		if len(cells) > 2 {
			event.RegistrationURL = firstHref(cells[2][1])
		}
		if len(cells) > 3 {
			event.Notes = cleanText(cells[3][1])
			event.Description = festDescription(event.Notes)
			if event.Description != "" {
				event.Notes = event.Description
			}
			event.InfoURL = firstDocHref(cells[3][1])
		}
		if event.Description == "" {
			event.Description = categoryDescription("fest")
		}
		events = append(events, event)
	}
	return events
}

func parseWrittenDateRange(value string) (time.Time, time.Time, bool) {
	fullRangeRe := regexp.MustCompile(`(?i)([A-Z][a-z]+)\s+(\d{1,2})\s*-\s*([A-Z][a-z]+)\s+(\d{1,2}),\s*(20\d{2})`)
	if match := fullRangeRe.FindStringSubmatch(value); len(match) == 6 {
		startMonth, startOK := months[match[1]]
		endMonth, endOK := months[match[3]]
		if !startOK || !endOK {
			return time.Time{}, time.Time{}, false
		}
		endYear := atoi(match[5])
		startYear := endYear
		if startMonth > endMonth {
			startYear--
		}
		return dateUTC(startYear, startMonth, atoi(match[2])), dateUTC(endYear, endMonth, atoi(match[4])), true
	}

	sameMonthRe := regexp.MustCompile(`(?i)([A-Z][a-z]+)\s+(\d{1,2})\s*-\s*(\d{1,2}),\s*(20\d{2})`)
	if match := sameMonthRe.FindStringSubmatch(value); len(match) == 5 {
		month, ok := months[match[1]]
		if !ok {
			return time.Time{}, time.Time{}, false
		}
		year := atoi(match[4])
		return dateUTC(year, month, atoi(match[2])), dateUTC(year, month, atoi(match[3])), true
	}

	return time.Time{}, time.Time{}, false
}

func parseFestDateRange(raw string, year int) (time.Time, time.Time, bool) {
	text := cleanText(strings.ReplaceAll(raw, "<br>", " - "))
	rangeRe := regexp.MustCompile(`(?i)([A-Z][a-z]+)\s+(\d{1,2})\s*-\s*([A-Z][a-z]+)\s+(\d{1,2})`)
	match := rangeRe.FindStringSubmatch(text)
	if len(match) != 5 {
		return time.Time{}, time.Time{}, false
	}
	startMonth, startOK := months[match[1]]
	endMonth, endOK := months[match[3]]
	if !startOK || !endOK {
		return time.Time{}, time.Time{}, false
	}
	endYear := year
	if endMonth < startMonth {
		endYear++
	}
	return dateUTC(year, startMonth, atoi(match[2])), dateUTC(endYear, endMonth, atoi(match[4])), true
}

func eventFromDates(name string, start, end time.Time, category string) Event {
	return Event{
		Name:        name,
		StartDate:   start.Format(time.DateOnly),
		EndDate:     end.Format(time.DateOnly),
		Source:      "steamworks",
		Category:    category,
		Timezone:    "PT",
		Description: categoryDescription(category),
	}
}

func categoryDescription(category string) string {
	switch category {
	case "seasonal":
		return "Steam-wide seasonal sale event. Games released at least 30 days before the event start date can participate with a discount."
	case "fest":
		return "Themed sale event spotlighting a particular category of games with corresponding eligibility criteria."
	case "next_fest":
		return "Multi-day celebration of upcoming games with playable demos, livestreams, developer chats, and early player feedback."
	default:
		return ""
	}
}

func festDescription(notes string) string {
	notes = strings.TrimSpace(notes)
	if notes == "" || strings.Contains(notes, "This Fest has ended") {
		return ""
	}
	notes = regexp.MustCompile(`\s*More info\s*\.\s*$`).ReplaceAllString(notes, "")
	return strings.TrimSpace(notes)
}

func headingInfoURL(raw string, headingHTML string) string {
	start := strings.Index(raw, headingHTML)
	if start < 0 {
		return ""
	}
	section := raw[start+len(headingHTML):]
	if next := strings.Index(strings.ToLower(section), "<h2"); next >= 0 {
		section = section[:next]
	}
	return firstDocHref(section)
}

func firstHref(raw string) string {
	hrefRe := regexp.MustCompile(`(?is)<a[^>]+href="([^"]+)"`)
	match := hrefRe.FindStringSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	return html.UnescapeString(match[1])
}

func firstDocHref(raw string) string {
	hrefRe := regexp.MustCompile(`(?is)<a[^>]+href="([^"]+)"[^>]*class="[^"]*bb_doclink[^"]*"`)
	match := hrefRe.FindStringSubmatch(raw)
	if len(match) >= 2 {
		return html.UnescapeString(match[1])
	}
	return firstHref(raw)
}

func dedupeEvents(events []Event) []Event {
	seen := map[string]bool{}
	unique := make([]Event, 0, len(events))
	for _, event := range events {
		key := event.Name + "|" + event.StartDate + "|" + event.EndDate
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, event)
	}
	return unique
}

func eventStatus(today, start, end time.Time) string {
	if !today.Before(start) && !today.After(end) {
		return "active"
	}
	if today.Before(start) {
		return "upcoming"
	}
	return "recent"
}

func parseISODate(value string) (time.Time, error) {
	return time.ParseInLocation(time.DateOnly, value, time.Local)
}

func dateUTC(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func documentationSection(value string) string {
	re := regexp.MustCompile(`(?is)<div class="documentation_bbcode">(.*?)<div id="hashLocationHighlight">`)
	match := re.FindStringSubmatch(value)
	if len(match) == 2 {
		return match[1]
	}
	return value
}

func cleanText(value string) string {
	value = strings.ReplaceAll(value, "<br />", " ")
	value = strings.ReplaceAll(value, "<br/>", " ")
	value = strings.ReplaceAll(value, "<br>", " ")
	return strings.Join(strings.Fields(stripTags(value)), " ")
}

func stripTags(value string) string {
	tagRe := regexp.MustCompile(`<[^>]+>`)
	return tagRe.ReplaceAllString(value, " ")
}

func atoi(value string) int {
	var n int
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + int(ch-'0')
	}
	return n
}
