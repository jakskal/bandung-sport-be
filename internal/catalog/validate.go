package catalog

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Locked enums (docs/plan-v2.md § 6). Adding a value is a PR, not a guess.
var (
	sports = []string{
		"fitness", "futsal", "padel", "badminton", "basketball", "swimming",
		"running", "football", "yoga", "pilates", "zumba", "muay-thai", "hiking",
	}
	audiences     = []string{"umum", "anak", "remaja", "lansia", "prenatal", "pemula", "perempuan"}
	offeringTypes = []string{
		"membership", "visit-pass", "personal-training", "class",
		"court-rental", "program", "community-session", "free-access",
	}
	priceUnits = []string{
		"monthly", "weekly", "per-visit", "per-hour",
		"per-session-pack", "per-year", "free",
	}
	joinMethods       = []string{"walk-in", "wa", "daftar-online"}
	weekdays          = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}
	priceVisibilities = []string{VisibilityPublic, VisibilityOnRequest, VisibilityUnknown}
)

var (
	waPattern   = regexp.MustCompile(`^\+62\d{8,13}$`)
	datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
)

// Bandung Raya bounding box and the freshness window from plan-v2 § 6.
const (
	latMin, latMax = -7.2, -6.7
	lngMin, lngMax = 107.4, 107.9
	freshDays      = 45
)

// Validate returns hard rule violations. Warnings (no offering yet, stale
// lastVerified, empty join methods) are returned by Warnings.
func (c *Catalog) Validate() []error { return c.check().errors }

// Warnings returns non-blocking findings for the curation backlog.
func (c *Catalog) Warnings() []string { return c.check().warnings }

type checker struct {
	errors   []error
	warnings []string
}

func (chk *checker) errorf(format string, args ...any) {
	chk.errors = append(chk.errors, fmt.Errorf(format, args...))
}

func (chk *checker) warnf(format string, args ...any) {
	chk.warnings = append(chk.warnings, fmt.Sprintf(format, args...))
}

func (chk *checker) enum(where, value string, allowed []string) {
	if !slices.Contains(allowed, value) {
		chk.errorf("%s: %q is not one of %s", where, value, strings.Join(allowed, " | "))
	}
}

func (chk *checker) enumList(where string, values, allowed []string) {
	if values == nil {
		chk.errorf("%s: must be an array", where)
		return
	}
	if len(values) == 0 {
		chk.warnf("%s: empty array", where)
	}
	for i, value := range values {
		chk.enum(fmt.Sprintf("%s[%d]", where, i), value, allowed)
	}
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if seen[value] {
			chk.errorf("%s: contains duplicates", where)
			break
		}
		seen[value] = true
	}
}

func (chk *checker) id(where string, seen map[string]bool, id, prefix string) {
	switch {
	case !strings.HasPrefix(id, prefix):
		chk.errorf("%s: must be a string starting with %q", where, prefix)
	case seen[id]:
		chk.errorf("%s: duplicate id %q", where, id)
	default:
		seen[id] = true
	}
}

