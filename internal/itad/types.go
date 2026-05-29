package itad

type Game struct {
	ID     string     `json:"id"`
	Slug   string     `json:"slug"`
	Title  string     `json:"title"`
	Type   string     `json:"type"`
	Mature bool       `json:"mature"`
	Assets GameAssets `json:"assets"`
	AppID  int        `json:"appid,omitempty"`
}

type GameAssets struct {
	BoxArt    string `json:"boxart"`
	Banner145 string `json:"banner145"`
	Banner300 string `json:"banner300"`
	Banner400 string `json:"banner400"`
	Banner600 string `json:"banner600"`
}

type Money struct {
	Amount    float64 `json:"amount"`
	AmountInt int     `json:"amountInt"`
	Currency  string  `json:"currency"`
}

type Shop struct {
	ID     int     `json:"id"`
	Title  string  `json:"title"`
	Deals  int     `json:"deals"`
	Games  int     `json:"games"`
	Update *string `json:"update"`
}

type ShopRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Deal struct {
	Shop      ShopRef   `json:"shop"`
	Price     Money     `json:"price"`
	Regular   Money     `json:"regular"`
	Cut       int       `json:"cut"`
	Timestamp string    `json:"timestamp"`
	Expiry    *string   `json:"expiry"`
	URL       string    `json:"url"`
	DRM       []ShopRef `json:"drm,omitempty"`
}

type HistoricDeal struct {
	Shop      ShopRef `json:"shop"`
	Price     Money   `json:"price"`
	Regular   Money   `json:"regular"`
	Cut       int     `json:"cut"`
	Timestamp string  `json:"timestamp"`
}

type Overview struct {
	ID      string        `json:"id"`
	Current *Deal         `json:"current,omitempty"`
	Lowest  *HistoricDeal `json:"lowest,omitempty"`
	Bundled int           `json:"bundled"`
	URLs    struct {
		Game string `json:"game"`
	} `json:"urls"`
}

type Bundle struct {
	ID      int     `json:"id"`
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Details string  `json:"details"`
	Publish string  `json:"publish"`
	Expiry  *string `json:"expiry"`
	Counts  struct {
		Games int `json:"games"`
		Media int `json:"media"`
	} `json:"counts"`
	Tiers []BundleTier `json:"tiers"`
}

type BundleTier struct {
	Price Money  `json:"price"`
	Addon bool   `json:"addon"`
	Games []Game `json:"games"`
}

type HistoryLow struct {
	ID         string `json:"id"`
	HistoryLow struct {
		All *Money `json:"all"`
		Y1  *Money `json:"y1"`
		M3  *Money `json:"m3"`
	} `json:"historyLow"`
	Deals []Deal `json:"deals"`
}

type StoreLow struct {
	ID  string        `json:"id"`
	Low *HistoricDeal `json:"low,omitempty"`
}

type HistoryEntry struct {
	Timestamp string  `json:"timestamp"`
	Shop      ShopRef `json:"shop"`
	Deal      struct {
		Price   Money `json:"price"`
		Regular Money `json:"regular"`
		Cut     int   `json:"cut"`
	} `json:"deal"`
}

type Summary struct {
	Game       Game        `json:"game"`
	Overview   *Overview   `json:"overview,omitempty"`
	HistoryLow *HistoryLow `json:"history_low,omitempty"`
	SteamLow   *StoreLow   `json:"steam_low,omitempty"`
	Bundles    []Bundle    `json:"bundles,omitempty"`
}
