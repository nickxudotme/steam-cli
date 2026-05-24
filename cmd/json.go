package cmd

import (
	"strings"
	"time"
)

// jsonEnvelope is the unified output shape for all commands when --json is set.
type jsonEnvelope struct {
	OK          bool       `json:"ok"`
	Command     string     `json:"command"`
	Schema      string     `json:"schema"`
	GeneratedAt string     `json:"generated_at"`
	Meta        jsonMeta   `json:"meta"`
	Data        any        `json:"data,omitempty"`
	Error       *jsonError `json:"error,omitempty"`
}

type jsonMeta struct {
	Version    string       `json:"version"`
	CC         string       `json:"cc"`
	Lang       string       `json:"lang"`
	Timeout    int          `json:"timeout"`
	Quiet      bool         `json:"quiet"`
	NoColor    bool         `json:"no_color"`
	Verbose    bool         `json:"verbose"`
	UILang     string       `json:"ui_lang"`
	ObservedAt string       `json:"observed_at"`
	Sources    []sourceInfo `json:"sources,omitempty"`
}

type jsonError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type sourceInfo struct {
	Name       string `json:"name"`
	URL        string `json:"url,omitempty"`
	Type       string `json:"type"`
	Confidence string `json:"confidence"`
}

func commandSchema(commandPath string) string {
	parts := strings.Fields(commandPath)
	if len(parts) == 0 {
		return "steam-cli.unknown.v1"
	}
	return "steam-cli." + parts[len(parts)-1] + ".v1"
}

func responseMeta() jsonMeta {
	return responseMetaFor(errorCommandPath())
}

func responseMetaFor(commandPath string) jsonMeta {
	return jsonMeta{
		Version:    version,
		CC:         strings.ToUpper(opts.cc),
		Lang:       opts.lang,
		Timeout:    opts.timeout,
		Quiet:      opts.quiet,
		NoColor:    opts.noColor,
		Verbose:    opts.verbose,
		UILang:     opts.uiLang,
		ObservedAt: time.Now().UTC().Format(time.RFC3339),
		Sources:    commandSources(commandPath),
	}
}

// commandSources maps a command path to the public Steam endpoints it touches.
// Adding a new command requires registering it here so meta.sources is filled.
func commandSources(commandPath string) []sourceInfo {
	name := commandName(commandPath)
	common := map[string]sourceInfo{
		"appdetails":    {Name: "appdetails", URL: "https://store.steampowered.com/api/appdetails", Type: "official_store_api", Confidence: "high"},
		"storebrowse":   {Name: "IStoreBrowseService/GetItems", URL: "https://api.steampowered.com/IStoreBrowseService/GetItems/v1/", Type: "public_steam_api_observed", Confidence: "medium"},
		"appreviews":    {Name: "appreviews", URL: "https://store.steampowered.com/appreviews/{appid}", Type: "official_store_api", Confidence: "high"},
		"players":       {Name: "GetNumberOfCurrentPlayers", URL: "https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/", Type: "official_web_api", Confidence: "high"},
		"news":          {Name: "GetNewsForApp", URL: "https://api.steampowered.com/ISteamNews/GetNewsForApp/v2/", Type: "official_web_api", Confidence: "high"},
		"storesearch":   {Name: "storesearch", URL: "https://store.steampowered.com/api/storesearch", Type: "official_store_api", Confidence: "high"},
		"searchresults": {Name: "search/results", URL: "https://store.steampowered.com/search/results/", Type: "official_html_json", Confidence: "medium"},
		"achievements":  {Name: "GetGlobalAchievementPercentagesForApp", URL: "https://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/", Type: "official_web_api", Confidence: "high"},
		"events":        {Name: "Steamworks Upcoming Steam Events", URL: "https://partner.steamgames.com/doc/marketing/upcoming_events", Type: "official_html_parse", Confidence: "medium"},
		"storesales":    {Name: "Steam Store sale pages", URL: "https://store.steampowered.com/sale/{vanity}", Type: "public_store_html_parse", Confidence: "medium"},
		"community":     {Name: "Steam Community profile XML", URL: "https://steamcommunity.com/profiles/{steamid64}/?xml=1", Type: "public_community_xml", Confidence: "high"},
		"wishlist":      {Name: "IWishlistService/GetWishlist", URL: "https://api.steampowered.com/IWishlistService/GetWishlist/v1/", Type: "public_steam_api_observed", Confidence: "medium"},
		"cdn":           {Name: "Steam CDN", URL: "https://cdn.akamai.steamstatic.com/steam/apps/{appid}/...", Type: "cdn_convention", Confidence: "medium"},
		"storehome":     {Name: "Steam Store language menu", URL: "https://store.steampowered.com/", Type: "official_html_parse", Confidence: "medium"},
	}
	pick := func(keys ...string) []sourceInfo {
		out := make([]sourceInfo, 0, len(keys))
		for _, key := range keys {
			out = append(out, common[key])
		}
		return out
	}
	switch name {
	case "search":
		return pick("storesearch")
	case "app":
		return pick("appdetails", "storebrowse", "appreviews", "players", "news")
	case "price":
		return pick("appdetails", "storebrowse")
	case "media":
		return pick("appdetails", "storebrowse", "cdn")
	case "dlc":
		return pick("appdetails", "storebrowse")
	case "similar":
		return []sourceInfo{{Name: "IStoreQueryService/MoreLikeThis", URL: "https://api.steampowered.com/IStoreQueryService/MoreLikeThis/v1/", Type: "public_steam_api_observed", Confidence: "medium"}}
	case "deals":
		return pick("searchresults")
	case "reviews":
		return pick("appreviews")
	case "news":
		return pick("news")
	case "achievements":
		return pick("achievements")
	case "events":
		return pick("events", "storesales")
	case "user":
		return pick("community")
	case "wishlist":
		return pick("wishlist", "appdetails")
	case "locales":
		return pick("storehome", "appdetails")
	case "doctor":
		return pick("appdetails", "players", "community", "events")
	default:
		return nil
	}
}

func commandName(commandPath string) string {
	parts := strings.Fields(commandPath)
	if len(parts) == 0 {
		return ""
	}
	name := parts[len(parts)-1]
	switch name {
	case "find", "lookup":
		return "search"
	case "game", "info":
		return "app"
	case "prices":
		return "price"
	case "images", "assets":
		return "media"
	case "recommend", "recommendations":
		return "similar"
	case "sale", "sales", "topsellers":
		return "deals"
	case "fests", "festivals":
		return "events"
	default:
		return name
	}
}