func (chk *checker) lastVerified(where string, value, whatsapp *string) {
	if value == nil {
		return
	}
	if !datePattern.MatchString(*value) {
		chk.errorf("%s: %q is not a YYYY-MM-DD date", where, *value)
		return
	}
	date, err := time.Parse("2006-01-02", *value)
	if err != nil {
		chk.errorf("%s: %q is not a valid date", where, *value)
		return
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if date.After(today) {
		chk.errorf("%s: %q is in the future", where, *value)
		return
	}

	age := int(today.Sub(date).Hours() / 24)
	if age <= freshDays {
		return
	}
	link := ""
	if whatsapp != nil {
		text := url.QueryEscape("Halo, saya dari Bandung Sport. Mau konfirmasi harga/jadwal yang masih berlaku. Terima kasih.")
		link = fmt.Sprintf(" https://wa.me/%s?text=%s", strings.TrimPrefix(*whatsapp, "+"), text)
	}
	chk.warnf("%s: %d days old - re-verify%s", where, age, link)
}

func (c *Catalog) check() *checker {
	chk := &checker{}

	venueIDs := make(map[string]bool, len(c.Venues))
	venuesWithOffering := make(map[string]bool, len(c.Offerings))
	for _, offering := range c.Offerings {
		if offering.VenueID != nil {
			venuesWithOffering[*offering.VenueID] = true
		}
	}

	for i := range c.Venues {
		c.checkVenue(chk, i, venueIDs, venuesWithOffering)
	}

	offeringIDs := make(map[string]bool, len(c.Offerings))
	for i := range c.Offerings {
		c.checkOffering(chk, i, offeringIDs, venueIDs)
	}

	return chk
}

func (c *Catalog) checkVenue(chk *checker, i int, venueIDs, venuesWithOffering map[string]bool) {
	venue := &c.Venues[i]
	at := fmt.Sprintf("venues[%d]", i)

	chk.id(at+".id", venueIDs, venue.ID, "v-")

	for _, field := range []struct {
		name  string
		value string
	}{{"name", venue.Name}, {"area", venue.Area}} {
		if strings.TrimSpace(field.value) == "" {
			chk.errorf("%s.%s: must be a non-empty string", at, field.name)
		}
	}

	if len(venue.Sports) == 0 {
		chk.errorf("%s.sports: must be a non-empty array", at)
	}
	for j, sport := range venue.Sports {
		chk.enum(fmt.Sprintf("%s.sports[%d]", at, j), sport, sports)
	}

	switch {
	case venue.Lat == nil && venue.Lng == nil:
		// Allowed: coordinates are filled during the curation pass.
	case venue.Lat == nil || venue.Lng == nil:
		chk.errorf("%s: lat and lng must both be set or both be null", at)
	default:
		if *venue.Lat < latMin || *venue.Lat > latMax {
			chk.errorf("%s.lat: %v is outside Bandung Raya", at, *venue.Lat)
		}
		if *venue.Lng < lngMin || *venue.Lng > lngMax {
			chk.errorf("%s.lng: %v is outside Bandung Raya", at, *venue.Lng)
		}
	}

	if venue.WhatsApp != nil && !waPattern.MatchString(*venue.WhatsApp) {
		chk.errorf("%s.whatsapp: %q is not +62... format", at, *venue.WhatsApp)
	}

	if venue.Tags == nil {
		chk.errorf("%s.tags: must be an array ([] when none)", at)
	}

	chk.lastVerified(at+".lastVerified", venue.LastVerified, venue.WhatsApp)

	if !venuesWithOffering[venue.ID] {
		chk.warnf("%s: no offering yet - %s", at, venue.Name)
	}
}

func (c *Catalog) checkOffering(chk *checker, i int, offeringIDs, venueIDs map[string]bool) {
	offering := &c.Offerings[i]
	at := fmt.Sprintf("offerings[%d]", i)

	chk.id(at+".id", offeringIDs, offering.ID, "of-")

	if offering.VenueID == nil {
		chk.errorf("%s: needs a venueId until organisers exist (Fase 3)", at)
	} else if !venueIDs[*offering.VenueID] {
		chk.errorf("%s.venueId: %q does not exist in venues.json", at, *offering.VenueID)
	}
	if offering.OrganizerID != nil {
		chk.errorf("%s.organizerId: must stay null until Fase 3 (organizers.json)", at)
	}

	chk.enum(at+".sport", offering.Sport, sports)
	chk.enum(at+".offeringType", offering.OfferingType, offeringTypes)
	chk.enumList(at+".audience", offering.Audience, audiences)
	chk.enumList(at+".joinMethod", offering.JoinMethod, joinMethods)

	if strings.TrimSpace(offering.Title) == "" {
		chk.errorf("%s.title: must be a non-empty string", at)
	}

	visibility := offering.PriceVisibility
	chk.enum(at+".priceVisibility", visibility, priceVisibilities)

	switch visibility {
	case VisibilityPublic:
		if offering.PriceMin == nil || *offering.PriceMin < 0 {
			chk.errorf("%s.priceMin: public price needs an integer >= 0", at)
		}
		if offering.PriceUnit == nil {
			chk.errorf("%s.priceUnit: public price needs a unit", at)
		} else {
			chk.enum(at+".priceUnit", *offering.PriceUnit, priceUnits)
		}
	case VisibilityOnRequest, VisibilityUnknown:
		if offering.PriceMin != nil || offering.PriceMax != nil {
			chk.errorf("%s: %s requires priceMin and priceMax to be null", at, visibility)
		}
		if offering.PriceUnit != nil {
			chk.errorf("%s.priceUnit: %s requires a null unit", at, visibility)
		}
	}

	if offering.PriceMin != nil && offering.PriceMax != nil && *offering.PriceMax < *offering.PriceMin {
		chk.errorf("%s: priceMax (%d) is below priceMin (%d)", at, *offering.PriceMax, *offering.PriceMin)
	}

	if offering.PriceUnit != nil && *offering.PriceUnit == "per-session-pack" && offering.SessionsPerPack == nil {
		chk.errorf("%s.sessionsPerPack: per-session-pack needs sessionsPerPack", at)
	}
	for _, field := range []struct {
		name  string
		value *int
	}{{"sessionsPerPack", offering.SessionsPerPack}, {"groupSize", offering.GroupSize}, {"durationMin", offering.DurationMin}} {
		if field.value != nil && *field.value < 1 {
			chk.errorf("%s.%s: must be an integer >= 1 or null", at, field.name)
		}
	}

	if offering.AgeMin != nil && *offering.AgeMin < 0 {
		chk.errorf("%s.ageMin: must be an integer >= 0 or null", at)
	}
	if offering.AgeMax != nil && *offering.AgeMax < 0 {
		chk.errorf("%s.ageMax: must be an integer >= 0 or null", at)
	}
	if offering.AgeMin != nil && offering.AgeMax != nil && *offering.AgeMin > *offering.AgeMax {
		chk.errorf("%s: ageMin (%d) is above ageMax (%d)", at, *offering.AgeMin, *offering.AgeMax)
	}
	if (offering.AgeMin != nil || offering.AgeMax != nil) &&
		!slices.Contains(offering.Audience, "anak") && !slices.Contains(offering.Audience, "remaja") {
		chk.warnf("%s: age fields set but audience has no anak/remaja", at)
	}

	if offering.Schedule == nil {
		chk.errorf("%s.schedule: must be an array ([] when unknown)", at)
	}
	for j, slot := range offering.Schedule {
		slotAt := fmt.Sprintf("%s.schedule[%d]", at, j)
		if len(slot.Days) == 0 {
			chk.errorf("%s.days: must be a non-empty array", slotAt)
		}
		for k, day := range slot.Days {
			chk.enum(fmt.Sprintf("%s.days[%d]", slotAt, k), day, weekdays)
		}
		if !timePattern.MatchString(slot.Start) {
			chk.errorf("%s.start: must be HH:MM", slotAt)
		}
		if !timePattern.MatchString(slot.End) {
			chk.errorf("%s.end: must be HH:MM", slotAt)
		}
		if timePattern.MatchString(slot.Start) && timePattern.MatchString(slot.End) && slot.Start >= slot.End {
			chk.errorf("%s: start must be before end", slotAt)
		}
	}

	chk.lastVerified(at+".lastVerified", offering.LastVerified, nil)
}
