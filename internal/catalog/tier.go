package catalog

// tierRules is keyed by sport and compares raw prices in that sport's natural
// unit. A sport without a rule has no tier until a rule is added.
//
// Initial guesses: tune them after looking at the real price distribution.
var tierRules = map[string]struct {
	Unit   string
	Murah  int
	Sedang int
}{
	"fitness":  {Unit: "monthly", Murah: 200000, Sedang: 600000},
	"swimming": {Unit: "per-visit", Murah: 25000, Sedang: 60000},
	"futsal":   {Unit: "per-hour", Murah: 120000, Sedang: 250000},
	"padel":    {Unit: "per-hour", Murah: 250000, Sedang: 450000},
	"running":  {Unit: "per-visit", Murah: 10000, Sedang: 30000},
}

// Tier returns the price band for a venue, or nil when no public, general
// audience offering matches the rule for any of its sports.
//
// Sports are checked in the venue's declared order and the first one with a
// usable rule and price wins.
func (c *Catalog) Tier(venue Venue) *Tier {
	for _, sport := range venue.Sports {
		rule, ok := tierRules[sport]
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
		if unit != rule.Unit {
			continue
		}

		tier := bucket(price, rule.Murah, rule.Sedang)
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
	}
}
