package catalog

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// Catalog is the in-memory catalog with lookup indexes built once at startup.
type Catalog struct {
	Venues    []Venue
	Offerings []Offering

	venuesByID       map[string]*Venue
	offeringsByVenue map[string][]Offering
}

// Load reads venues.json and offerings.json from fsys and indexes them.
// It does not validate: call Validate (and log Warnings) after loading.
func Load(fsys fs.FS) (*Catalog, error) {
	venues, err := readJSON[[]Venue](fsys, "venues.json")
	if err != nil {
		return nil, err
	}
	offerings, err := readJSON[[]Offering](fsys, "offerings.json")
	if err != nil {
		return nil, err
	}

	c := &Catalog{Venues: venues, Offerings: offerings}
	c.index()
	return c, nil
}

func readJSON[T any](fsys fs.FS, name string) (T, error) {
	var value T

	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return value, fmt.Errorf("read %s: %w", name, err)
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, fmt.Errorf("parse %s: %w", name, err)
	}
	return value, nil
}

func (c *Catalog) index() {
	c.venuesByID = make(map[string]*Venue, len(c.Venues))
	for i := range c.Venues {
		c.venuesByID[c.Venues[i].ID] = &c.Venues[i]
	}

	c.offeringsByVenue = make(map[string][]Offering, len(c.Venues))
	for _, offering := range c.Offerings {
		if offering.VenueID != nil {
			c.offeringsByVenue[*offering.VenueID] = append(c.offeringsByVenue[*offering.VenueID], offering)
		}
	}
}

// Venue returns the venue with the given id.
func (c *Catalog) Venue(id string) (*Venue, bool) {
	venue, ok := c.venuesByID[id]
	return venue, ok
}

// OfferingsByVenue returns the offerings attached to a venue, in file order.
func (c *Catalog) OfferingsByVenue(id string) []Offering {
	return c.offeringsByVenue[id]
}
