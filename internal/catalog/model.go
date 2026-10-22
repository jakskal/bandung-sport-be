// Package catalog loads and validates the read-only venue catalog.
//
// The JSON shape mirrors fe/src/data/types.js: fields that may be unknown are
// pointers so "null" stays distinguishable from a real zero value.
package catalog

// Tier is the price band shown on a card. A nil *Tier means the price has not
// been checked yet ("Harga belum dicek").
type Tier string

const (
	TierGratis Tier = "Gratis"
	TierMurah  Tier = "Murah"
	TierSedang Tier = "Sedang"
	TierMahal  Tier = "Mahal"
)

// Offer visibility: why a price is missing, not just that it is missing.
const (
	VisibilityPublic    = "public"
	VisibilityOnRequest = "on-request"
	VisibilityUnknown   = "unknown"
)

// Venue is a physical place. It knows nothing about prices; those live in
// Offering.
type Venue struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Area         string   `json:"area"`
	Sports       []string `json:"sports"`
	Lat          *float64 `json:"lat"`
	Lng          *float64 `json:"lng"`
	Address      *string  `json:"address"`
	WhatsApp     *string  `json:"whatsapp"`
	Instagram    *string  `json:"instagram"`
	Website      *string  `json:"website"`
	SourceURL    *string  `json:"sourceUrl"`
	Tags         []string `json:"tags"`
	LastVerified *string  `json:"lastVerified"`
}

// Offering is something a person can join or pay for.
type Offering struct {
	ID              string         `json:"id"`
	VenueID         *string        `json:"venueId"`
	OrganizerID     *string        `json:"organizerId"`
	Sport           string         `json:"sport"`
	OfferingType    string         `json:"offeringType"`
	Title           string         `json:"title"`
	Audience        []string       `json:"audience"`
	AgeMin          *int           `json:"ageMin"`
	AgeMax          *int           `json:"ageMax"`
	PriceVisibility string         `json:"priceVisibility"`
	PriceMin        *int           `json:"priceMin"`
	PriceMax        *int           `json:"priceMax"`
	PriceUnit       *string        `json:"priceUnit"`
	SessionsPerPack *int           `json:"sessionsPerPack"`
	GroupSize       *int           `json:"groupSize"`
	DurationMin     *int           `json:"durationMin"`
	JoinMethod      []string       `json:"joinMethod"`
	Schedule        []ScheduleSlot `json:"schedule"`
	LocationNote    *string        `json:"locationNote"`
	LastVerified    *string        `json:"lastVerified"`
}

// ScheduleSlot repeats weekly.
type ScheduleSlot struct {
	Days  []string `json:"days"`
	Start string   `json:"start"`
	End   string   `json:"end"`
}

// Price is the cheapest public offering of a venue, as sent in VenueSummary.
type Price struct {
	PriceMin        int    `json:"priceMin"`
	PriceMax        *int   `json:"priceMax"`
	PriceUnit       string `json:"priceUnit"`
	PriceVisibility string `json:"priceVisibility"`
}
