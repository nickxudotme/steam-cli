package steam

type AppBundle struct {
	AppID          int            `json:"appid"`
	Details        *AppDetails    `json:"details"`
	StoreItem      *StoreItem     `json:"store_item,omitempty"`
	Reviews        *ReviewSummary `json:"reviews"`
	CurrentPlayers *int           `json:"current_players,omitempty"`
	News           []NewsItem     `json:"news"`
	// Warnings collects non-fatal failures from sibling lookups (store item,
	// reviews, current players, news). The bundle is returned even when these
	// fail so the user gets partial data; the warnings explain what's missing.
	Warnings []string `json:"warnings,omitempty"`
}

type AppDetails struct {
	Type               string             `json:"type"`
	Name               string             `json:"name"`
	SteamAppID         int                `json:"steam_appid"`
	RequiredAge        FlexibleString     `json:"required_age"`
	IsFree             bool               `json:"is_free"`
	ControllerSupport  string             `json:"controller_support"`
	ShortDescription   string             `json:"short_description"`
	SupportedLanguages string             `json:"supported_languages"`
	HeaderImage        string             `json:"header_image"`
	Website            *string            `json:"website"`
	Developers         []string           `json:"developers"`
	Publishers         []string           `json:"publishers"`
	PriceOverview      *PriceOverview     `json:"price_overview,omitempty"`
	Genres             []NamedValue       `json:"genres"`
	Categories         []NamedValue       `json:"categories"`
	DLC                []int              `json:"dlc"`
	Packages           []int              `json:"packages"`
	PackageGroups      []PackageGroup     `json:"package_groups"`
	ReleaseDate        ReleaseDate        `json:"release_date"`
	Metacritic         *Metacritic        `json:"metacritic,omitempty"`
	Recommendations    *Recommendations   `json:"recommendations,omitempty"`
	Achievements       *AppAchievements   `json:"achievements,omitempty"`
	Platforms          map[string]bool    `json:"platforms"`
	Screenshots        []Screenshot       `json:"screenshots"`
	Movies             []Movie            `json:"movies"`
	PCRequirements     Requirements       `json:"pc_requirements"`
	MacRequirements    Requirements       `json:"mac_requirements"`
	LinuxRequirements  Requirements       `json:"linux_requirements"`
	SupportInfo        SupportInfo        `json:"support_info"`
	ContentDescriptors ContentDescriptors `json:"content_descriptors"`
}

type PriceOverview struct {
	Currency         string `json:"currency"`
	Initial          int    `json:"initial"`
	Final            int    `json:"final"`
	DiscountPercent  int    `json:"discount_percent"`
	InitialFormatted string `json:"initial_formatted"`
	FinalFormatted   string `json:"final_formatted"`
}

type StoreItem struct {
	AppID                 int              `json:"appid"`
	ID                    int              `json:"id,omitempty"`
	ItemType              int              `json:"item_type,omitempty"`
	Success               int              `json:"success,omitempty"`
	Visible               bool             `json:"visible,omitempty"`
	Name                  string           `json:"name"`
	StoreURLPath          string           `json:"store_url_path,omitempty"`
	Assets                *StoreAssets     `json:"assets,omitempty"`
	BasicInfo             *StoreBasicInfo  `json:"basic_info,omitempty"`
	Release               *StoreRelease    `json:"release,omitempty"`
	Platforms             *StorePlatforms  `json:"platforms,omitempty"`
	Reviews               *StoreReviews    `json:"reviews,omitempty"`
	Tags                  []StoreTag       `json:"tags,omitempty"`
	TagIDs                []int            `json:"tagids,omitempty"`
	GameRating            *GameRating      `json:"game_rating,omitempty"`
	Links                 []StoreLink      `json:"links,omitempty"`
	SupportedLanguages    []StoreLanguage  `json:"supported_languages,omitempty"`
	FullDescriptionBBCode string           `json:"full_description_bbcode,omitempty"`
	BestPurchaseOption    *PurchaseOption  `json:"best_purchase_option,omitempty"`
	PurchaseOptions       []PurchaseOption `json:"purchase_options,omitempty"`
	IsComingSoon          bool             `json:"is_coming_soon,omitempty"`
	IsFree                bool             `json:"is_free,omitempty"`
}

