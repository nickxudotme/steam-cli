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
	lang := strings.TrimSpace(c.Lang)
	if lang == "" {
		lang = "english"
	}
	parsed, err := c.steamworksEvents(lang)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch live Steamworks upcoming events: %w", err)
	}
	if len(parsed) == 0 && !strings.EqualFold(lang, "english") {
		parsed, err = c.steamworksEvents("english")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch live Steamworks upcoming events: %w", err)
		}
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("no live Steamworks events could be parsed from %s", SteamworksUpcomingEvents)
	}
	return FilterEvents(parsed, query), nil
}

func (c *Client) steamworksEvents(lang string) ([]Event, error) {
	body, err := c.GetText(SteamworksUpcomingEvents, url.Values{"l": {lang}})
	if err != nil {
		return nil, err
	}
	return ParseSteamworksEvents(body), nil
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
	tableRe := regexp.MustCompile(`(?is)<table\b[^>]*>(.*?)</table>`)
	tableRanges := tableRe.FindAllStringSubmatchIndex(doc, -1)
	if len(tableRanges) == 0 {
		events = append(events, parseHeadingEvents(doc, "seasonal", sectionDescription(doc, false))...)
		return dedupeEvents(events)
	}

	firstTable := tableRanges[0]
	seasonalSegment := doc[:firstTable[0]]
	nextFestSegment := doc[firstTable[1]:]
	events = append(events, parseHeadingEvents(seasonalSegment, "seasonal", sectionDescription(seasonalSegment, false))...)
	events = append(events, parseFestTables(doc, tableRanges, sectionDescription(seasonalSegment, true))...)
	events = append(events, parseHeadingEvents(nextFestSegment, "next_fest", sectionDescription(nextFestSegment, false))...)
	return dedupeEvents(events)
}

func parseHeadingEvents(raw, category, description string) []Event {
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
		dateRange := cleanDateText(parts[1])
		start, end, ok := parseEventDateRange(dateRange, 0)
		if !ok {
			continue
		}
		if category == "next_fest" && name == "Next Fest" {
			name = "Steam Next Fest"
		}
		event := eventFromDates(name, start, end, category)
		if description != "" {
			event.Description = description
		}
		event.InfoURL = headingInfoURL(raw, match[0])
		events = append(events, event)
	}
	return events
}

func parseFestTables(raw string, tableRanges [][]int, fallbackDescription string) []Event {
	rowRe := regexp.MustCompile(`(?is)<tr\b[^>]*>(.*?)</tr>`)
	cellRe := regexp.MustCompile(`(?is)<td\b[^>]*>(.*?)</td>`)

	events := []Event{}
	for _, tableRange := range tableRanges {
		if len(tableRange) < 4 {
			continue
		}
		tableHTML := raw[tableRange[2]:tableRange[3]]
		year := nearestYearBefore(raw[:tableRange[0]])
		rows := rowRe.FindAllStringSubmatch(tableHTML, -1)
		for _, row := range rows {
			cells := cellRe.FindAllStringSubmatch(row[1], -1)
			if len(cells) < 2 {
				continue
			}
			start, end, ok := parseEventDateRange(cells[0][1], year)
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
				event.Description = fallbackDescription
			}
			if event.Description == "" {
				event.Description = categoryDescription(event.Category)
			}
			events = append(events, event)
		}
	}
	return events
}

