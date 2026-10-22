// Package httpapi exposes the catalog as a small read-only JSON API.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jakskal/bandung-sport-be/internal/catalog"
)

// Server holds the dependencies shared by all handlers.
type Server struct {
	cat     *catalog.Catalog
	origins map[string]bool
	log     *slog.Logger
}

// New returns the HTTP handler: routes plus CORS.
func New(cat *catalog.Catalog, allowedOrigins []string, log *slog.Logger) http.Handler {
	s := &Server{cat: cat, origins: make(map[string]bool, len(allowedOrigins)), log: log}
	for _, origin := range allowedOrigins {
		s.origins[origin] = true
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/venues", s.listVenues)
	mux.HandleFunc("GET /api/venues/{id}", s.getVenue)

	// Method-less patterns catch everything the GET routes do not, so the
	// answer stays JSON instead of the mux's plain-text 405/404.
	mux.HandleFunc("/healthz", s.methodNotAllowed)
	mux.HandleFunc("/api/venues", s.methodNotAllowed)
	mux.HandleFunc("/api/venues/{id}", s.methodNotAllowed)
	mux.HandleFunc("/", s.notFound)

	return s.cors(mux)
}

// VenueSummary is one card in the list.
type VenueSummary struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Area          string         `json:"area"`
	Sports        []string       `json:"sports"`
	Lat           *float64       `json:"lat"`
	Lng           *float64       `json:"lng"`
	Tier          *catalog.Tier  `json:"tier"`
	PriceFrom     *catalog.Price `json:"priceFrom"`
	OfferingCount int            `json:"offeringCount"`
	LastVerified  *string        `json:"lastVerified"`
}

// VenueDetail is a summary plus contact details and all offerings.
type VenueDetail struct {
	VenueSummary
	Address   *string            `json:"address"`
	WhatsApp  *string            `json:"whatsapp"`
	Instagram *string            `json:"instagram"`
	Website   *string            `json:"website"`
	SourceURL *string            `json:"sourceUrl"`
	Tags      []string           `json:"tags"`
	Offerings []catalog.Offering `json:"offerings"`
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"venues":    len(s.cat.Venues),
		"offerings": len(s.cat.Offerings),
	})
}

func (s *Server) listVenues(w http.ResponseWriter, _ *http.Request) {
	summaries := make([]VenueSummary, 0, len(s.cat.Venues))
	for _, venue := range s.cat.Venues {
		summaries = append(summaries, s.summary(venue))
	}
	s.writeJSON(w, http.StatusOK, summaries)
}

func (s *Server) getVenue(w http.ResponseWriter, r *http.Request) {
	venue, ok := s.cat.Venue(r.PathValue("id"))
	if !ok {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}

	offerings := s.cat.OfferingsByVenue(venue.ID)
	if offerings == nil {
		offerings = []catalog.Offering{}
	}

	s.writeJSON(w, http.StatusOK, VenueDetail{
		VenueSummary: s.summary(*venue),
		Address:      venue.Address,
		WhatsApp:     venue.WhatsApp,
		Instagram:    venue.Instagram,
		Website:      venue.Website,
		SourceURL:    venue.SourceURL,
		Tags:         venue.Tags,
		Offerings:    offerings,
	})
}

func (s *Server) notFound(w http.ResponseWriter, _ *http.Request) {
	s.writeError(w, http.StatusNotFound, "not found")
}

func (s *Server) methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Allow", "GET, OPTIONS")
	s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (s *Server) summary(venue catalog.Venue) VenueSummary {
	return VenueSummary{
		ID:            venue.ID,
		Name:          venue.Name,
		Area:          venue.Area,
		Sports:        venue.Sports,
		Lat:           venue.Lat,
		Lng:           venue.Lng,
		Tier:          s.cat.Tier(venue),
		PriceFrom:     s.cat.PriceFrom(venue),
		OfferingCount: len(s.cat.OfferingsByVenue(venue.ID)),
		LastVerified:  venue.LastVerified,
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		s.log.Error("write response", "err", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}