type StoreAssets struct {
	AssetURLFormat     string `json:"asset_url_format"`
	MainCapsule        string `json:"main_capsule"`
	MainCapsule2x      string `json:"main_capsule_2x"`
	SmallCapsule       string `json:"small_capsule"`
	SmallCapsule2x     string `json:"small_capsule_2x"`
	Header             string `json:"header"`
	Header2x           string `json:"header_2x"`
	HeroCapsule        string `json:"hero_capsule"`
	HeroCapsule2x      string `json:"hero_capsule_2x"`
	LibraryCapsule     string `json:"library_capsule"`
	LibraryCapsule2x   string `json:"library_capsule_2x"`
	LibraryHero        string `json:"library_hero"`
	LibraryHero2x      string `json:"library_hero_2x"`
	CommunityIcon      string `json:"community_icon"`
	PageBackground     string `json:"page_background"`
	PageBackgroundPath string `json:"page_background_path"`
	RawPageBackground  string `json:"raw_page_background"`
}

type StoreBasicInfo struct {
	ShortDescription string             `json:"short_description"`
	Publishers       []StoreAssociation `json:"publishers"`
	Developers       []StoreAssociation `json:"developers"`
	Franchises       []StoreAssociation `json:"franchises"`
}

type StoreAssociation struct {
	Name                 string `json:"name"`
	CreatorClanAccountID int64  `json:"creator_clan_account_id,omitempty"`
}

type StoreRelease struct {
	SteamReleaseDate         int64 `json:"steam_release_date,omitempty"`
	OriginalSteamReleaseDate int64 `json:"original_steam_release_date,omitempty"`
	OriginalReleaseDate      int64 `json:"original_release_date,omitempty"`
	IsComingSoon             bool  `json:"is_coming_soon,omitempty"`
}

type StorePlatforms struct {
	Windows                 bool       `json:"windows"`
	Mac                     bool       `json:"mac"`
	Linux                   bool       `json:"linux"`
	VRSupport               *VRSupport `json:"vr_support,omitempty"`
	SteamDeckCompatCategory int        `json:"steam_deck_compat_category,omitempty"`
	SteamOSCompatCategory   int        `json:"steam_os_compat_category,omitempty"`
}

type VRSupport struct {
	VRHMD      bool `json:"vrhmd,omitempty"`
	HTCVive    bool `json:"htc_vive,omitempty"`
	OculusRift bool `json:"oculus_rift,omitempty"`
	ValveIndex bool `json:"valve_index,omitempty"`
}

type StoreReviews struct {
	SummaryFiltered         *StoreReviewSummary `json:"summary_filtered,omitempty"`
	SummaryLanguageSpecific *StoreReviewSummary `json:"summary_language_specific,omitempty"`
}

type StoreReviewSummary struct {
	ReviewCount      int    `json:"review_count"`
	PercentPositive  int    `json:"percent_positive"`
	ReviewScore      int    `json:"review_score"`
	ReviewScoreLabel string `json:"review_score_label"`
}

type StoreTag struct {
	TagID  int `json:"tagid"`
	Weight int `json:"weight"`
}

type GameRating struct {
	Type        string   `json:"type"`
	Rating      string   `json:"rating"`
	Descriptors []string `json:"descriptors"`
	RequiredAge int      `json:"required_age"`
	UseAgeGate  bool     `json:"use_age_gate"`
	ImageURL    string   `json:"image_url"`
}

type StoreLink struct {
	LinkType int    `json:"link_type"`
	URL      string `json:"url"`
}

type StoreLanguage struct {
	ELanguage           int  `json:"elanguage"`
	EAdditionalLanguage int  `json:"eadditionallanguage"`
	Supported           bool `json:"supported"`
	FullAudio           bool `json:"full_audio"`
	Subtitles           bool `json:"subtitles"`
}

type PurchaseOption struct {
	PackageID                          int              `json:"packageid,omitempty"`
	BundleID                           int              `json:"bundleid,omitempty"`
	PurchaseOptionName                 string           `json:"purchase_option_name"`
	FinalPriceInCents                  int64            `json:"final_price_in_cents,string"`
	OriginalPriceInCents               int64            `json:"original_price_in_cents,string"`
	FormattedFinalPrice                string           `json:"formatted_final_price"`
	FormattedOriginalPrice             string           `json:"formatted_original_price"`
	DiscountPct                        int              `json:"discount_pct"`
	BundleDiscountPct                  int              `json:"bundle_discount_pct,omitempty"`
	IsFreeToKeep                       bool             `json:"is_free_to_keep,omitempty"`
	PriceBeforeBundleDiscount          int64            `json:"price_before_bundle_discount,string,omitempty"`
	FormattedPriceBeforeBundleDiscount string           `json:"formatted_price_before_bundle_discount,omitempty"`
	ActiveDiscounts                    []ActiveDiscount `json:"active_discounts,omitempty"`
	FreeToKeepEnds                     int64            `json:"free_to_keep_ends,omitempty"`
	PackageGroup                       string           `json:"package_group,omitempty"`
	PriceCannotBeDisplayedAsDiscount   bool             `json:"price_cannot_be_displayed_as_discount,omitempty"`
}

