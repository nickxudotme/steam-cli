package steam

type storeItemID struct {
	AppID     int `json:"appid,omitempty"`
	PackageID int `json:"packageid,omitempty"`
	BundleID  int `json:"bundleid,omitempty"`
}

type storeBrowseContext struct {
	Language    string `json:"language"`
	CountryCode string `json:"country_code"`
	SteamRealm  int    `json:"steam_realm,omitempty"`
}

type storeItemDataRequest struct {
	IncludeAssets             bool `json:"include_assets,omitempty"`
	IncludeRelease            bool `json:"include_release,omitempty"`
	IncludePlatforms          bool `json:"include_platforms,omitempty"`
	IncludeAllPurchaseOptions bool `json:"include_all_purchase_options,omitempty"`
	IncludeScreenshots        bool `json:"include_screenshots,omitempty"`
	IncludeTrailers           bool `json:"include_trailers,omitempty"`
	IncludeRatings            bool `json:"include_ratings,omitempty"`
	IncludeTagCount           int  `json:"include_tag_count,omitempty"`
	IncludeReviews            bool `json:"include_reviews,omitempty"`
	IncludeBasicInfo          bool `json:"include_basic_info,omitempty"`
	IncludeSupportedLanguages bool `json:"include_supported_languages,omitempty"`
	IncludeFullDescription    bool `json:"include_full_description,omitempty"`
	IncludeLinks              bool `json:"include_links,omitempty"`
}

type storeBrowseRequest struct {
	IDs         []storeItemID        `json:"ids"`
	Context     storeBrowseContext   `json:"context"`
	DataRequest storeItemDataRequest `json:"data_request"`
}

type storeSimilarRequest struct {
	ItemID      storeItemID          `json:"item_id"`
	Context     storeBrowseContext   `json:"context"`
	DataRequest storeItemDataRequest `json:"data_request"`
	Count       int                  `json:"count"`
}

func fullStoreItemDataRequest() storeItemDataRequest {
	return storeItemDataRequest{
		IncludeAssets:             true,
		IncludeRelease:            true,
		IncludePlatforms:          true,
		IncludeAllPurchaseOptions: true,
		IncludeRatings:            true,
		IncludeTagCount:           12,
		IncludeReviews:            true,
		IncludeBasicInfo:          true,
		IncludeSupportedLanguages: true,
		IncludeLinks:              true,
	}
}
