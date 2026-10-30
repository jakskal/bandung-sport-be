package catalog

import (
	"reflect"
	"testing"

	"github.com/jakskal/bandung-sport-be/data"
)

func intPtr(v int) *int { return &v }

func strPtr(v string) *string { return &v }

func tierPtr(t Tier) *Tier { return &t }

// offeringFor builds a valid public offering; mutate overrides one field.
func offeringFor(venueID, sport, unit string, price int, mutate ...func(*Offering)) Offering {
	o := Offering{
		ID:              "of-test",
		VenueID:         strPtr(venueID),
		Sport:           sport,
		OfferingType:    "membership",
		Title:           "Test offering",
		Audience:        []string{"umum"},
		PriceVisibility: VisibilityPublic,
		PriceMin:        intPtr(price),
		PriceUnit:       strPtr(unit),
		JoinMethod:      []string{"wa"},
		Schedule:        []ScheduleSlot{},
	}
	for _, fn := range mutate {
		fn(&o)
	}
	return o
}

func newCatalog(venues []Venue, offerings ...Offering) *Catalog {
	c := &Catalog{Venues: venues, Offerings: offerings}
	c.index()
	return c
}

func TestTier(t *testing.T) {
	fitnessVenue := Venue{ID: "v-a", Sports: []string{"fitness"}}

	tests := []struct {
		name      string
		venue     Venue
		offerings []Offering
		want      *Tier
	}{
		{
			name:  "no offerings means no tier",
			venue: fitnessVenue,
			want:  nil,
		},
		{
			name:      "cheapest public offering wins",
			venue:     fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "monthly", 300000), offeringFor("v-a", "fitness", "monthly", 100000)},
			want:      tierPtr(TierMurah),
		},
		{
			name:      "price below murah threshold",
			venue:     fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "monthly", 150000)},
			want:      tierPtr(TierMurah),
		},
		{
			name:      "price exactly at murah threshold is sedang",
			venue:     fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "monthly", 200000)},
			want:      tierPtr(TierSedang),
		},
		{
			name:      "price below sedang threshold is sedang",
			venue:     fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "monthly", 599999)},
			want:      tierPtr(TierSedang),
		},
		{
			name:      "price at sedang threshold is mahal",
			venue:     fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "monthly", 600000)},
			want:      tierPtr(TierMahal),
		},
		{
			name:      "price zero is gratis",
			venue:     Venue{ID: "v-a", Sports: []string{"running"}},
			offerings: []Offering{offeringFor("v-a", "running", "per-visit", 0)},
			want:      tierPtr(TierGratis),
		},
		{
			name:      "unit free is gratis",
			venue:     Venue{ID: "v-a", Sports: []string{"running"}},
			offerings: []Offering{offeringFor("v-a", "running", "free", 0)},
			want:      tierPtr(TierGratis),
		},
		{
			name:      "unit mismatch yields no tier",
			venue:     fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "weekly", 100000)},
			want:      nil,
		},
		{
			name:  "on-request price yields no tier",
			venue: fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "monthly", 0, func(o *Offering) {
				o.PriceVisibility = VisibilityOnRequest
				o.PriceMin = nil
				o.PriceUnit = nil
			})},
			want: nil,
		},
		{
			name:  "audience without umum yields no tier",
			venue: fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "monthly", 100000, func(o *Offering) {
				o.Audience = []string{"anak"}
			})},
			want: nil,
		},
		{
			name:      "sport without a rule yields no tier",
			venue:     Venue{ID: "v-a", Sports: []string{"yoga"}},
			offerings: []Offering{offeringFor("v-a", "yoga", "per-visit", 100000)},
			want:      nil,
		},
		{
			name:      "multi sport uses the first sport with a rule",
			venue:     Venue{ID: "v-a", Sports: []string{"yoga", "futsal"}},
			offerings: []Offering{offeringFor("v-a", "futsal", "per-hour", 150000)},
			want:      tierPtr(TierSedang),
		},
		{
			name:      "offerings of another venue are ignored",
			venue:     fitnessVenue,
			offerings: []Offering{offeringFor("v-b", "fitness", "monthly", 100000)},
			want:      nil,
		},
		{
			name:      "per-visit fitness uses the visit band",
			venue:     fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "per-visit", 20000)},
			want:      tierPtr(TierMurah),
		},
		{
			name:      "per-visit fitness above the visit band is sedang",
			venue:     fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "per-visit", 40000)},
			want:      tierPtr(TierSedang),
		},
		{
			name:      "cheapest offering picks the band of its own unit",
			venue:     fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "monthly", 300000), offeringFor("v-a", "fitness", "per-visit", 25000)},
			want:      tierPtr(TierMurah),
		},
		{
			name:  "per-session-pack has no band yet",
			venue: fitnessVenue,
			offerings: []Offering{offeringFor("v-a", "fitness", "per-session-pack", 700000, func(o *Offering) {
				o.SessionsPerPack = intPtr(12)
			})},
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newCatalog([]Venue{tc.venue}, tc.offerings...)

			got := c.Tier(tc.venue)
			if (got == nil) != (tc.want == nil) {
				t.Fatalf("Tier() = %v, want %v", deref(got), deref(tc.want))
			}
			if got != nil && *got != *tc.want {
				t.Fatalf("Tier() = %q, want %q", *got, *tc.want)
			}
		})
	}
}