type ActiveDiscount struct {
	DiscountAmount      int64  `json:"discount_amount,string"`
	DiscountDescription string `json:"discount_description"`
	DiscountEndDate     int64  `json:"discount_end_date"`
}

type DLCResult struct {
	AppID int         `json:"appid"`
	Name  string      `json:"name"`
	Items []StoreItem `json:"items"`
	Total int         `json:"total"`
}

type SimilarResult struct {
	AppID int         `json:"appid"`
	Items []StoreItem `json:"items"`
	Total int         `json:"total"`
}

type NamedValue struct {
	ID          any    `json:"id"`
	Description string `json:"description"`
}

type ReleaseDate struct {
	ComingSoon bool   `json:"coming_soon"`
	Date       string `json:"date"`
}

type Metacritic struct {
	Score int    `json:"score"`
	URL   string `json:"url"`
}

type Recommendations struct {
	Total int `json:"total"`
}

type AppAchievements struct {
	Total       int                      `json:"total"`
	Highlighted []HighlightedAchievement `json:"highlighted"`
}

type HighlightedAchievement struct {
	Name          string `json:"name"`
	LocalizedName string `json:"localized_name"`
	Path          string `json:"path"`
}

type PackageGroup struct {
	Name  string       `json:"name"`
	Title string       `json:"title"`
	Subs  []PackageSub `json:"subs"`
}

type PackageSub struct {
	PackageID                int    `json:"packageid"`
	PercentSavingsText       string `json:"percent_savings_text"`
	OptionText               string `json:"option_text"`
	PriceInCentsWithDiscount int    `json:"price_in_cents_with_discount"`
}

type Screenshot struct {
	ID            int    `json:"id"`
	PathThumbnail string `json:"path_thumbnail"`
	PathFull      string `json:"path_full"`
}

type Movie struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Thumbnail string `json:"thumbnail"`
	Highlight bool   `json:"highlight"`
}

type Requirements struct {
	Minimum     string `json:"minimum"`
	Recommended string `json:"recommended"`
}

type SupportInfo struct {
	URL   string `json:"url"`
	Email string `json:"email"`
}

type ContentDescriptors struct {
	IDs   []int `json:"ids"`
	Notes any   `json:"notes"`
}

type ReviewSummary struct {
	ReviewScore     int    `json:"review_score"`
	ReviewScoreDesc string `json:"review_score_desc"`
	TotalPositive   int    `json:"total_positive"`
	TotalNegative   int    `json:"total_negative"`
	TotalReviews    int    `json:"total_reviews"`
}

type NewsItem struct {
	GID           string `json:"gid"`
	Title         string `json:"title"`
	URL           string `json:"url"`
	IsExternalURL bool   `json:"is_external_url"`
	Author        string `json:"author"`
	Contents      string `json:"contents"`
	FeedLabel     string `json:"feedlabel"`
	FeedName      string `json:"feedname"`
	Date          int64  `json:"date"`
}

type SearchItem struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	TinyImage string         `json:"tiny_image"`
	Price     *PriceOverview `json:"price,omitempty"`
	Metascore string         `json:"metascore,omitempty"`
}

type ReviewResponse struct {
	Success      int           `json:"success"`
	QuerySummary ReviewSummary `json:"query_summary"`
	Reviews      []Review      `json:"reviews"`
	Cursor       string        `json:"cursor"`
}

