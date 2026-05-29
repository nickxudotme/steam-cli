package steam

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const SteamworksUpcomingEvents = "https://partner.steamgames.com/doc/marketing/upcoming_events"
const SteamStoreSpecials = "https://store.steampowered.com/specials"

const maxDiscoveredStoreSalePages = 16

type Event struct {
	Name                 string   `json:"name"`
	StartDate            string   `json:"start_date"`
	EndDate              string   `json:"end_date"`
	StartTime            int64    `json:"start_time,omitempty"`
	EndTime              int64    `json:"end_time,omitempty"`
	Status               string   `json:"status"`
	Source               string   `json:"source"`
	Sources              []string `json:"sources,omitempty"`
	Category             string   `json:"category"`
	Timezone             string   `json:"timezone"`
	Description          string   `json:"description,omitempty"`
	Notes                string   `json:"notes,omitempty"`
	RegistrationURL      string   `json:"registration_url,omitempty"`
	InfoURL              string   `json:"info_url,omitempty"`
	StoreURL             string   `json:"store_url,omitempty"`
	EventGID             string   `json:"event_gid,omitempty"`
	ClanSteamID          string   `json:"clan_steamid,omitempty"`
	EventType            int      `json:"event_type,omitempty"`
	AppID                int      `json:"appid,omitempty"`
	GroupName            string   `json:"group_name,omitempty"`
	GroupURL             string   `json:"group_url,omitempty"`
	AnnouncementHeadline string   `json:"announcement_headline,omitempty"`
	AnnouncementURL      string   `json:"announcement_url,omitempty"`
	LastModifiedTime     int64    `json:"last_modified_time,omitempty"`
	VisibilityStartTime  int64    `json:"visibility_start_time,omitempty"`
	VisibilityEndTime    int64    `json:"visibility_end_time,omitempty"`
	ImageURL             string   `json:"image_url,omitempty"`
	TitleImageURL        string   `json:"title_image_url,omitempty"`
	CapsuleImageURL      string   `json:"capsule_image_url,omitempty"`
	BackgroundImageURL   string   `json:"background_image_url,omitempty"`
}

type EventQuery struct {
	PastDays          int
	FutureDays        int
	IncludeStoreSales bool
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
	parsed, saleURLs, err := c.steamworksEventsAndSaleURLs(lang)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch live Steamworks upcoming events: %w", err)
	}
	if len(parsed) == 0 && !strings.EqualFold(lang, "english") {
		parsed, saleURLs, err = c.steamworksEventsAndSaleURLs("english")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch live Steamworks upcoming events: %w", err)
		}
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("no live Steamworks events could be parsed from %s", SteamworksUpcomingEvents)
	}
	if query.IncludeStoreSales {
		if storeEvents, err := c.storeSaleEvents(query, lang, saleURLs); err == nil {
			parsed = mergeStoreEvents(parsed, storeEvents)
		}
	}
	return FilterEvents(parsed, query), nil
}

func (c *Client) steamworksEvents(lang string) ([]Event, error) {
	events, _, err := c.steamworksEventsAndSaleURLs(lang)
	return events, err
}

func (c *Client) steamworksEventsAndSaleURLs(lang string) ([]Event, []string, error) {
	body, err := c.GetText(SteamworksUpcomingEvents, url.Values{"l": {lang}})
	if err != nil {
		return nil, nil, err
	}
	return ParseSteamworksEvents(body), ParseSteamStoreSaleURLs(body, c.Endpoints.Store), nil
}

func (c *Client) storeSaleEvents(query EventQuery, lang string, seedURLs []string) ([]Event, error) {
	saleURLs := c.storeSaleCandidateURLs(query)
	saleURLs = append(saleURLs, seedURLs...)
	if discovered, err := c.storeSaleURLsFromSpecials(lang); err == nil {
		saleURLs = append(saleURLs, discovered...)
	}

	events := []Event{}
	for _, saleURL := range uniqueStrings(saleURLs) {
		body, err := c.GetText(saleURL, url.Values{"l": {lang}})
		if err != nil {
			continue
		}
		events = append(events, ParseSteamStoreSalePageWithLang(body, saleURL, lang)...)
	}
	return dedupeEvents(events), nil
}

