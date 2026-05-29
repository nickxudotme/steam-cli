# Steam CLI

Steam CLI is a Go command-line tool for querying public Steam Store, Steam Community, Steam Web API, and Steamworks event data. It is built with [Cobra](https://github.com/spf13/cobra) for commands and [Charmbracelet Lip Gloss](https://github.com/charmbracelet/lipgloss) for polished terminal output.

The goal is to replace common Steam web lookups from the terminal while keeping the default data path public, live, and API-key-free.

[中文文档](README.zh-CN.md)

## Features

- App overview: price, regional pricing, discount, discount end time, reviews, tags, Steam Deck status, rating, official links, media, and news.
- Store search first, so users can find an appid from a game name before querying app-specific data.
- Price lookup with `--cc` and `--lang`.
- Regional price comparison with `price --compare CN,US,JP`.
- Optional IsThereAnyDeal enhancement for cross-store best deals, price history, Steam store lows, shop lookup, and price-history commands.
- `doctor` checks core Steam data sources and current locale/network settings.
- `--quiet` for script-friendly output and `--no-color` for plain terminals or CI logs.
- Bilingual CLI prompts with `--ui-lang auto|en|zh-CN`, defaulting from `LC_ALL`, `LC_MESSAGES`, or `LANG`.
- Command aliases such as `game`, `find`, `images`, `sales`, and `recommend`.
- Store purchase options: packages, bundles, original price, final price, discount percentage, and active discount end time.
- Media assets: header, capsules, library capsule, 2x images, library hero, screenshots, trailers, and achievement icons.
- DLC lookup with current store data.
- Similar game recommendations.
- Store search, deals/top sellers/new/upcoming lists, user reviews, news, and global achievement percentages.
- Recent and upcoming Steam events/fests from the official Steamworks events page.
- Public Steam Community profiles and public wishlists.
- Unified JSON envelope for every command.

## Install

```bash
go build -o steam-cli .
```

## Global Flags

```text
--cc        Country/region code for pricing. Default: auto (detected from system locale).
--lang      Steam language. Default: auto (detected from system locale).
--timeout   Request timeout in seconds. Default: 12
--rate-ms   Minimum milliseconds between requests from a single client. Default: 0 (no throttling).
--json      Print a unified JSON envelope
--quiet     Print only the most important fields for supported commands
--no-color  Disable ANSI color
--verbose   Print retry and diagnostic messages to stderr
--ui-lang   Steam CLI interface language. Default: auto
--itad-key  Advanced pricing API key. Default: STEAM_CLI_ITAD_KEY
```

Get an IsThereAnyDeal API key from <https://isthereanydeal.com/apps/> before using `--itad-key` or `STEAM_CLI_ITAD_KEY`.

`--cc auto` and `--lang auto` read your system locale, in order:

1. `LC_ALL` / `LC_MESSAGES` / `LC_MONETARY` / `LANG` environment variables.
2. `defaults read -g AppleLocale` (macOS) / `(Get-Culture).Name` (Windows) / `/etc/locale.conf` (Linux).
3. Conservative fallback: `--cc US` and `--lang english`.

So `LANG=zh_CN.UTF-8` becomes `--cc CN --lang schinese`; `LANG=ja_JP.UTF-8` becomes `--cc JP --lang japanese`; `LANG=es_MX.UTF-8` becomes `--cc MX --lang latam`. Run `steam-cli doctor` to see what was detected.

`UK` is normalized to `GB` for `--cc`. Common `--lang` aliases like `chinese`, `zh-cn`, `zh-tw`, `korean`, `pt-br` are mapped to their Steam codes (`schinese`, `schinese`, `tchinese`, `koreana`, `brazilian`).

Examples:

```bash
./steam-cli search "cyberpunk 2077"
./steam-cli price 264710                          # auto-detected region
./steam-cli price 264710 --cc US                  # force US pricing
./steam-cli price 264710 --compare CN,US,JP
./steam-cli price 264710 --enhanced --cc US
./steam-cli app 264710 --json
./steam-cli history 264710 --days 30
./steam-cli history 264710 --all-stores
./steam-cli doctor
./steam-cli --ui-lang zh-CN search "赛博朋克 2077"
./steam-cli locales
./steam-cli locales --type languages --live
```

`--ui-lang` controls Steam CLI's own prompts, help text, table headers, and human-readable JSON error hints. It is separate from `--lang`, which controls the Steam content language sent to Steam endpoints. With `--ui-lang auto`, Steam CLI first reads `LC_ALL`, `LC_MESSAGES`, and `LANG`. If those are neutral values such as `C.UTF-8`, it falls back to OS language settings: AppleLanguages on macOS, locale config files on Linux, and PowerShell culture on Windows.

## Commands

| Command | Description | Main data sources |
| --- | --- | --- |
| [`search TERM`](#search) | Search Steam Store games and discover appids | [`storesearch`][source-storesearch] |
| [`app APPID`](#app) | App overview, price, discount end time, reviews, tags, Deck status, rating, links, purchase options, media, news; `--enhanced` adds advanced pricing insights | [`appdetails`][source-appdetails], [`IStoreBrowseService/GetItems`][source-storebrowse], [`appreviews`][source-appreviews], [`GetNumberOfCurrentPlayers`][source-currentplayers], [`GetNewsForApp`][source-news], [advanced pricing lookup][source-itad-lookup] |
| [`price APPID`](#price) | Price, regional pricing, discount, discount end time; `--compare` compares regions; `--enhanced` adds advanced pricing insights | [`appdetails`][source-appdetails], [`IStoreBrowseService/GetItems`][source-storebrowse], [advanced pricing lookup][source-itad-lookup] |
| [`media APPID`](#media) | CDN assets, 2x images, screenshots, trailers, achievement icons; `--probe` checks HTTP status | [`appdetails`][source-appdetails], [`IStoreBrowseService/GetItems`][source-storebrowse], [Steam CDN][source-cdn] |
| [`dlc APPID`](#dlc) | DLC list with current price, discount, reviews, release date | [`appdetails.dlc`][source-appdetails] + batch [`IStoreBrowseService/GetItems`][source-storebrowse] |
| [`similar APPID`](#similar) | Similar/recommended games | [`IStoreQueryService/MoreLikeThis`][source-morelikethis] |
| [`locales`](#locales) | Common `--cc` regions, live Steam `--lang` language codes, optional region price probing | built-in reference list, [Steam Store language menu][source-store-home], [`appdetails`][source-appdetails] |
| [`deals`](#deals) | Specials, top sellers, new releases, upcoming games, discount end time | [`search/results`][source-search-results], [`IStoreBrowseService/GetItems`][source-storebrowse] |
| [`reviews APPID`](#reviews) | Review summary and review list with filters and cursor pagination | [`appreviews`][source-appreviews] |
| [`news APPID`](#news) | Steam news, announcements, community events | [`GetNewsForApp`][source-news] |
| [`achievements APPID`](#achievements) | Global achievement unlock percentages | [`GetGlobalAchievementPercentagesForApp`][source-achievements] |
| [`events`](#events) | Steam seasonal sales, fests, Next Fest, upcoming events | [Steamworks Upcoming Steam Events][source-steamworks-events] |
| [`history APPID`](#history) | Steam price history for an app, with optional all-store expansion | [advanced pricing history][source-itad-history] |
| [`user USER`](#user) | Public Steam Community profile | [Community profile XML][source-community-xml] |
| [`wishlist USER`](#wishlist) | Public wishlist with optional app details | [`IWishlistService/GetWishlist`][source-wishlist] + [`appdetails`][source-appdetails] |
| [`doctor`](#doctor) | Data-source reachability, current locale, timeout, and network checks | Steam Store, Steam Web API, Steam Community, Steamworks |

## Examples

<a id="search"></a>

### Search First

```bash
./steam-cli search "cyberpunk 2077" --count 5
```

Use the returned appid with the app-specific commands below.

<a id="price-compare"></a>

### Compare Regional Prices

```bash
./steam-cli price 264710 --compare CN,US,JP,GB
```

For scripts:

```bash
./steam-cli price 264710 --compare CN,US,JP --quiet
```

<a id="doctor"></a>

### Diagnose Setup

```bash
./steam-cli doctor
```

`doctor` checks whether Steam Store, appdetails, Steam Web API, Steam Community, and Steamworks event pages are reachable with the current timeout and locale settings.

<a id="app"></a>

### App Overview

```bash
./steam-cli app 264710 --news 1
```

Example output:

```text
Subnautica (264710)

┌──────┬──────────────┬──────┬─────────────┬────────────┬──────────┐
│ Type │ Release      │ Free │ Players Now │ Metacritic │ Deck     │
├──────┼──────────────┼──────┼─────────────┼────────────┼──────────┤
│ game │ Jan 23, 2018 │ no   │ 42863       │ 87         │ verified │
└──────┴──────────────┴──────┴─────────────┴────────────┴──────────┘

Price: 7.49 USD -75%  original 29.99 USD
Discount ends: 2026-05-26 01:00 UTC+08:00 (UTC 2026-05-25 17:00, PT 2026-05-25 10:00)

Store profile
Developers: Unknown Worlds Entertainment
Languages: 27 supported, 1 full audio, 27 subtitles
Store tags: 1100689(820), 1662(816), 1667(745), 9157(688), ...
Rating: ESRB - e10 - Fantasy Violence, Mild Language
```

`app` shows both StoreBrowse review data and `appreviews` data:

- `Store Reviews`: aggregated StoreBrowse review summary.
- `App Reviews`: review summary from the public app review endpoint.

The four sibling lookups (store item, reviews, current player count, news) run concurrently. If any of them fail, the bundle is still returned and the failures are listed under a `Warnings` section in terminal output and `data.warnings` in `--json`.

The four sibling lookups (store item, reviews, current player count, news) run concurrently. If any of them fail, the bundle is still returned and the failures are listed under a `Warnings` section in terminal output and `data.warnings` in `--json`.

<a id="price"></a>

### Price and Discount End Time

```bash
./steam-cli price 264710 --cc US
```

Example:

```text
Subnautica (264710)

Price: 7.49 USD -75%  original 29.99 USD
Discount ends: 2026-05-26 01:00 UTC+08:00 (UTC 2026-05-25 17:00, PT 2026-05-25 10:00)
```

The discount end time comes from `IStoreBrowseService/GetItems` field `active_discounts[].discount_end_date`. JSON keeps event times such as `discount_end` and `release_time` as Unix seconds. Steam's current public response does not expose an authoritative discount start time. A start time can only be inferred from related event windows or observed by tracking price changes over time.

### Advanced Price Enhancement

Advanced pricing uses IsThereAnyDeal. Create an API key at <https://isthereanydeal.com/apps/>, then pass it with `--itad-key` or `STEAM_CLI_ITAD_KEY`.

```bash
STEAM_CLI_ITAD_KEY=your_key ./steam-cli price 264710 --enhanced --cc US
STEAM_CLI_ITAD_KEY=your_key ./steam-cli app 264710 --enhanced --news 0
```

`--enhanced` adds optional advanced pricing data without changing the default Steam-only path. The current implementation adds:

- Cross-store best current deal
- Historical best-ever deal
- Multi-window historical low snapshots
- Steam store low from the third-party price history backend
- A price page URL and current deal URL

`--enhanced` currently does not combine with `price --compare`; use one or the other in the current release.

<a id="history"></a>

### Price History

```bash
STEAM_CLI_ITAD_KEY=your_key ./steam-cli history 264710
STEAM_CLI_ITAD_KEY=your_key ./steam-cli history 264710 --days 30
STEAM_CLI_ITAD_KEY=your_key ./steam-cli history 264710 --all-stores
STEAM_CLI_ITAD_KEY=your_key ./steam-cli history 264710 --sales
```

`history` stays Steam-first: by default it shows Steam store price changes for the app. Add `--all-stores` to expand the view to authorized non-Steam sellers handled by the pricing backend.

Use `--sales` when you want historical discount windows rather than raw price-change points. This view infers when a discounted phase started and when the next price change ended it.

In `--json` mode, both raw history entries and inferred sales windows also include precise RFC3339 timestamps and Unix timestamps for scripting.

<a id="media"></a>

### Media

```bash
./steam-cli media 264710
./steam-cli media 264710 --probe
```

Example:

```text
CDN assets
┌─────────┬────────────────────┬────────┬────────────────────────────────────────────────────────────┐
│ Kind    │ Name               │ Status │ URL                                                        │
├─────────┼────────────────────┼────────┼────────────────────────────────────────────────────────────┤
│ capsule │ main_capsule       │ -      │ https://cdn.akamai.steamstatic.com/steam/apps/...          │
│ library │ library_600x900    │ -      │ https://cdn.akamai.steamstatic.com/steam/apps/...          │
│ library │ library_600x900_2x │ -      │ https://cdn.akamai.steamstatic.com/steam/apps/...          │
│ library │ library_hero_2x    │ -      │ https://cdn.akamai.steamstatic.com/steam/apps/...          │
└─────────┴────────────────────┴────────┴────────────────────────────────────────────────────────────┘
```

`media` uses StoreBrowse `assets.asset_url_format` when available, so it can return assets such as `library_600x900_2x`, `library_hero_2x`, and `hero_capsule`. If StoreBrowse is unavailable, it falls back to common fixed CDN paths.

<a id="dlc"></a>

### DLC

```bash
./steam-cli dlc 264710
```

Example:

```text
Subnautica DLC (264710)

┌─────────┬────────────────────────────────┬────────────┬────────────┬───────────────────────────┐
│ AppID   │ Name                           │ Release    │ Price      │ Reviews                   │
├─────────┼────────────────────────────────┼────────────┼────────────┼───────────────────────────┤
│ 1619300 │ Subnautica Original Soundtrack │ 2021-05-01 │ $2.49 -75% │ Very Positive, 92% of 440 │
└─────────┴────────────────────────────────┴────────────┴────────────┴───────────────────────────┘
```

The public WebAPI form of `IStoreBrowseService/GetDLCForApps` currently returns `401` without a key, so the default implementation uses `appdetails.dlc` to discover DLC appids and batch `StoreBrowse.GetItems` to enrich them.

<a id="similar"></a>

### Similar Games

```bash
./steam-cli similar 264710 --count 5
```

Example:

```text
Similar to 264710

┌─────────┬────────────────────────┬────────────┬──────────────────────────────┬──────────────┐
│ AppID   │ Name                   │ Price      │ Reviews                      │ Platforms    │
├─────────┼────────────────────────┼────────────┼──────────────────────────────┼──────────────┤
│ 848450  │ Subnautica: Below Zero │ $7.49 -75% │ Very Positive, 90% of 95999  │ windows, mac │
│ 1962700 │ Subnautica 2           │ $29.99     │ Very Positive, 91% of 77416  │ windows      │
└─────────┴────────────────────────┴────────────┴──────────────────────────────┴──────────────┘
```

<a id="deals"></a>

### Deals

```bash
./steam-cli deals --filter specials --count 10
./steam-cli deals --filter topsellers --any discounted,preorder --count 10
./steam-cli deals --filter discountedtopsellers --count 10
./steam-cli deals --filter preordertopsellers --count 10
./steam-cli deals --filter topsellers --count 10
```

Use `--any discounted,preorder` to keep the selected store list order while showing games that match at least one condition. For example, `--filter topsellers --any discounted,preorder --count 10` shows top sellers that are either currently discounted or available for pre-order. JSON output uses Unix seconds for event times such as `release_time` and `discount_end`.

<a id="reviews"></a>

### Reviews

```bash
./steam-cli reviews 1245620 --type negative --count 5
./steam-cli reviews 1245620 --filter all --count 5
```

`--filter recent` (the default) reflects only recent reviews after the type/purchase filter — its `query_summary` will show 0 totals when the filtered window is empty. Pass `--filter all` to get the cumulative review score and totals.

<a id="news"></a>

### News

```bash
./steam-cli news 1245620 --count 10
```

<a id="achievements"></a>

### Achievements

```bash
./steam-cli achievements 1245620 --count 20
```

<a id="locales"></a>

### Region and Language Hints

```bash
./steam-cli locales
./steam-cli locales --type regions
./steam-cli locales --type languages
./steam-cli locales --type languages --live
./steam-cli locales --type regions --probe --appid 264710
```

`--cc` uses country/region codes such as `CN`, `US`, `JP`, `GB`, `DE`, and `BR`. Steam generally accepts ISO 3166-1 alpha-2 country codes when that store region is supported.

`--lang` uses Steam language codes such as `english`, `schinese`, `tchinese`, `japanese`, `koreana`, `brazilian`, and `latam`.

Use `--live` with `--type languages` to parse the current Steam Store language menu from `https://store.steampowered.com/`. Use `--probe` with `--type regions` to test the listed regions against Steam `appdetails` for a real app. Region probing is observed availability, not an official exhaustive registry.

<a id="user"></a>

### Users

```bash
./steam-cli user nickxudotme
./steam-cli user "https://steamcommunity.com/profiles/76561198115468824/"
```

<a id="wishlist"></a>

### Wishlists

```bash
./steam-cli wishlist 76561198115468824 --count 20
./steam-cli wishlist 76561198115468824 --count 20 --no-details
```

User data is read only from public profile XML. Steam CLI does not bypass privacy settings. Wishlists must be public.

<a id="events"></a>

### Steam Events

```bash
./steam-cli events
./steam-cli events --past-days 15 --future-days 90
```

The events command parses the official Steamworks Upcoming Steam Events page plus public Steam Store sale pages (for example Lunar New Year pages) and returns event name, date range, status, type, PT timezone marker, description/eligibility notes, registration link, and more-info link. When a Store sale page exposes embedded event metadata, JSON also includes exact Unix `start_time`/`end_time`, `store_url`, event/group ids, announcement link, and image assets such as `image_url`, `title_image_url`, `capsule_image_url`, and `background_image_url`. Use `--store-sales=false` to show only the Steamworks event calendar.
By default it uses a broad window (`--past-days 365 --future-days 365`) so annual sale pages remain visible after they end; narrow the window with flags when you only want near-term events.

The terminal table shows the 6 most useful columns (Start, End, Status, Category, Event, Description). Use `--json` to see the full payload including `registration_url`, `info_url`, `store_url`, `notes`, precise times, and image URLs.

## JSON Output

All commands support `--json`. Successful response:

```json
{
  "ok": true,
  "command": "steam-cli price",
  "schema": "steam-cli.price.v1",
  "generated_at": "2026-05-22T10:00:00Z",
  "meta": {
    "version": "0.1.0",
    "cc": "US",
    "lang": "english",
    "timeout": 12,
    "quiet": false,
    "no_color": false,
    "verbose": false,
    "observed_at": "2026-05-22T10:00:00Z",
    "sources": [
      {
        "name": "appdetails",
        "type": "official_store_api",
        "confidence": "high"
      }
    ]
  },
  "data": {
    "appid": 264710,
    "name": "Subnautica",
    "is_free": false,
    "price_overview": {
      "currency": "USD",
      "initial": 2999,
      "final": 749,
      "discount_percent": 75
    },
    "store_item": {
      "best_purchase_option": {
        "packageid": 575971,
        "discount_pct": 75,
        "active_discounts": [
          {
            "discount_end_date": 1779728400
          }
        ]
      }
    }
  }
}
```

Error response:

```json
{
  "ok": false,
  "command": "steam-cli price",
  "schema": "steam-cli.price.v1",
  "generated_at": "2026-05-22T10:00:00Z",
  "meta": {
    "version": "0.1.0",
    "cc": "CN",
    "lang": "schinese",
    "timeout": 12,
    "quiet": false,
    "no_color": false,
    "observed_at": "2026-05-22T10:00:00Z"
  },
  "error": {
    "type": "invalid_input",
    "message": "invalid appid \"not-an-app\"",
    "hint": "Use steam-cli search \"game name\" to find a numeric appid."
  }
}
```

New commands only need to use the internal `runCommand(load, render)` wrapper. JSON success/error envelopes, observed time, command sources, and error classification are handled globally.

`error.type` is a stable enum from `internal/steam`:

| `error.type` | Triggered by |
| --- | --- |
| `invalid_input` | Non-numeric appid, malformed profile URL, etc. |
| `not_found` | Steam returned `success:false` or no public data for the input. |
| `rate_limited` | HTTP 429 (after retries). |
| `access_denied` | HTTP 401 / 403. |
| `privacy_restricted` | Wishlist or profile is private / friends-only. |
| `network_timeout` | Request exceeded `--timeout` or a context deadline. |
| `source_changed` | Steam returned an unexpected response shape. |
| `unknown` | Anything not matched above. |

`error.hint` is localized by `--ui-lang`.

## Data Sources

Steam CLI uses public, live sources by default and does not require a Steam API key.

Optional advanced pricing commands and `--enhanced` features require a third-party pricing API key via `--itad-key` or `STEAM_CLI_ITAD_KEY`.
Create one at <https://isthereanydeal.com/apps/>.

| Source | Used for |
| --- | --- |
| [`https://store.steampowered.com/api/appdetails`][source-appdetails] | Basic app details, regional price, DLC appids, screenshots, movies, language HTML, categories, developers/publishers |
| [`https://api.steampowered.com/IStoreBrowseService/GetItems/v1/`][source-storebrowse] | Purchase options, package/bundle price, active discount end time, Store reviews, tags, Deck categories, ratings, official links, structured languages, media assets |
| [`https://api.steampowered.com/IStoreQueryService/MoreLikeThis/v1/`][source-morelikethis] | Similar game recommendations |
| [`https://store.steampowered.com/appreviews/{appid}`][source-appreviews] | Review summaries and review list |
| [`https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/`][source-currentplayers] | Current player count |
| [`https://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/`][source-achievements] | Global achievement unlock percentages |
| [`https://api.steampowered.com/ISteamNews/GetNewsForApp/v2/`][source-news] | News, announcements, community events |
| [`https://store.steampowered.com/api/storesearch`][source-storesearch] | Store search |
| [`https://store.steampowered.com/search/results/`][source-search-results] | Specials, top sellers, new releases, upcoming games |
| [Official upcoming Steam events/fests][source-steamworks-events] | Official upcoming Steam events/fests |
| [`https://store.steampowered.com/sale/{vanity}`][source-store-sale-pages] | Public Steam Store sale pages, including third-party and regional sale events |
| [`https://steamcommunity.com/id/{custom_url}/?xml=1`][source-community-id-xml] | Public profile XML |
| [`https://steamcommunity.com/profiles/{steamid64}/?xml=1`][source-community-xml] | Public profile XML |
| [`https://api.steampowered.com/IWishlistService/GetWishlist/v1/`][source-wishlist] | Public wishlist appids, priority, date added |
| [Steam CDN][source-cdn], e.g. `https://cdn.akamai.steamstatic.com/steam/apps/{appid}/...` | App image assets |
| [`https://api.isthereanydeal.com/games/lookup/v1`][source-itad-lookup] | Map Steam appids to the advanced pricing backend's game IDs |
| [`https://api.isthereanydeal.com/games/overview/v2`][source-itad-overview] | Advanced best-deal snapshot, bundle references, and cross-store comparisons |
| [`https://api.isthereanydeal.com/games/history/v2`][source-itad-history] | Advanced price-change history log |
| [`https://api.isthereanydeal.com/games/historylow/v1`][source-itad-historylow] | Multi-window historical low snapshots |
| [`https://api.isthereanydeal.com/games/storelow/v2`][source-itad-storelow] | Store-specific historical low, such as Steam store low |
| [`https://api.isthereanydeal.com/service/shops/v1`][source-itad-shops] | Internal shop mapping used by the advanced pricing backend |

## Notes from SteamKit and SteamCMD

Steam CLI cross-checks implementation details against [SteamKit](https://github.com/SteamRE/SteamKit) protocol definitions and [SteamCMD](https://developer.valvesoftware.com/wiki/SteamCMD) behavior:

- SteamKit's generated `StoreBrowse.GetItems#1` model exposes `PurchaseOption.active_discounts[].discount_end_date`, which explains where a live discount end time can come from.
- StoreBrowse also exposes tags, reviews, assets, ratings, links, supported languages, and Steam Deck categories. These are integrated into the default app model and terminal output.
- SteamCMD/PICS can expose deeper appinfo such as depots, branches, manifest gids, launch configuration, Steam Cloud/UFS rules, Deck test details, and localized library assets.

SteamCMD is not used in the default runtime path because it is an external binary and Steam network dependency. A future `appinfo APPID --steamcmd` command could expose depot/branch/manifest technical metadata when SteamCMD is installed locally.

## Cache and Rate Limits

Caching is in-memory only. Steam CLI does not write persistent cache files. The cache is shared across `Client` instances in one process so multi-region price comparison reuses entries.

- App details: appid + `--cc` + `--lang`, 10 minutes.
- Store browse item: appid + `--cc` + `--lang`, 10 minutes. Prevents `app`, `price`, `media`, and `dlc` from each fetching the same item.
- Public profile: profile path, 10 minutes.
- Wishlist appid list: SteamID64, 5 minutes.

`--rate-ms N` enables a per-client minimum interval between requests. By default Steam CLI does not throttle and instead relies on the cache plus retry on 429.

HTTP retries cover `429`, `502`, `503`, and `504`. `Retry-After` is honored when present (capped at 30 seconds — anything longer aborts immediately rather than hanging the CLI). Exponential backoff is used otherwise. Network-level errors retry through the same loop.

In non-JSON mode, waits of 2 seconds or longer print a short retry notice to `stderr` so the CLI does not look stuck. Use `--verbose` to print every retry notice. JSON mode keeps `stdout` as pure JSON and suppresses these retry notices.

Observed headers:

- `IWishlistService/GetWishlist/v1`: normal `200` includes `x-eresult: 1`; no `Retry-After` or `X-RateLimit-*`.
- `appdetails`: normal `200` includes `Cache-Control: public,max-age=3600`, `Expires`, `Last-Modified`; no remaining quota header.
- `appreviews`: normal `200` includes `Cache-Control: private,max-age=600` and Steam cookies; no remaining quota header.

One conservative probe observed `HTTP 429` after about 7 wishlist requests to the same endpoint over roughly 14 seconds from the same IP. This is only an observation, not an official threshold.

Official public notes:

- Steam Web API Terms of Use state a daily Steam Web API limit of 100,000 calls.
- [Steamworks Web API Overview][source-webapi-overview] notes strict IP rate limiting for requests that generate `403`.
- Valve does not publish exact per-minute, per-IP, per-endpoint 429 thresholds.

## Testing

```bash
go vet ./...
go test -race ./...
go build -o steam-cli .
```

CI runs `go vet`, `go test -race`, and `go build` on every push (`.github/workflows/ci.yml`).

Unit tests cover:

- Profile URL / custom URL name / SteamID64 parsing.
- `Retry-After` seconds and HTTP date parsing.
- Store CDN asset URL construction (absolute, protocol-relative, format placeholders).
- Store search result HTML parsing.
- Flexible JSON parsing for Steam fields.
- Terminal output helpers for Store reviews, platforms, language summaries, and purchase option IDs.
- HTTP retry policy via `httptest`: 429 + `Retry-After`, 5xx retry, retry-after cap, source-changed JSON, exhausted retries, `--rate-ms` throttle.
- Typed error classification (`steam.CodeOf`, `errors.As`).
- i18n key parity across `en` and `zh-CN` maps.

Live commands verified during development:

```bash
./steam-cli price 264710 --cc US
./steam-cli app 264710 --news 1
./steam-cli media 264710
./steam-cli dlc 264710
./steam-cli similar 264710 --count 5
```

## Limitations

- Active discount end time is available from StoreBrowse. Authoritative discount start time is not exposed in the current public response.
- Steam user fuzzy search by nickname is not integrated because there is no stable public unauthenticated interface.
- Private/friends-only profiles, wishlists, inventories, and libraries are not bypassed.
- StoreBrowse and StoreQuery are publicly reachable but not formally documented stable APIs. Some commands fall back to older Store endpoints where possible.

[source-appdetails]: https://store.steampowered.com/api/appdetails?appids=264710&cc=US&l=english
[source-storebrowse]: https://api.steampowered.com/IStoreBrowseService/GetItems/v1/
[source-morelikethis]: https://api.steampowered.com/IStoreQueryService/MoreLikeThis/v1/
[source-appreviews]: https://store.steampowered.com/appreviews/264710?json=1
[source-currentplayers]: https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/?appid=264710
[source-achievements]: https://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/?gameid=264710
[source-news]: https://api.steampowered.com/ISteamNews/GetNewsForApp/v2/?appid=264710&count=3&format=json
[source-storesearch]: https://store.steampowered.com/api/storesearch?term=subnautica&cc=US&l=english
[source-search-results]: https://store.steampowered.com/search/results/?query=&start=0&count=10&dynamic_data=&infinite=1&specials=1&cc=US&l=english
[source-steamworks-events]: https://partner.steamgames.com/doc/marketing/upcoming_events?l=english
[source-store-sale-pages]: https://store.steampowered.com/sale/lny2026
[source-webapi-overview]: https://partner.steamgames.com/doc/webapi_overview
[source-community-id-xml]: https://steamcommunity.com/id/nickxudotme/?xml=1
[source-community-xml]: https://steamcommunity.com/profiles/76561198115468824/?xml=1
[source-wishlist]: https://api.steampowered.com/IWishlistService/GetWishlist/v1/?steamid=76561198115468824
[source-cdn]: https://cdn.akamai.steamstatic.com/steam/apps/264710/library_600x900.jpg
[source-store-home]: https://store.steampowered.com/?l=english
[source-itad-lookup]: https://api.isthereanydeal.com/games/lookup/v1
[source-itad-overview]: https://api.isthereanydeal.com/games/overview/v2
[source-itad-history]: https://api.isthereanydeal.com/games/history/v2
[source-itad-historylow]: https://api.isthereanydeal.com/games/historylow/v1
[source-itad-storelow]: https://api.isthereanydeal.com/games/storelow/v2
[source-itad-shops]: https://api.isthereanydeal.com/service/shops/v1