type Review struct {
	RecommendationID         string       `json:"recommendationid"`
	Author                   ReviewAuthor `json:"author"`
	Language                 string       `json:"language"`
	Review                   string       `json:"review"`
	TimestampCreated         int64        `json:"timestamp_created"`
	TimestampUpdated         int64        `json:"timestamp_updated"`
	VotedUp                  bool         `json:"voted_up"`
	VotesUp                  int          `json:"votes_up"`
	VotesFunny               int          `json:"votes_funny"`
	WeightedVoteScore        any          `json:"weighted_vote_score"`
	CommentCount             int          `json:"comment_count"`
	SteamPurchase            bool         `json:"steam_purchase"`
	ReceivedForFree          bool         `json:"received_for_free"`
	Refunded                 bool         `json:"refunded"`
	WrittenDuringEarlyAccess bool         `json:"written_during_early_access"`
}

type ReviewAuthor struct {
	SteamID              string `json:"steamid"`
	PersonaName          string `json:"personaname"`
	PlaytimeForever      int    `json:"playtime_forever"`
	PlaytimeAtReview     int    `json:"playtime_at_review"`
	PlaytimeLastTwoWeeks int    `json:"playtime_last_two_weeks"`
}

type GlobalAchievement struct {
	Name    string  `json:"name"`
	Percent float64 `json:"percent"`
}

type StoreResult struct {
	AppID       int    `json:"appid"`
	Name        string `json:"name"`
	Release     string `json:"-"`
	ReleaseTime int64  `json:"release_time,omitempty"`
	Review      string `json:"review"`
	Discount    string `json:"discount"`
	Original    string `json:"original"`
	Final       string `json:"final"`
	DiscountEnd int64  `json:"discount_end,omitempty"`
	URL         string `json:"url"`
}

type StoreResultCondition string

const (
	StoreResultConditionDiscounted StoreResultCondition = "discounted"
	StoreResultConditionPreorder   StoreResultCondition = "preorder"
)

type StoreResultsQuery struct {
	Filter string                 `json:"filter"`
	Count  int                    `json:"count"`
	Any    []StoreResultCondition `json:"any,omitempty"`
	All    []StoreResultCondition `json:"all,omitempty"`
}

type UserProfile struct {
	SteamID64        string `xml:"steamID64" json:"steamid64"`
	SteamID          string `xml:"steamID" json:"steamid"`
	OnlineState      string `xml:"onlineState" json:"online_state"`
	StateMessage     string `xml:"stateMessage" json:"state_message"`
	PrivacyState     string `xml:"privacyState" json:"privacy_state"`
	VisibilityState  int    `xml:"visibilityState" json:"visibility_state"`
	AvatarIcon       string `xml:"avatarIcon" json:"avatar_icon"`
	AvatarMedium     string `xml:"avatarMedium" json:"avatar_medium"`
	AvatarFull       string `xml:"avatarFull" json:"avatar_full"`
	VACBanned        int    `xml:"vacBanned" json:"vac_banned"`
	TradeBanState    string `xml:"tradeBanState" json:"trade_ban_state"`
	IsLimitedAccount int    `xml:"isLimitedAccount" json:"is_limited_account"`
	CustomURL        string `xml:"customURL" json:"custom_url,omitempty"`
	MemberSince      string `xml:"memberSince" json:"member_since,omitempty"`
	Location         string `xml:"location" json:"location,omitempty"`
	RealName         string `xml:"realname" json:"real_name,omitempty"`
	Summary          string `xml:"summary" json:"summary,omitempty"`
}

type WishlistItem struct {
	AppID     int         `json:"appid"`
	Priority  int         `json:"priority"`
	DateAdded int64       `json:"date_added"`
	Details   *AppDetails `json:"details,omitempty"`
	Error     string      `json:"error,omitempty"`
}

type Wishlist struct {
	SteamID64 string         `json:"steamid64"`
	Items     []WishlistItem `json:"items"`
	Total     int            `json:"total"`
	Offset    int            `json:"offset"`
	Count     int            `json:"count"`
}

type MediaAsset struct {
	Kind      string `json:"kind"`
	Name      string `json:"name,omitempty"`
	URL       string `json:"url"`
	Available *bool  `json:"available,omitempty"`
	Status    int    `json:"status,omitempty"`
}

type Media struct {
	AppID            int                      `json:"appid"`
	Name             string                   `json:"name"`
	HeaderImage      string                   `json:"header_image,omitempty"`
	CDNAssets        []MediaAsset             `json:"cdn_assets"`
	Screenshots      []Screenshot             `json:"screenshots"`
	Movies           []Movie                  `json:"movies"`
	AchievementIcons []HighlightedAchievement `json:"achievement_icons"`
}
