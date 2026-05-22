package i18n

import (
	"strings"
)

type Language string

const (
	Auto Language = "auto"
	EN   Language = "en"
	ZhCN Language = "zh-CN"
)

var current = EN

// Set picks the active UI language. "auto" runs system-locale detection
// (sync.Once-cached via DetectSystemLocale).
func Set(value string) Language {
	lang := Normalize(value)
	if lang == Auto {
		lang = languageFromSystemLocale(DetectSystemLocale())
	}
	current = lang
	return current
}

func Current() Language {
	return current
}

// Normalize folds common spellings of the supported UI languages into one of
// Auto / EN / ZhCN. Anything unrecognized maps to Auto so detection can run.
func Normalize(value string) Language {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return Auto
	case "zh", "zh-cn", "zh_cn", "zh-hans", "zh_hans", "schinese", "chinese":
		return ZhCN
	case "en", "en-us", "en_us", "english":
		return EN
	default:
		return Auto
	}
}

// DetectFromEnv returns the UI language inferred from the system locale. It's
// a thin wrapper over DetectSystemLocale; new code should prefer the latter
// when it needs region/script information too.
func DetectFromEnv() Language {
	return languageFromSystemLocale(DetectSystemLocale())
}

func languageFromSystemLocale(loc SystemLocale) Language {
	if loc.Language == "zh" {
		return ZhCN
	}
	return EN
}

// languageFromLocale parses a single locale string into a UI Language.
// Returns ok=false for empty / neutral input.
func languageFromLocale(value string) (Language, bool) {
	loc, ok := parseLocale(value)
	if !ok {
		return "", false
	}
	return languageFromSystemLocale(loc), true
}

// detectFromEnvValues short-circuits on the first non-neutral LC_ env var.
// Kept for the existing TestDetectFromEnvValuesFallsBackForNeutralLocale.
func detectFromEnvValues(getenv func(string) string) (Language, bool) {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if loc, ok := parseLocale(getenv(key)); ok {
			return languageFromSystemLocale(loc), true
		}
	}
	return "", false
}

// detectFromText pulls a Language out of a noisy multi-line string, e.g.
// AppleLanguages plist output or /etc/locale.conf contents.
func detectFromText(value string) (Language, bool) {
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, `()"',`)
		if eq := strings.IndexByte(line, '='); eq >= 0 {
			line = line[eq+1:]
			line = strings.Trim(line, `"' `)
		}
		if loc, ok := parseLocale(line); ok {
			return languageFromSystemLocale(loc), true
		}
	}
	return "", false
}

// isNeutralLocale recognizes locale identifiers that carry no useful
// language/region info ("C", "POSIX", "C.UTF-8").
func isNeutralLocale(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == "c" || value == "posix" || strings.HasPrefix(value, "c.")
}

// T looks up an i18n message key. Falls back to the English entry, then to
// the key itself if neither map has it. TestKeysetParity guards both maps.
func T(key string) string {
	if current == ZhCN {
		if value, ok := zhCN[key]; ok {
			return value
		}
	}
	if value, ok := en[key]; ok {
		return value
	}
	return key
}