func (c *Client) storeSaleURLsFromSpecials(lang string) ([]string, error) {
	body, err := c.GetText(c.Endpoints.Store+"/specials", url.Values{"l": {lang}})
	if err != nil {
		return nil, err
	}
	return ParseSteamStoreSaleURLs(body, c.Endpoints.Store), nil
}

func (c *Client) storeSaleCandidateURLs(query EventQuery) []string {
	years := eventQueryYears(query, time.Now())
	formats := []string{
		"lny%d",
		"lunarnewyear%d",
		"chinesenewyear%d",
	}

	urls := make([]string, 0, len(years)*len(formats))
	for _, year := range years {
		for _, format := range formats {
			urls = append(urls, c.Endpoints.Store+"/sale/"+fmt.Sprintf(format, year))
		}
	}
	return urls
}

func eventQueryYears(query EventQuery, now time.Time) []int {
	minYear := now.AddDate(0, 0, -query.PastDays).Year()
	maxYear := now.AddDate(0, 0, query.FutureDays).Year()
	if minYear > maxYear {
		minYear, maxYear = maxYear, minYear
	}
	if maxYear-minYear > 8 {
		minYear = now.Year() - 2
		maxYear = now.Year() + 2
	}

	years := make([]int, 0, maxYear-minYear+1)
	for year := minYear; year <= maxYear; year++ {
		years = append(years, year)
	}
	return years
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

func mergeStoreEvents(events, storeEvents []Event) []Event {
	merged := append([]Event(nil), events...)
	for _, storeEvent := range storeEvents {
		if index := matchingEventIndex(merged, storeEvent); index >= 0 {
			mergeEventDetails(&merged[index], storeEvent)
			continue
		}
		merged = append(merged, storeEvent)
	}
	return dedupeEvents(merged)
}

func matchingEventIndex(events []Event, storeEvent Event) int {
	for index, event := range events {
		if event.StoreURL != "" && event.StoreURL == storeEvent.StoreURL {
			return index
		}
		if event.StartDate != storeEvent.StartDate || event.EndDate != storeEvent.EndDate {
			continue
		}
		if similarEventName(event.Name, storeEvent.Name) {
			return index
		}
	}
	return -1
}

func similarEventName(a, b string) bool {
	a = eventMatchName(a)
	b = eventMatchName(b)
	if a == "" || b == "" {
		return false
	}
	return a == b || strings.Contains(a, b) || strings.Contains(b, a)
}

func eventMatchName(value string) string {
	value = strings.ToLower(cleanText(value))
	value = regexp.MustCompile(`\b20\d{2}\b`).ReplaceAllString(value, "")
	value = strings.NewReplacer("steam", "", "sale", "", "fest", "", "特卖", "", "游戏节", "", "新品节", "next").Replace(value)
	value = regexp.MustCompile(`[^a-z0-9\p{Han}]+`).ReplaceAllString(value, "")
	return strings.TrimSpace(value)
}

func mergeEventDetails(target *Event, storeEvent Event) {
	target.Sources = appendUniqueStrings(target.Sources, "steamworks", "steam_store")
	copyStringIfEmpty(&target.StoreURL, storeEvent.StoreURL)
	copyStringIfEmpty(&target.EventGID, storeEvent.EventGID)
	copyStringIfEmpty(&target.ClanSteamID, storeEvent.ClanSteamID)
	copyStringIfEmpty(&target.GroupName, storeEvent.GroupName)
	copyStringIfEmpty(&target.GroupURL, storeEvent.GroupURL)
	copyStringIfEmpty(&target.AnnouncementHeadline, storeEvent.AnnouncementHeadline)
	copyStringIfEmpty(&target.AnnouncementURL, storeEvent.AnnouncementURL)
	copyStringIfEmpty(&target.ImageURL, storeEvent.ImageURL)
	copyStringIfEmpty(&target.TitleImageURL, storeEvent.TitleImageURL)
	copyStringIfEmpty(&target.CapsuleImageURL, storeEvent.CapsuleImageURL)
	copyStringIfEmpty(&target.BackgroundImageURL, storeEvent.BackgroundImageURL)
	if target.StartTime == 0 {
		target.StartTime = storeEvent.StartTime
	}
	if target.EndTime == 0 {
		target.EndTime = storeEvent.EndTime
	}
	if target.EventType == 0 {
		target.EventType = storeEvent.EventType
	}
	if target.AppID == 0 {
		target.AppID = storeEvent.AppID
	}
	if target.LastModifiedTime == 0 {
		target.LastModifiedTime = storeEvent.LastModifiedTime
	}
	if target.VisibilityStartTime == 0 {
		target.VisibilityStartTime = storeEvent.VisibilityStartTime
	}
	if target.VisibilityEndTime == 0 {
		target.VisibilityEndTime = storeEvent.VisibilityEndTime
	}
	if target.Description == "" || target.Description == categoryDescription(target.Category) {
		target.Description = storeEvent.Description
	}
	if target.Notes == "" {
		target.Notes = storeEvent.Notes
	}
}

func copyStringIfEmpty(target *string, value string) {
	if *target == "" {
		*target = value
	}
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

// ParseSteamStoreSalePage extracts the top-level event metadata embedded in
// public Steam Store sale pages such as /sale/lny2026.
func ParseSteamStoreSalePage(raw, pageURL string) []Event {
	return ParseSteamStoreSalePageWithLang(raw, pageURL, "english")
}

// ParseSteamStoreSalePageWithLang extracts the top-level event metadata
// embedded in public Steam Store sale pages such as /sale/lny2026.
func ParseSteamStoreSalePageWithLang(raw, pageURL, lang string) []Event {
	attr := htmlDataAttr(raw, "data-partnereventstore")
	if attr == "" {
		return nil
	}

	var storeEvents []storeSaleEvent
	if err := json.Unmarshal([]byte(attr), &storeEvents); err != nil {
		return nil
	}
	if len(storeEvents) == 0 {
		return nil
	}

	group := storeSaleGroupInfo(raw)
	title := metaContent(raw, "og:title")
	imageURL := metaContent(raw, "og:image")
	if imageURL == "" {
		imageURL = metaContent(raw, "twitter:image")
	}
	backgroundURL := storeSaleBackgroundURL(raw)
	for _, storeEvent := range storeEvents {
		if storeEvent.EventName == "" || storeEvent.StartTime == 0 || storeEvent.EndTime == 0 {
			continue
		}
		details := storeEventDetails(storeEvent, group, lang)
		name := storeEvent.EventName
		if title != "" && !strings.EqualFold(title, "Steam") {
			name = title
		}
		if details.Title != "" {
			name = details.Title
		}
		description := details.Description
		if description == "" {
			description = metaContent(raw, "og:description")
		}
		if description == "" {
			description = metaContent(raw, "description")
		}
		if strings.EqualFold(description, "Steam is the ultimate destination for playing, discussing, and creating games.") ||
			strings.EqualFold(description, name) ||
			strings.EqualFold(description, title) ||
			strings.EqualFold(description, storeEvent.EventName) ||
			strings.EqualFold(description, details.Title) {
			description = ""
		}
		if description == "" && details.AnnouncementHeadline != "" &&
			!strings.EqualFold(details.AnnouncementHeadline, name) &&
			!strings.EqualFold(details.AnnouncementHeadline, title) &&
			!strings.EqualFold(details.AnnouncementHeadline, storeEvent.EventName) &&
			!strings.EqualFold(details.AnnouncementHeadline, details.Title) {
			description = details.AnnouncementHeadline
		}
		if description == "" && group.GroupName != "" {
			description = "Steam Store sale page presented by " + group.GroupName + "."
		}
		if description == "" {
			description = "Steam Store sale page."
		}
		if imageURL == "" {
			imageURL = details.CapsuleImageURL
		}
		if backgroundURL == "" {
			backgroundURL = details.BackgroundImageURL
		}

		event := Event{
			Name:                 cleanText(name),
			StartDate:            unixDateInPacific(storeEvent.StartTime),
			EndDate:              unixDateInPacific(storeEvent.EndTime),
			StartTime:            storeEvent.StartTime,
			EndTime:              storeEvent.EndTime,
			Source:               "steam_store",
			Sources:              []string{"steam_store"},
			Category:             "store_sale",
			Timezone:             "PT",
			Description:          description,
			InfoURL:              pageURL,
			StoreURL:             pageURL,
			EventGID:             storeEvent.GID,
			ClanSteamID:          storeEvent.ClanSteamID,
			EventType:            storeEvent.EventType,
			AppID:                storeEvent.AppID,
			GroupName:            group.GroupName,
			GroupURL:             group.URL(storeEvent.ClanSteamID),
			AnnouncementHeadline: details.AnnouncementHeadline,
			AnnouncementURL:      details.AnnouncementURL,
			LastModifiedTime:     storeEvent.LastModifiedTime,
			VisibilityStartTime:  storeEvent.VisibilityStartTime,
			VisibilityEndTime:    storeEvent.VisibilityEndTime,
			ImageURL:             imageURL,
			TitleImageURL:        details.TitleImageURL,
			CapsuleImageURL:      details.CapsuleImageURL,
			BackgroundImageURL:   backgroundURL,
		}
		if details.Notes != "" {
			event.Notes = details.Notes
		} else if group.GroupName != "" {
			event.Notes = "Steam Store sale page presented by " + group.GroupName + "."
		}
		return []Event{event}
	}
	return nil
}

// ParseSteamStoreSaleURLs extracts sale page URLs from public Store hub HTML.
func ParseSteamStoreSaleURLs(raw, storeBase string) []string {
	urls := []string{}

	attr := htmlDataAttr(raw, "data-ch_spotlights_data")
	if attr != "" {
		var spotlights []struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(attr), &spotlights); err == nil {
			for _, spotlight := range spotlights {
				urls = append(urls, normalizeStoreSaleURL(spotlight.URL, storeBase))
			}
		}
	}

	cleaned := strings.ReplaceAll(html.UnescapeString(raw), `\/`, `/`)
	saleURLRe := regexp.MustCompile(`(?i)(?:https?://[^"'<>\s]+)?/sale/[a-z0-9_-]+`)
	for _, match := range saleURLRe.FindAllString(cleaned, -1) {
		urls = append(urls, normalizeStoreSaleURL(match, storeBase))
	}

	unique := uniqueStrings(urls)
	if len(unique) > maxDiscoveredStoreSalePages {
		return unique[:maxDiscoveredStoreSalePages]
	}
	return unique
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
		Sources:     []string{"steamworks"},
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
	case "store_sale":
		return "Steam Store sale page."
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

type storeSaleEvent struct {
	GID                 string                 `json:"gid"`
	ClanSteamID         string                 `json:"clan_steamid"`
	EventName           string                 `json:"event_name"`
	EventType           int                    `json:"event_type"`
	AppID               int                    `json:"appid"`
	StartTime           int64                  `json:"rtime32_start_time"`
	EndTime             int64                  `json:"rtime32_end_time"`
	EventNotes          string                 `json:"event_notes"`
	JSONData            string                 `json:"jsondata"`
	AnnouncementBody    *storeSaleAnnouncement `json:"announcement_body"`
	LastModifiedTime    int64                  `json:"rtime32_last_modified"`
	VisibilityStartTime int64                  `json:"rtime32_visibility_start"`
	VisibilityEndTime   int64                  `json:"rtime32_visibility_end"`
}

type storeSaleGroup struct {
	ClanAccountID int64  `json:"clanAccountID"`
	VanityURL     string `json:"vanity_url"`
	GroupName     string `json:"group_name"`
	AvatarFullURL string `json:"avatar_full_url"`
}

func (g storeSaleGroup) URL(clanSteamID string) string {
	if g.VanityURL != "" {
		return "https://steamcommunity.com/groups/" + g.VanityURL
	}
	if clanSteamID != "" {
		return "https://steamcommunity.com/gid/" + clanSteamID
	}
	return ""
}

type storeSaleAnnouncement struct {
	GID        string `json:"gid"`
	ClanID     string `json:"clanid"`
	Headline   string `json:"headline"`
	Body       string `json:"body"`
	PostTime   int64  `json:"posttime"`
	UpdateTime int64  `json:"updatetime"`
}

type storeSaleJSONData struct {
	LocalizedSubtitle     []string `json:"localized_subtitle"`
	LocalizedSummary      []string `json:"localized_summary"`
	LocalizedTitleImage   []string `json:"localized_title_image"`
	LocalizedCapsuleImage []string `json:"localized_capsule_image"`
	LocalizedSaleHeader   []string `json:"localized_sale_header"`
	LocalizedSaleLogo     []string `json:"localized_sale_logo"`
	SaleVanityID          string   `json:"sale_vanity_id"`
	SaleBackgroundColor   string   `json:"sale_background_color"`
	SaleAssociatedAppID   int      `json:"sale_associated_advertising_appid"`
	ReferencedAppIDs      []int    `json:"referenced_appids"`
}

type storeSaleDetails struct {
	Title                string
	Description          string
	Notes                string
	TitleImageURL        string
	CapsuleImageURL      string
	BackgroundImageURL   string
	AnnouncementHeadline string
	AnnouncementURL      string
}

func htmlDataAttr(raw, name string) string {
	attrRe := regexp.MustCompile(`(?is)\b` + regexp.QuoteMeta(name) + `="([^"]*)"`)
	match := attrRe.FindStringSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	return html.UnescapeString(match[1])
}

func metaContent(raw, property string) string {
	metaRe := regexp.MustCompile(`(?is)<meta\b[^>]*(?:property|name)=["']` + regexp.QuoteMeta(property) + `["'][^>]*\bcontent=["']([^"']*)["'][^>]*>`)
	match := metaRe.FindStringSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	return cleanText(html.UnescapeString(match[1]))
}

func storeSaleBackgroundURL(raw string) string {
	bgRe := regexp.MustCompile(`(?is)class=["'][^"']*\breact_landing_background\b[^"']*["'][^>]*background-image:\s*url\(["']?([^"')]+)["']?\)`)
	match := bgRe.FindStringSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	return cleanText(html.UnescapeString(match[1]))
}

func storeSaleGroupInfo(raw string) storeSaleGroup {
	attr := htmlDataAttr(raw, "data-groupvanityinfo")
	if attr == "" {
		return storeSaleGroup{}
	}
	var groups []storeSaleGroup
	if err := json.Unmarshal([]byte(attr), &groups); err != nil || len(groups) == 0 {
		return storeSaleGroup{}
	}
	groups[0].GroupName = cleanText(groups[0].GroupName)
	groups[0].AvatarFullURL = cleanText(groups[0].AvatarFullURL)
	return groups[0]
}

func storeEventDetails(event storeSaleEvent, group storeSaleGroup, lang string) storeSaleDetails {
	details := storeSaleDetails{}
	if event.AnnouncementBody != nil {
		details.AnnouncementHeadline = cleanText(event.AnnouncementBody.Headline)
		details.AnnouncementURL = announcementURL(event, event.AnnouncementBody.GID)
	}
	notes := cleanText(event.EventNotes)
	if notes != "" && !strings.EqualFold(notes, "see announcement body") {
		details.Notes = notes
	}
	if event.JSONData == "" {
		if details.Description == "" && event.AnnouncementBody != nil {
			details.Description = announcementSummary(event.AnnouncementBody.Body)
		}
		return details
	}

	var data storeSaleJSONData
	if err := json.Unmarshal([]byte(event.JSONData), &data); err != nil {
		return details
	}
	details.Title = localizedText(data.LocalizedSaleHeader, lang)
	if details.Title == "" {
		details.Title = localizedText(data.LocalizedSubtitle, lang)
	}
	details.Description = localizedText(data.LocalizedSummary, lang)
	if details.Description == "" && event.AnnouncementBody != nil {
		details.Description = announcementSummary(event.AnnouncementBody.Body)
	}
	details.TitleImageURL = clanImageURL(group.ClanAccountID, localizedText(data.LocalizedTitleImage, lang))
	details.CapsuleImageURL = clanImageURL(group.ClanAccountID, localizedText(data.LocalizedCapsuleImage, lang))
	if details.TitleImageURL == "" {
		details.TitleImageURL = clanImageURL(group.ClanAccountID, localizedText(data.LocalizedSaleLogo, lang))
	}
	return details
}

func announcementURL(event storeSaleEvent, announcementGID string) string {
	if announcementGID == "" || event.ClanSteamID == "" {
		return ""
	}
	return "https://steamcommunity.com/gid/" + event.ClanSteamID + "/announcements/detail/" + announcementGID
}

func clanImageURL(clanAccountID int64, value string) string {
	value = cleanText(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if clanAccountID <= 0 {
		return value
	}
	return fmt.Sprintf("https://clan.akamai.steamstatic.com/images/%d/%s", clanAccountID, value)
}

func localizedText(values []string, lang string) string {
	if len(values) == 0 {
		return ""
	}
	if index, ok := steamLanguageIndex(lang); ok && index >= 0 && index < len(values) {
		if value := cleanText(values[index]); value != "" {
			return value
		}
	}
	for _, value := range values {
		if value := cleanText(value); value != "" {
			return value
		}
	}
	return ""
}

func steamLanguageIndex(lang string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "english", "en":
		return 0, true
	case "german", "de":
		return 1, true
	case "french", "fr":
		return 2, true
	case "italian", "it":
		return 3, true
	case "koreana", "korean", "ko":
		return 4, true
	case "spanish", "es":
		return 5, true
	case "schinese", "chinese", "zh-cn":
		return 6, true
	case "tchinese", "zh-tw":
		return 7, true
	case "russian", "ru":
		return 8, true
	case "thai", "th":
		return 9, true
	case "japanese", "ja":
		return 10, true
	case "portuguese", "pt":
		return 11, true
	case "polish", "pl":
		return 12, true
	case "danish", "da":
		return 13, true
	case "dutch", "nl":
		return 14, true
	case "finnish", "fi":
		return 15, true
	case "norwegian", "no":
		return 16, true
	case "swedish", "sv":
		return 17, true
	case "hungarian", "hu":
		return 18, true
	case "czech", "cs":
		return 19, true
	case "romanian", "ro":
		return 20, true
	case "turkish", "tr":
		return 21, true
	case "arabic", "ar":
		return 22, true
	case "bulgarian", "bg":
		return 23, true
	case "greek", "el":
		return 24, true
	case "vietnamese", "vi":
		return 25, true
	case "latam":
		return 27, true
	case "brazilian", "pt-br":
		return 28, true
	}
	return 0, false
}

func announcementSummary(value string) string {
	value = cleanBBCode(value)
	if value == "" {
		return ""
	}
	paragraphs := regexp.MustCompile(`\n{2,}`).Split(value, -1)
	for _, paragraph := range paragraphs {
		paragraph = cleanText(paragraph)
		if paragraph != "" {
			return paragraph
		}
	}
	return cleanText(value)
}

func cleanBBCode(value string) string {
	value = strings.ReplaceAll(value, `\/`, `/`)
	value = regexp.MustCompile(`(?is)\[dynamiclink[^\]]*\].*?\[/dynamiclink\]`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?is)\[(?:img|previewyoutube|video)[^\]]*\].*?\[/(?:img|previewyoutube|video)\]`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?is)\[(?:h1|h2|h3|p|list|olist|quote|b|i|u|url|code|table|tr|td|th|color|size|spoiler|strike|noparse)[^\]]*\]`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?is)\[/(?:h1|h2|h3|p|list|olist|quote|b|i|u|url|code|table|tr|td|th|color|size|spoiler|strike|noparse)\]`).ReplaceAllString(value, "\n\n")
	value = regexp.MustCompile(`(?is)\[[^\]]+\]`).ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	return strings.TrimSpace(value)
}

func unixDateInPacific(value int64) string {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.FixedZone("PT", -8*60*60)
	}
	return time.Unix(value, 0).In(loc).Format(time.DateOnly)
}

func normalizeStoreSaleURL(rawURL, storeBase string) string {
	rawURL = strings.TrimSpace(strings.ReplaceAll(html.UnescapeString(rawURL), `\/`, `/`))
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	base, err := url.Parse(storeBase)
	if err != nil {
		return ""
	}
	parsed.Scheme = base.Scheme
	parsed.Host = base.Host
	if !strings.HasPrefix(parsed.Path, "/sale/") {
		return ""
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func appendUniqueStrings(values []string, extra ...string) []string {
	return uniqueStrings(append(values, extra...))
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
