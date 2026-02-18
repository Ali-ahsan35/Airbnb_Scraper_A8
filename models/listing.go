package models

type Listing struct {
	ID          int
	Platform    string
	Title       string
	Price       float64
	RawPrice    string
	Location    string
	Rating      float64
	ReviewCount int
	URL         string
	Description string
}

type ScrapeJob struct {
	URL        string
	PageNumber int
}

type ScrapeResult struct {
	Listings   []Listing
	Error      error
	PageNumber int
}