var en = map[string]string{
	"root.short":                 "Query Steam public store data, prices, reviews, news, and sale events",
	"flag.cc":                    "country/region code for prices; \"auto\" detects from system locale (default), run steam-cli locales --type regions for explicit values",
	"flag.lang":                  "Steam content language; \"auto\" detects from system locale (default), run steam-cli locales --type languages for values",
	"flag.ui_lang":               "Steam CLI interface language: auto, en, zh-CN",
	"flag.timeout":               "request timeout in seconds",
	"flag.json":                  "print JSON envelope",
	"flag.quiet":                 "print only the most important fields for supported commands",
	"flag.no_color":              "disable ANSI color in terminal output",
	"flag.verbose":               "print retry and diagnostic messages to stderr",
	"flag.rate_ms":               "minimum milliseconds between requests from a single client; default 0 leaves throttling off",
	"flag.help":                  "help for this command",
	"flag.version":               "version for steam-cli",
	"search.short":               "Search Steam store games",
	"app.short":                  "Show public info for a Steam app",
	"price.short":                "Show price for a Steam app",
	"media.short":                "Show Steam app images, screenshots, trailers, and media assets",
	"dlc.short":                  "List DLC for a Steam app with current store data",
	"similar.short":              "Show Steam store recommendations similar to an app",
	"deals.short":                "Show Steam store lists such as specials, top sellers, new releases, and upcoming games",
	"reviews.short":              "Show Steam user reviews for an app",
	"news.short":                 "Show Steam news and announcements for an app",
	"achievements.short":         "Show global achievement unlock percentages for an app",
	"events.short":               "List recent and upcoming Steam sale events",
	"user.short":                 "Show public Steam Community profile information",
	"wishlist.short":             "Show a public Steam user's wishlist",
	"locales.short":              "List common --cc regions and --lang Steam languages",
	"doctor.short":               "Check Steam CLI network access, locale settings, and core data sources",
	"help.short":                 "Help about any command",
	"search.flag.count":          "number of results",
	"price.flag.compare":         "comma-separated country/region codes to compare, for example CN,US,JP",
	"app.flag.news":              "number of news items to include; 0 disables news fetch",
	"media.flag.probe":           "HEAD-probe fixed CDN assets and include HTTP status / availability",
	"similar.flag.count":         "number of similar games to return",
	"deals.flag.count":           "number of store list items to return",
	"deals.flag.filter":          "list filter: specials, topsellers, new, comingsoon",
	"reviews.flag.count":         "number of reviews to return",
	"reviews.flag.filter":        "review filter: recent, updated, all (use 'all' to get cumulative summary)",
	"reviews.flag.type":          "review type: all, positive, negative",
	"reviews.flag.purchase":      "purchase type: all, steam, non_steam_purchase",
	"reviews.flag.cursor":        "pagination cursor returned from a previous response",
	"news.flag.count":            "number of news items to return",
	"achievements.flag.count":    "number of achievements to return",
	"events.flag.past_days":      "include events that ended within this many days",
	"events.flag.future_days":    "include events starting within this many days",
	"wishlist.flag.count":        "number of wishlist items to display; 0 shows all",
	"wishlist.flag.offset":       "start offset in the wishlist",
	"wishlist.flag.no_details":   "skip appdetails lookups and show only wishlist appids",
	"locales.flag.type":          "which locale list to show: all, regions, languages",
	"locales.flag.live":          "fetch languages live from the Steam Store language menu",
	"locales.flag.probe":         "probe listed regions against Steam appdetails pricing",
	"locales.flag.appid":         "appid used by --probe",
	"doctor.title":               "Steam CLI doctor",
	"doctor.status.ok":           "ok",
	"doctor.status.failed":       "failed",
	"doctor.message.reachable":   "reachable",
	"doctor.header.check":        "Check",
	"doctor.header.status":       "Status",
	"doctor.header.http":         "HTTP",
	"doctor.header.message":      "Message",
	"label.cc":                   "CC",
	"label.lang":                 "Lang",
	"label.ui_lang":              "UI language",
	"label.observed":             "Observed",
	"table.appid":                "AppID",
	"table.name":                 "Name",
	"table.price":                "Price",
	"table.discount":             "Discount",
	"table.available":            "Available",
	"table.discount_ends":        "Discount Ends",
	"price.discount_ends":        "Discount ends: ",
	"price.observed_from":        "Observed at %s from %s",
	"hint.rate_limited":          "Wait and retry later, or lower request frequency.",
	"hint.access_denied":         "This source may require a key, login, or lower request frequency.",
	"hint.privacy_restricted":    "Steam CLI only reads public Steam data and does not bypass privacy settings.",
	"hint.not_found":             "Check the appid/user input, or use steam-cli search to discover appids.",
	"hint.invalid_appid":         "Use steam-cli search \"game name\" to find a numeric appid.",
	"hint.network_timeout":       "Increase --timeout or retry later.",
	"hint.source_changed":        "Steam may have changed the response shape; try again later or report the command and appid.",
	"hint.invalid_profile_input": "Use a SteamID64, custom URL name from steamcommunity.com/id/<name>, or a /profiles/... URL.",
	"help.usage":                 "Usage:",
	"help.aliases":               "Aliases:",
	"help.examples":              "Examples:",
	"help.available_commands":    "Available Commands:",
	"help.flags":                 "Flags:",
	"help.global_flags":          "Global Flags:",
	"help.more_info":             "Use \"%s [command] --help\" for more information about a command.",
}

