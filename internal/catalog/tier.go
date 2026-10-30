package catalog

// tierBand is the price band for one unit inside one sport.
type tierBand struct {
	Murah  int
	Sedang int
}

// tierRules is keyed by sport, then by the unit that sport publishes prices in
// (docs/plan-v2.md § 6). A unit without a band has no tier: a per-session-pack
// price cannot be compared with a per-hour court rate while session length is
// unknown, so those venues stay tierless instead of getting a guessed bucket.
//
// Initial guesses: tune them after looking at the real price distribution.
var tierRules = map[string]map[string]tierBand{
	"fitness": {
		"monthly":   {Murah: 200000, Sedang: 600000},
		"per-visit": {Murah: 35000, Sedang: 75000},
	},
	"swimming": {"per-visit": {Murah: 25000, Sedang: 60000}},
	"futsal":   {"per-hour": {Murah: 120000, Sedang: 250000}},
	"padel":    {"per-hour": {Murah: 250000, Sedang: 450000}},
	"running":  {"per-visit": {Murah: 10000, Sedang: 30000}},
}

// Tier returns the price band for a venue, or nil when no public, general
// audience offering matches a band for any of its sports.
//
// Sports are checked in the venue's declared order and the first one with a
// usable band and price wins.
func (c *Catalog) Tier(venue Venue) *Tier {
	for _, sport := range venue.Sports {
		bands, ok := tierRules[sport]
		if !ok {
			continue
		}

		price, unit, ok := c.cheapestPublic(venue.ID, sport)
		if !ok {
			continue
		}
		if unit == "free" || price == 0 {
			tier := TierGratis
			return &tier
		}
		band, ok := bands[unit]
		if !ok {
			continue
		}

		tier := bucket(price, band.Murah, band.Sedang)
		return &tier
	}
	return nil
}

func bucket(price, murah, sedang int) Tier {
	switch {
	case price < murah:
		return TierMurah
	case price < sedang:
		return TierSedang
	default:
		return TierMahal
	}
}

// cheapestPublic returns the lowest price among public offerings for the venue
// and sport that are open to the general audience.
func (c *Catalog) cheapestPublic(venueID, sport string) (price int, unit string, ok bool) {
	for _, offering := range c.OfferingsByVenue(venueID) {
		if offering.Sport != sport || offering.PriceVisibility != VisibilityPublic {
			continue
		}
		if offering.PriceMin == nil || offering.PriceUnit == nil {
			continue
		}
		if !hasAudience(offering, "umum") {
			continue
		}
		if !ok || *offering.PriceMin < price {
			price, unit, ok = *offering.PriceMin, *offering.PriceUnit, true
		}
	}
	return price, unit, ok
}

func hasAudience(offering Offering, audience string) bool {
	for _, a := range offering.Audience {
		if a == audience {
			return true
		}
	}
	return false
}

// PriceFrom returns the cheapest public offering of a venue, or nil when the
// venue has no published price. Units are not normalised: this is the raw
// "mulai dari" figure shown on a card.
func (c *Catalog) PriceFrom(venue Venue) *Price {
	var cheapest *Offering

	for i := range c.Offerings {
		offering := &c.Offerings[i]
		if offering.VenueID == nil || *offering.VenueID != venue.ID {
			continue
		}
		if offering.PriceVisibility != VisibilityPublic || offering.PriceMin == nil || offering.PriceUnit == nil {
			continue
		}
		if cheapest == nil || *offering.PriceMin < *cheapest.PriceMin {
			cheapest = offering
		}
	}
	if cheapest == nil {
		return nil
	}

	return &Price{
		PriceMin:        *cheapest.PriceMin,
		PriceMax:        cheapest.PriceMax,
		PriceUnit:       *cheapest.PriceUnit,
		PriceVisibility: cheapest.PriceVisibility,
		SessionsPerPack: cheapest.SessionsPerPack,
	}
}
