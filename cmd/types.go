package cmd

type folio struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	DisplayName       string    `json:"display_name"`
	Kind              string    `json:"kind"`
	SystemKey         string    `json:"system_key"`
	Description       string    `json:"description"`
	OwnerEmail        string    `json:"owner_email"`
	Access            string    `json:"access"`
	CreatedAt         string    `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
	HoldingsUpdatedAt string    `json:"holdings_updated_at"`
	Entries           string    `json:"entries"`
	Holdings          []holding `json:"holdings"`
}

type holding struct {
	Ticker string `json:"ticker"`
	Shares any    `json:"shares"`
}

type share struct {
	ID             int    `json:"id"`
	Folio          folio  `json:"folio"`
	InviterEmail   string `json:"inviter_email"`
	RecipientEmail string `json:"recipient_email"`
	Permission     string `json:"permission"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

type report struct {
	GeneratedAt string `json:"generated_at"`
	Folio       folio  `json:"folio"`
	Results     []struct {
		Ticker        string             `json:"ticker"`
		Name          string             `json:"name"`
		Price         float64            `json:"price"`
		Value         *float64           `json:"value"`
		MarketCap     *float64           `json:"market_cap"`
		TrailingPE    *float64           `json:"trailing_pe"`
		Returns       map[string]float64 `json:"returns"`
		MomentumShape string             `json:"momentum_shape"`
	} `json:"results"`
}

type researchRun struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
	ErrorMessage string   `json:"error_message"`
	Tickers      []string `json:"tickers"`
	Results      []struct {
		Ticker        string             `json:"ticker"`
		Name          string             `json:"name"`
		Price         *float64           `json:"price"`
		TrailingPE    *float64           `json:"trailing_pe"`
		Returns       map[string]float64 `json:"returns"`
		ShortScore    *float64           `json:"short_score"`
		LongScore     *float64           `json:"long_score"`
		OverallScore  *float64           `json:"overall_score"`
		MomentumShape string             `json:"momentum_shape"`
		ErrorMessage  string             `json:"error_message"`
	} `json:"results"`
}
