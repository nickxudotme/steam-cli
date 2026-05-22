# Steam CLI

[English](README.md)

Steam CLI 是一个用 Go 编写的 Steam 信息查询命令行工具。项目使用 [Cobra](https://github.com/spf13/cobra) 组织命令，用 [Charmbracelet Lip Gloss](https://github.com/charmbracelet/lipgloss) 渲染表格和分区输出。

目标是在终端里替代常见 Steam Web 查询，同时保持默认数据路径实时、公开、无需 Steam API Key。

## 功能

- 查询游戏综合信息、区域价格、当前折扣、当前折扣结束时间。
- 优先提供商店搜索，用户可以先通过游戏名找到 appid，再查询具体游戏数据。
- 使用 `price --compare CN,US,JP` 对比多个地区价格。
- 使用 `doctor` 检查核心 Steam 数据源、当前地区/语言和网络连通性。
- 支持 `--quiet` 输出脚本友好的精简结果，支持 `--no-color` 关闭 ANSI 颜色。
- 支持 `--ui-lang auto|en|zh-CN` 中英双语界面，默认从 `LC_ALL`、`LC_MESSAGES`、`LANG` 推断。
- 支持 `game`、`find`、`images`、`sales`、`recommend` 等常用别名。
- 查看 StoreBrowse 商店聚合信息：Steam Deck 兼容等级、商店评价、标签权重、分级、官方链接、结构化语言支持。
- 查看购买选项：package、bundle、原价、现价、折扣、折扣结束时间。
- 查看图片资源：header、capsule、library capsule、2x 图、library hero、截图、视频封面、成就图标。
- 查询 DLC、相似游戏、商店搜索、特惠/榜单、用户评价、新闻、全局成就。
- 查询 Steam 官方近期/未来活动和主题节。
- 查询公开 Steam Community 资料和公开愿望单。
- 所有命令支持统一 JSON envelope。

## 构建

```bash
go build -o steam-cli .
```

## 全局参数

```text
--cc        国家/地区价格区,默认 auto(根据系统区域设置自动推断)。
--lang      Steam 语言,默认 auto(根据系统区域设置自动推断)。
--timeout   请求超时秒数,默认 12
--rate-ms   单个 client 两次请求之间的最小毫秒间隔。默认 0,即不主动节流。
--json      输出统一 JSON envelope
--quiet     对支持的命令只输出最关键字段
--no-color  关闭 ANSI 颜色
--verbose   把重试和诊断信息输出到 stderr
--ui-lang   Steam CLI 界面语言,默认 auto
```

`--cc auto` 与 `--lang auto` 按以下顺序读取系统区域:

1. `LC_ALL` / `LC_MESSAGES` / `LC_MONETARY` / `LANG` 环境变量。
2. macOS 的 `defaults read -g AppleLocale` / Windows 的 `(Get-Culture).Name` / Linux 的 `/etc/locale.conf`。
3. 都失败时退回 `--cc US` 和 `--lang english`。

例如 `LANG=zh_CN.UTF-8` 会得到 `--cc CN --lang schinese`;`LANG=ja_JP.UTF-8` 得到 `--cc JP --lang japanese`;`LANG=es_MX.UTF-8` 得到 `--cc MX --lang latam`。运行 `steam-cli doctor` 可以看到自动推断出的实际值。

`--cc UK` 会自动规范化为 `GB`。`--lang` 也会自动映射常用别名:`chinese`、`zh-cn` → `schinese`,`zh-tw` → `tchinese`,`korean` → `koreana`,`pt-br` → `brazilian`。

示例：

```bash
./steam-cli search "cyberpunk 2077"
./steam-cli price 264710                          # 自动按系统区域查价
./steam-cli price 264710 --cc US                  # 强制按美区查价
./steam-cli price 264710 --compare CN,US,JP
./steam-cli app 264710 --json
./steam-cli doctor
./steam-cli --ui-lang zh-CN search "赛博朋克 2077"
./steam-cli locales
./steam-cli locales --type languages --live
```

`--ui-lang` 控制 Steam CLI 自己的提示、help、表头和 JSON 错误 hint。它和 `--lang` 分离：`--lang` 是传给 Steam 接口的内容语言。`--ui-lang auto` 会先读取 `LC_ALL`、`LC_MESSAGES`、`LANG`；如果它们是 `C.UTF-8` 这类中性值，会继续读取操作系统语言：macOS 使用 AppleLanguages，Linux 读取常见 locale 配置文件，Windows 使用 PowerShell culture。

## 命令

| 命令 | 用途 | 主要数据源 |
| --- | --- | --- |
| [`search TERM`](#search) | 搜索 Steam 商店游戏，并找到 appid | [`storesearch`][source-storesearch] |
| [`app APPID`](#app) | 综合游戏信息、价格、折扣结束时间、评价、标签、Deck、分级、链接、购买选项、媒体、新闻 | [`appdetails`][source-appdetails]、[`IStoreBrowseService/GetItems`][source-storebrowse]、[`appreviews`][source-appreviews]、[`GetNumberOfCurrentPlayers`][source-currentplayers]、[`GetNewsForApp`][source-news] |
| [`price APPID`](#price) | 价格、折扣、区域价格、折扣结束时间；`--compare` 可对比多个地区 | [`appdetails`][source-appdetails]、[`IStoreBrowseService/GetItems`][source-storebrowse] |
| [`media APPID`](#media) | CDN 图片、2x 图、截图、视频封面、成就图标；`--probe` 可探测 HTTP 状态 | [`appdetails`][source-appdetails]、[`IStoreBrowseService/GetItems`][source-storebrowse]、[Steam CDN][source-cdn] |
| [`dlc APPID`](#dlc) | DLC 列表，附带价格、折扣、评价、发行日期 | [`appdetails.dlc`][source-appdetails] + 批量 [`IStoreBrowseService/GetItems`][source-storebrowse] |
| [`similar APPID`](#similar) | 相似/推荐游戏 | [`IStoreQueryService/MoreLikeThis`][source-morelikethis] |
| [`locales`](#locales) | 常用 `--cc` 地区代码、实时 Steam `--lang` 语言代码、可选地区价格探测 | 内置参考列表、[Steam Store 语言菜单][source-store-home]、[`appdetails`][source-appdetails] |
| [`deals`](#deals) | 特惠、热销、新品、即将推出列表 | [`search/results`][source-search-results] |
| [`reviews APPID`](#reviews) | 用户评价摘要和评论列表，支持正负评、购买类型、cursor | [`appreviews`][source-appreviews] |
| [`news APPID`](#news) | 游戏新闻、公告、社区活动 | [`GetNewsForApp`][source-news] |
| [`achievements APPID`](#achievements) | 全局成就解锁率 | [`GetGlobalAchievementPercentagesForApp`][source-achievements] |
| [`events`](#events) | Steam 官方活动、促销节、Next Fest、主题节 | [Steamworks Upcoming Steam Events][source-steamworks-events] |
| [`user USER`](#user) | 公开 Steam Community profile | [Community profile XML][source-community-xml] |
| [`wishlist USER`](#wishlist) | 公开愿望单，按地区/语言补齐详情 | [`IWishlistService/GetWishlist`][source-wishlist] + [`appdetails`][source-appdetails] |
| [`doctor`](#doctor) | 检查数据源连通性、当前地区/语言、超时和网络状态 | Steam Store、Steam Web API、Steam Community、Steamworks |

## 示例

<a id="search"></a>

### 先搜索

```bash
./steam-cli search "cyberpunk 2077" --count 5
```

拿到返回的 appid 后，再使用下面这些针对具体游戏的命令。

<a id="price-compare"></a>

### 对比地区价格

```bash
./steam-cli price 264710 --compare CN,US,JP,GB
```

脚本中可以使用：

```bash
./steam-cli price 264710 --compare CN,US,JP --quiet
```

<a id="doctor"></a>

### 诊断环境

```bash
./steam-cli doctor
```

`doctor` 会检查 Steam Store、appdetails、Steam Web API、Steam Community 和 Steamworks 活动页面在当前超时、地区和语言配置下是否可访问。

<a id="app"></a>

### 综合信息

```bash
./steam-cli app 264710 --news 1
```

示例输出：

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

`app` 会同时展示两类评价：

- `Store Reviews`：StoreBrowse 的商店聚合评价。
- `App Reviews`：`appreviews` 接口返回的评价摘要。

四个并列子查询(store item、reviews、当前在线、新闻)是并发执行的。任一失败时,主数据仍会返回,失败信息会出现在终端的 `Warnings` 段以及 `--json` 的 `data.warnings` 字段。

<a id="price"></a>

### 价格和折扣结束时间

```bash
./steam-cli price 264710 --cc US
```

示例：

```text
Subnautica (264710)

Price: 7.49 USD -75%  original 29.99 USD
Discount ends: 2026-05-26 01:00 UTC+08:00 (UTC 2026-05-25 17:00, PT 2026-05-25 10:00)
```

折扣结束时间来自 `IStoreBrowseService/GetItems` 的 `active_discounts[].discount_end_date`。Steam 当前公开响应没有权威的折扣开始时间；如果需要开始时间，只能用相关活动窗口推断，或自己长期观测价格变化。

<a id="media"></a>

### 图片和媒体

```bash
./steam-cli media 264710
./steam-cli media 264710 --probe
```

`media` 优先使用 StoreBrowse 的 `assets.asset_url_format` 拼接真实资源，可以返回 `library_600x900_2x`、`library_hero_2x`、`hero_capsule` 等。StoreBrowse 不可用时会退回常见固定 CDN 路径。

<a id="dlc"></a>

### DLC

```bash
./steam-cli dlc 264710
```

示例：

```text
Subnautica DLC (264710)

┌─────────┬────────────────────────────────┬────────────┬────────────┬───────────────────────────┐
│ AppID   │ Name                           │ Release    │ Price      │ Reviews                   │
├─────────┼────────────────────────────────┼────────────┼────────────┼───────────────────────────┤
│ 1619300 │ Subnautica Original Soundtrack │ 2021-05-01 │ $2.49 -75% │ Very Positive, 92% of 440 │
└─────────┴────────────────────────────────┴────────────┴────────────┴───────────────────────────┘
```

已验证 `IStoreBrowseService/GetDLCForApps` 当前公开 WebAPI 会返回 401 并要求 `key=`，所以默认实现不使用它。当前实现用 `appdetails.dlc` 拿 DLC appid，再用 `StoreBrowse.GetItems` 批量补详情。

<a id="similar"></a>

### 相似游戏

```bash
./steam-cli similar 264710 --count 5
```

示例：

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

### 特惠和榜单

```bash
./steam-cli deals --filter specials --count 10
./steam-cli deals --filter topsellers --count 10
```

<a id="reviews"></a>

### 评论

```bash
./steam-cli reviews 1245620 --type negative --count 5
./steam-cli reviews 1245620 --filter all --count 5
```

`--filter recent`(默认值)只反映在 type/purchase 过滤后的最近评论;若过滤后为空,`query_summary` 的 Total/Positive/Negative 都会是 0。需要累积评分汇总时请用 `--filter all`。

<a id="news"></a>

### 新闻

```bash
./steam-cli news 1245620 --count 10
```

<a id="achievements"></a>

### 成就

```bash
./steam-cli achievements 1245620 --count 20
```

<a id="locales"></a>

### 地区和语言提示

```bash
./steam-cli locales
./steam-cli locales --type regions
./steam-cli locales --type languages
./steam-cli locales --type languages --live
./steam-cli locales --type regions --probe --appid 264710
```

`--cc` 使用国家/地区代码，例如 `CN`、`US`、`JP`、`GB`、`DE`、`BR`。Steam 通常接受 ISO 3166-1 alpha-2 国家/地区代码，前提是 Steam 支持该商店区域。

`--lang` 使用 Steam 语言代码，例如 `english`、`schinese`、`tchinese`、`japanese`、`koreana`、`brazilian`、`latam`。

`--type languages --live` 会从 `https://store.steampowered.com/` 当前语言菜单实时解析可选 Steam 语言。`--type regions --probe` 会用真实 app 请求 Steam `appdetails` 来探测地区价格是否可用。地区探测是观测结果，不是官方完整注册表。

<a id="user"></a>

### 用户

```bash
./steam-cli user nickxudotme
./steam-cli user "https://steamcommunity.com/profiles/76561198115468824/"
```

<a id="wishlist"></a>

### 愿望单

```bash
./steam-cli wishlist 76561198115468824 --count 20
./steam-cli wishlist 76561198115468824 --count 20 --no-details
```

用户资料只读取公开 profile XML，不绕过隐私设置。愿望单也只支持公开愿望单。

<a id="events"></a>

### 活动/节日

```bash
./steam-cli events
./steam-cli events --past-days 15 --future-days 90
```

活动数据来自 Steamworks 官方 Upcoming Steam Events 页面，会解析活动名称、日期范围、状态、类型、PT 时区标记、简介/资格说明、注册链接和详情链接。

终端表只展示 6 列(Start、End、Status、Category、Event、Description)。`--json` 会返回完整字段,包括 `registration_url`、`info_url`、`notes`、`timezone`。

## JSON 输出

所有命令都支持 `--json`。成功响应：

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

错误响应：

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

新增命令只需要使用内部 `runCommand(load, render)` 包装器。JSON 成功/错误 envelope、观测时间、命令数据源和错误分类都由全局层统一处理。

`error.type` 是 `internal/steam` 暴露的稳定枚举:

| `error.type` | 触发场景 |
| --- | --- |
| `invalid_input` | appid 不是数字、profile URL 格式错误等 |
| `not_found` | Steam 返回 `success:false` 或没有公开数据 |
| `rate_limited` | HTTP 429(重试用尽后) |
| `access_denied` | HTTP 401 / 403 |
| `privacy_restricted` | 愿望单或 profile 是私密/仅好友可见 |
| `network_timeout` | 超过 `--timeout` 或 context deadline |
| `source_changed` | Steam 返回结构与预期不符,常见于 Valve 临时调整接口 |
| `unknown` | 上述都未命中 |

`error.hint` 会跟随 `--ui-lang` 本地化。

## 数据源

Steam CLI 默认使用公开实时数据源，不需要 Steam API Key。

| 数据源 | 用途 |
| --- | --- |
| [`https://store.steampowered.com/api/appdetails`][source-appdetails] | 基础游戏详情、区域价格、DLC appid、截图、视频、语言 HTML、分类、开发商/发行商 |
| [`https://api.steampowered.com/IStoreBrowseService/GetItems/v1/`][source-storebrowse] | 购买选项、package/bundle 价格、当前折扣结束时间、商店评价、标签、Deck 等级、分级、官方链接、结构化语言、媒体 assets |
| [`https://api.steampowered.com/IStoreQueryService/MoreLikeThis/v1/`][source-morelikethis] | 相似游戏推荐 |
| [`https://store.steampowered.com/appreviews/{appid}`][source-appreviews] | 评价摘要和评论列表 |
| [`https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/`][source-currentplayers] | 当前在线人数 |
| [`https://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/`][source-achievements] | 全局成就解锁率 |
| [`https://api.steampowered.com/ISteamNews/GetNewsForApp/v2/`][source-news] | 新闻、公告、社区活动 |
| [`https://store.steampowered.com/api/storesearch`][source-storesearch] | 商店搜索 |
| [`https://store.steampowered.com/search/results/`][source-search-results] | 特惠、热销、新品、即将推出 |
| [官方 Steam 活动/促销节页面][source-steamworks-events] | 官方 Steam 活动/促销节 |
| [`https://steamcommunity.com/id/{custom_url}/?xml=1`][source-community-id-xml] | 公开 profile XML |
| [`https://steamcommunity.com/profiles/{steamid64}/?xml=1`][source-community-xml] | 公开 profile XML |
| [`https://api.steampowered.com/IWishlistService/GetWishlist/v1/`][source-wishlist] | 公开愿望单 appid、优先级、加入时间 |
| [Steam CDN][source-cdn]，例如 `https://cdn.akamai.steamstatic.com/steam/apps/{appid}/...` | 游戏图片资源 |

## SteamKit / SteamCMD 研究结论

本项目对照了 [SteamKit](https://github.com/SteamRE/SteamKit) 协议定义和 [SteamCMD](https://developer.valvesoftware.com/wiki/SteamCMD) 行为：

- SteamKit 生成的 `StoreBrowse.GetItems#1` 模型暴露 `PurchaseOption.active_discounts[].discount_end_date`，可以解释实时折扣结束时间的来源之一。
- StoreBrowse 还提供 tags、reviews、assets、ratings、links、supported languages、Steam Deck categories，这些已整合到默认模型和终端输出。
- SteamCMD/PICS 能拿到更底层的 appinfo，例如 depot、branch、manifest gid、launch 配置、Steam Cloud/UFS、Deck 测试细节、本地化库图资源等。

默认运行路径没有集成 SteamCMD，因为它是外部二进制和 Steam network 依赖。后续可以单独增加 `appinfo APPID --steamcmd`，在本机安装 SteamCMD 时输出 depot/branch/manifest 技术元数据。

## 缓存和限流

缓存只在进程内内存中，不会持久化。同一进程的多个 `Client`(例如多区域价格对比)共享缓存,所以重复 `appid` 不会重复打 Steam:

- App details：appid + `--cc` + `--lang`，10 分钟。
- Store browse item: appid + `--cc` + `--lang`,10 分钟。让 `app`、`price`、`media`、`dlc` 不会各自重复拉取同一份 store item。
- Public profile：profile 路径，10 分钟。
- Wishlist appid 列表：SteamID64，5 分钟。

`--rate-ms N` 启用单 client 的最小请求间隔。默认不主动节流,主要靠缓存 + 命中 429 后的重试自我恢复。

HTTP 重试覆盖 `429`、`502`、`503`、`504`。出现 `Retry-After` 时优先遵守(最长 30 秒,超过会立即放弃以免阻塞 CLI),否则按指数退避。网络层短暂错误也走同一重试流程。

非 JSON 模式下,等待 2 秒及以上的重试会向 `stderr` 输出一行简短提示,避免用户以为 CLI 卡住。使用 `--verbose` 会输出每一次重试提示。JSON 模式会保持 `stdout` 为纯 JSON,并抑制这些重试提示。

已观察到：

- `IWishlistService/GetWishlist/v1` 正常 `200` 响应包含 `x-eresult: 1`，没有 `Retry-After` 或 `X-RateLimit-*`。
- `appdetails` 正常 `200` 响应包含 `Cache-Control: public,max-age=3600`、`Expires`、`Last-Modified`，没有限流剩余额度。
- `appreviews` 正常 `200` 响应包含 `Cache-Control: private,max-age=600` 和 Steam cookie，未看到限流剩余额度。
- 同一 IP 对同一个 wishlist endpoint 约 14 秒内连续请求 7 次曾触发 429。这只是观测，不是 Valve 官方阈值。

官方公开信息：

- Steam Web API Terms of Use 写明 Steam Web API 限制为每天 100,000 次调用。
- [Steamworks Web API Overview][source-webapi-overview] 写明产生 `403` 的请求会对连接 IP 触发严格限流。
- Valve 没有公开每分钟、每 IP、每 endpoint 的具体 429 阈值。

## 测试

```bash
go vet ./...
go test -race ./...
go build -o steam-cli .
```

CI(`.github/workflows/ci.yml`)在每次 push 时跑 `go vet`、`go test -race` 和 `go build`。

单元测试覆盖：

- profile URL / 自定义 URL 名称 / SteamID64 解析。
- `Retry-After` 秒数和 HTTP date 解析。
- Store CDN asset URL 拼接(绝对地址、协议相对、format 占位符)。
- Store 搜索结果 HTML 解析。
- Steam 字段的灵活 JSON 解析。
- 商店评价、平台、语言摘要、购买项 ID 等终端输出 helper。
- HTTP 层基于 `httptest`:429 + `Retry-After`、5xx 重试、retry-after 上限、source-changed JSON、重试用尽、`--rate-ms` 节流。
- 类型化错误分类(`steam.CodeOf`、`errors.As`)。
- `en` 与 `zh-CN` 文案 keyset 一致性。

实时命令验证过：

```bash
./steam-cli price 264710 --cc US
./steam-cli app 264710 --news 1
./steam-cli media 264710
./steam-cli dlc 264710
./steam-cli similar 264710 --count 5
```

## 限制

- 当前折扣结束时间可以从 StoreBrowse 拿到；权威折扣开始时间没有公开字段。
- Steam 用户昵称模糊搜索没有稳定公开未登录接口，未集成。
- 不会绕过私密/仅好友可见的 profile、愿望单、库存或游戏库。
- StoreBrowse 和 StoreQuery 是公开可访问但未正式文档化的 Steam 服务，字段可能变化；部分命令会尽量 fallback 到旧 Store 接口。

[source-appdetails]: https://store.steampowered.com/api/appdetails?appids=264710&cc=US&l=english
[source-storebrowse]: https://api.steampowered.com/IStoreBrowseService/GetItems/v1/
[source-morelikethis]: https://api.steampowered.com/IStoreQueryService/MoreLikeThis/v1/
[source-appreviews]: https://store.steampowered.com/appreviews/264710?json=1
[source-currentplayers]: https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/?appid=264710
[source-achievements]: https://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/?gameid=264710
[source-news]: https://api.steampowered.com/ISteamNews/GetNewsForApp/v2/?appid=264710&count=3&format=json
[source-storesearch]: https://store.steampowered.com/api/storesearch?term=subnautica&cc=US&l=english
[source-search-results]: https://store.steampowered.com/search/results/?query=&start=0&count=10&dynamic_data=&infinite=1&specials=1&cc=US&l=english
[source-steamworks-events]: https://partner.steamgames.com/doc/marketing/upcoming_events?l=schinese
[source-webapi-overview]: https://partner.steamgames.com/doc/webapi_overview
[source-community-id-xml]: https://steamcommunity.com/id/nickxudotme/?xml=1
[source-community-xml]: https://steamcommunity.com/profiles/76561198115468824/?xml=1
[source-wishlist]: https://api.steampowered.com/IWishlistService/GetWishlist/v1/?steamid=76561198115468824
[source-cdn]: https://cdn.akamai.steamstatic.com/steam/apps/264710/library_600x900.jpg
[source-store-home]: https://store.steampowered.com/?l=schinese