func deref(t *Tier) string {
	if t == nil {
		return "nil"
	}
	return string(*t)
}

func TestPriceFrom(t *testing.T) {
	venue := Venue{ID: "v-a"}

	tests := []struct {
		name      string
		offerings []Offering
		want      *Price
	}{
		{
			name: "no offerings",
			want: nil,
		},
		{
			name:      "cheapest public offering",
			offerings: []Offering{offeringFor("v-a", "fitness", "monthly", 500000), offeringFor("v-a", "fitness", "monthly", 250000)},
			want:      &Price{PriceMin: 250000, PriceUnit: "monthly", PriceVisibility: VisibilityPublic},
		},
		{
			name: "keeps priceMax when present",
			offerings: []Offering{offeringFor("v-a", "fitness", "monthly", 250000, func(o *Offering) {
				o.PriceMax = intPtr(325000)
			})},
			want: &Price{PriceMin: 250000, PriceMax: intPtr(325000), PriceUnit: "monthly", PriceVisibility: VisibilityPublic},
		},
		{
			name: "skips non-public offerings",
			offerings: []Offering{offeringFor("v-a", "fitness", "monthly", 100000, func(o *Offering) {
				o.PriceVisibility = VisibilityUnknown
				o.PriceMin = nil
				o.PriceUnit = nil
			})},
			want: nil,
		},
		{
			name:      "skips other venues",
			offerings: []Offering{offeringFor("v-b", "fitness", "monthly", 100000)},
			want:      nil,
		},
		{
			name: "carries sessionsPerPack for pack prices",
			offerings: []Offering{offeringFor("v-a", "fitness", "per-session-pack", 700000, func(o *Offering) {
				o.SessionsPerPack = intPtr(12)
			})},
			want: &Price{PriceMin: 700000, PriceUnit: "per-session-pack", PriceVisibility: VisibilityPublic, SessionsPerPack: intPtr(12)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newCatalog([]Venue{venue}, tc.offerings...)

			if got := c.PriceFrom(venue); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("PriceFrom() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestLoadIndexes(t *testing.T) {
	c, err := Load(data.FS)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	venue, ok := c.Venue("v-ftl-gym-dago-bandung")
	if !ok {
		t.Fatal("Venue(v-ftl-gym-dago-bandung) not found")
	}
	if got := c.OfferingsByVenue(venue.ID); len(got) != 1 {
		t.Fatalf("OfferingsByVenue(%q) = %d, want 1", venue.ID, len(got))
	}
}

// TestDataIsValid is the permanent validator: it fails the build when the
// embedded data breaks a rule from docs/plan-v2.md § 6.
func TestDataIsValid(t *testing.T) {
	c, err := Load(data.FS)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if errors := c.Validate(); len(errors) > 0 {
		for _, err := range errors {
			t.Errorf("invalid data: %v", err)
		}
		t.Fatalf("%d validation error(s)", len(errors))
	}

	if got := len(c.Venues); got != 22 {
		t.Errorf("venues = %d, want 22", got)
	}
	if got := len(c.Offerings); got < 8 {
		t.Errorf("offerings = %d, want >= 8", got)
	}
}