var zhCN = map[string]string{
	"root.short":                 "查询 Steam 公开商店、价格、评价、新闻和活动数据",
	"flag.cc":                    "价格国家/地区代码;auto 表示从系统区域设置自动推断（默认），运行 steam-cli locales --type regions 查看常用值",
	"flag.lang":                  "Steam 内容语言;auto 表示从系统区域设置自动推断（默认），运行 steam-cli locales --type languages 查看可选值",
	"flag.ui_lang":               "Steam CLI 界面语言：auto、en、zh-CN",
	"flag.timeout":               "请求超时秒数",
	"flag.json":                  "输出统一 JSON envelope",
	"flag.quiet":                 "对支持的命令只输出最关键字段",
	"flag.no_color":              "关闭终端 ANSI 颜色",
	"flag.verbose":               "把重试和诊断信息输出到 stderr",
	"flag.rate_ms":               "单个 client 两次请求之间的最小毫秒间隔；默认 0 表示不节流",
	"flag.help":                  "查看该命令帮助",
	"flag.version":               "显示 steam-cli 版本",
	"search.short":               "搜索 Steam 商店游戏",
	"app.short":                  "查看 Steam 游戏公开信息",
	"price.short":                "查看 Steam 游戏价格",
	"media.short":                "查看 Steam 游戏图片、截图、视频和媒体资源",
	"dlc.short":                  "查看 Steam 游戏 DLC 和当前商店数据",
	"similar.short":              "查看相似/推荐游戏",
	"deals.short":                "查看 Steam 特惠、热销、新品和即将推出列表",
	"reviews.short":              "查看 Steam 用户评价",
	"news.short":                 "查看 Steam 游戏新闻和公告",
	"achievements.short":         "查看全局成就解锁率",
	"events.short":               "查看近期和未来 Steam 促销活动",
	"user.short":                 "查看公开 Steam Community 资料",
	"wishlist.short":             "查看公开 Steam 用户愿望单",
	"locales.short":              "列出常用 --cc 地区和 --lang Steam 语言",
	"doctor.short":               "检查 Steam CLI 网络、地区/语言和核心数据源",
	"help.short":                 "查看某个命令的帮助",
	"search.flag.count":          "结果数量",
	"price.flag.compare":         "要对比的国家/地区代码，用逗号分隔，例如 CN,US,JP",
	"app.flag.news":              "包含的新闻条数；0 表示不拉取新闻",
	"media.flag.probe":           "对每个 CDN 资源做 HEAD 探测，输出 HTTP 状态与可用性",
	"similar.flag.count":         "返回的相似游戏数量",
	"deals.flag.count":           "返回的条目数量",
	"deals.flag.filter":          "列表筛选：specials、topsellers、new、comingsoon",
	"reviews.flag.count":         "返回的评测数量",
	"reviews.flag.filter":        "评测过滤：recent、updated、all（用 all 才能拿到累积评分汇总）",
	"reviews.flag.type":          "评测类型：all、positive、negative",
	"reviews.flag.purchase":      "购买类型：all、steam、non_steam_purchase",
	"reviews.flag.cursor":        "上一次响应返回的分页 cursor",
	"news.flag.count":            "返回的新闻条数",
	"achievements.flag.count":    "返回的成就数量",
	"events.flag.past_days":      "包含最近多少天内已结束的活动",
	"events.flag.future_days":    "包含未来多少天内开始的活动",
	"wishlist.flag.count":        "显示的愿望单条目数；0 表示全部",
	"wishlist.flag.offset":       "愿望单起始偏移",
	"wishlist.flag.no_details":   "跳过 appdetails，只显示愿望单 appid",
	"locales.flag.type":          "要显示的列表：all、regions、languages",
	"locales.flag.live":          "从 Steam 商店语言菜单实时拉取语言列表",
	"locales.flag.probe":         "用 Steam appdetails 价格探测列表中的地区",
	"locales.flag.appid":         "--probe 使用的 appid",
	"doctor.title":               "Steam CLI 诊断",
	"doctor.status.ok":           "正常",
	"doctor.status.failed":       "失败",
	"doctor.message.reachable":   "可访问",
	"doctor.header.check":        "检查项",
	"doctor.header.status":       "状态",
	"doctor.header.http":         "HTTP",
	"doctor.header.message":      "信息",
	"label.cc":                   "地区",
	"label.lang":                 "Steam 语言",
	"label.ui_lang":              "界面语言",
	"label.observed":             "观测时间",
	"table.appid":                "AppID",
	"table.name":                 "名称",
	"table.price":                "价格",
	"table.discount":             "折扣",
	"table.available":            "可用",
	"table.discount_ends":        "折扣结束",
	"price.discount_ends":        "折扣结束：",
	"price.observed_from":        "观测时间 %s，来源 %s",
	"hint.rate_limited":          "请稍后重试，或降低请求频率。",
	"hint.access_denied":         "该数据源可能需要 key、登录，或需要降低请求频率。",
	"hint.privacy_restricted":    "Steam CLI 只读取公开 Steam 数据，不绕过隐私设置。",
	"hint.not_found":             "请检查 appid/用户输入，或使用 steam-cli search 查找 appid。",
	"hint.invalid_appid":         "请使用 steam-cli search \"游戏名\" 查找数字 appid。",
	"hint.network_timeout":       "请增大 --timeout 或稍后重试。",
	"hint.source_changed":        "Steam 可能调整了响应结构；请稍后重试，或反馈命令和 appid。",
	"hint.invalid_profile_input": "请使用 SteamID64、steamcommunity.com/id/<name> 里的自定义 URL 名称，或 /profiles/... URL。",
	"help.usage":                 "用法：",
	"help.aliases":               "别名：",
	"help.examples":              "示例：",
	"help.available_commands":    "可用命令：",
	"help.flags":                 "参数：",
	"help.global_flags":          "全局参数：",
	"help.more_info":             "使用 \"%s [command] --help\" 查看某个命令的更多信息。",
}