func parseEventDateRange(value string, defaultYear int) (time.Time, time.Time, bool) {
	raw := value
	cleaned := cleanDateText(value)
	if start, end, ok := parseWrittenDateRange(cleaned); ok {
		return start, end, true
	}
	if start, end, ok := parseLocalizedWrittenDateRange(cleaned); ok {
		return start, end, true
	}
	if defaultYear > 0 {
		if start, end, ok := parseFestDateRange(raw, defaultYear); ok {
			return start, end, true
		}
		if start, end, ok := parseEnglishMonthDayRange(cleaned, defaultYear); ok {
			return start, end, true
		}
		if start, end, ok := parseLocalizedMonthDayRange(cleaned, defaultYear); ok {
			return start, end, true
		}
	}
	return time.Time{}, time.Time{}, false
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
	text := cleanText(regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(raw, " - "))
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

func parseEnglishMonthDayRange(value string, year int) (time.Time, time.Time, bool) {
	text := cleanText(value)
	dateRe := regexp.MustCompile(`(?i)\b([A-Z][a-z]+)\s+(\d{1,2})\b`)
	matches := dateRe.FindAllStringSubmatch(text, -1)
	if len(matches) < 2 {
		return time.Time{}, time.Time{}, false
	}
	startMonth, startOK := months[matches[0][1]]
	endMonth, endOK := months[matches[1][1]]
	if !startOK || !endOK {
		return time.Time{}, time.Time{}, false
	}
	endYear := year
	if endMonth < startMonth {
		endYear++
	}
	return dateUTC(year, startMonth, atoi(matches[0][2])), dateUTC(endYear, endMonth, atoi(matches[1][2])), true
}

func parseLocalizedWrittenDateRange(value string) (time.Time, time.Time, bool) {
	text := normalizeDateSeparators(cleanText(value))
	rangeRe := regexp.MustCompile(`(?i)(20\d{2})\s*年?\s*(\d{1,2})\s*月\s*(\d{1,2})\s*日?\s*-\s*(?:(20\d{2})\s*年?\s*)?(\d{1,2})\s*月\s*(\d{1,2})\s*日?`)
	if match := rangeRe.FindStringSubmatch(text); len(match) == 7 {
		startYear := atoi(match[1])
		endYear := startYear
		if match[4] != "" {
			endYear = atoi(match[4])
		}
		return dateUTC(startYear, time.Month(atoi(match[2])), atoi(match[3])), dateUTC(endYear, time.Month(atoi(match[5])), atoi(match[6])), true
	}

	sameMonthRe := regexp.MustCompile(`(?i)(20\d{2})\s*年?\s*(\d{1,2})\s*月\s*(\d{1,2})\s*日?\s*-\s*(\d{1,2})\s*日?`)
	if match := sameMonthRe.FindStringSubmatch(text); len(match) == 5 {
		year := atoi(match[1])
		month := time.Month(atoi(match[2]))
		return dateUTC(year, month, atoi(match[3])), dateUTC(year, month, atoi(match[4])), true
	}
	return time.Time{}, time.Time{}, false
}

func parseLocalizedMonthDayRange(value string, year int) (time.Time, time.Time, bool) {
	text := normalizeDateSeparators(cleanText(value))
	dateRe := regexp.MustCompile(`(\d{1,2})\s*月\s*(\d{1,2})\s*日?`)
	matches := dateRe.FindAllStringSubmatch(text, -1)
	if len(matches) < 2 {
		return time.Time{}, time.Time{}, false
	}
	startMonth := time.Month(atoi(matches[0][1]))
	endMonth := time.Month(atoi(matches[1][1]))
	endYear := year
	if endMonth < startMonth {
		endYear++
	}
	return dateUTC(year, startMonth, atoi(matches[0][2])), dateUTC(endYear, endMonth, atoi(matches[1][2])), true
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
	if notes == "" {
		return ""
	}
	if ended := endedFestDescription(notes); ended != "" {
		return ended
	}
	notes = regexp.MustCompile(`\s*(More info|更多信息|详细信息|詳細情報)\s*[.。]\s*$`).ReplaceAllString(notes, "")
	return strings.TrimSpace(notes)
}

func endedFestDescription(notes string) string {
	endedPhrases := []string{
		"This Fest has ended.",
		"此游戏节已结束。",
		"このフェスは終了しました。",
	}
	for _, phrase := range endedPhrases {
		if strings.Contains(notes, phrase) {
			return phrase
		}
	}
	return ""
}

func sectionDescription(raw string, last bool) string {
	headingRe := regexp.MustCompile(`(?is)<h2[^>]*class="[^"]*bb_section[^"]*"[^>]*>.*?</h2>`)
	matches := headingRe.FindAllStringIndex(raw, -1)
	if len(matches) == 0 {
		headingRe = regexp.MustCompile(`(?is)<h2[^>]*>.*?</h2>`)
		matches = headingRe.FindAllStringIndex(raw, -1)
	}
	if len(matches) == 0 {
		return ""
	}
	index := 0
	if last {
		index = len(matches) - 1
	}
	start := matches[index][1]
	end := len(raw)
	nextHeadingRe := regexp.MustCompile(`(?is)<h2[^>]*>`)
	if next := nextHeadingRe.FindStringIndex(raw[start:]); len(next) == 2 {
		end = start + next[0]
	}
	return cleanText(raw[start:end])
}

func cleanDateText(value string) string {
	value = cleanText(value)
	value = regexp.MustCompile(`\([^)]*\)|（[^）]*）`).ReplaceAllString(value, "")
	return strings.TrimSpace(value)
}

func normalizeDateSeparators(value string) string {
	replacer := strings.NewReplacer(
		"－", "-",
		"–", "-",
		"—", "-",
		"~", "-",
		"～", "-",
		"至", "-",
	)
	return replacer.Replace(value)
}

func nearestYearBefore(value string) int {
	yearRe := regexp.MustCompile(`20\d{2}`)
	matches := yearRe.FindAllString(value, -1)
	if len(matches) == 0 {
		return time.Now().Year()
	}
	return atoi(matches[len(matches)-1])
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
	re := regexp.MustCompile(`(?is)<div[^>]*class="[^"]*\bdocumentation_bbcode\b[^"]*"[^>]*>(.*?)<div[^>]*id="hashLocationHighlight"`)
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
