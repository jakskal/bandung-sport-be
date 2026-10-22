package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jakskal/bandung-sport-be/data"
	"github.com/jakskal/bandung-sport-be/internal/catalog"
)

const allowedOrigin = "http://localhost:5173"

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	cat, err := catalog.Load(data.FS)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	return New(cat, []string{allowedOrigin}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestRoutes(t *testing.T) {
	handler := newTestHandler(t)

	tests := []struct {
		name       string
		method     string
		path       string
		origin     string
		wantStatus int
		check      func(t *testing.T, res *httptest.ResponseRecorder)
	}{
		{
			name:       "healthz reports counts",
			method:     http.MethodGet,
			path:       "/healthz",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, res *httptest.ResponseRecorder) {
				var body struct {
					Status    string `json:"status"`
					Venues    int    `json:"venues"`
					Offerings int    `json:"offerings"`
				}
				decode(t, res, &body)
				if body.Status != "ok" || body.Venues != 22 || body.Offerings < 8 {
					t.Fatalf("body = %+v", body)
				}
			},
		},
		{
			name:       "list returns every venue",
			method:     http.MethodGet,
			path:       "/api/venues",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, res *httptest.ResponseRecorder) {
				if got := res.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
					t.Fatalf("Content-Type = %q", got)
				}

				var venues []VenueSummary
				decode(t, res, &venues)
				if len(venues) != 22 {
					t.Fatalf("len(venues) = %d, want 22", len(venues))
				}
				ftl := venues[0]
				if ftl.ID != "v-ftl-gym-dago-bandung" {
					t.Fatalf("first id = %q", ftl.ID)
				}
				if ftl.Tier == nil || *ftl.Tier != catalog.TierSedang {
					t.Fatalf("tier = %v, want Sedang", ftl.Tier)
				}
				if ftl.PriceFrom == nil || ftl.PriceFrom.PriceMin != 500000 {
					t.Fatalf("priceFrom = %+v", ftl.PriceFrom)
				}
				if ftl.OfferingCount != 1 {
					t.Fatalf("offeringCount = %d", ftl.OfferingCount)
				}
			},
		},
		{
			name:       "detail includes contact and offerings",
			method:     http.MethodGet,
			path:       "/api/venues/v-ftl-gym-dago-bandung",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, res *httptest.ResponseRecorder) {
				var venue VenueDetail
				decode(t, res, &venue)
				if venue.ID != "v-ftl-gym-dago-bandung" {
					t.Fatalf("id = %q", venue.ID)
				}
				if len(venue.Offerings) != 1 {
					t.Fatalf("len(offerings) = %d, want 1", len(venue.Offerings))
				}
				if venue.WhatsApp == nil {
					t.Fatal("whatsapp is nil")
				}
			},
		},
		{
			name:       "unknown venue is a json 404",
			method:     http.MethodGet,
			path:       "/api/venues/v-nope",
			wantStatus: http.StatusNotFound,
			check:      checkErrorBody,
		},
		{
			name:       "unknown path is a json 404",
			method:     http.MethodGet,
			path:       "/api/nope",
			wantStatus: http.StatusNotFound,
			check:      checkErrorBody,
		},
		{
			name:       "post is not allowed",
			method:     http.MethodPost,
			path:       "/api/venues",
			wantStatus: http.StatusMethodNotAllowed,
			check: func(t *testing.T, res *httptest.ResponseRecorder) {
				if got := res.Header().Get("Allow"); got != "GET, OPTIONS" {
					t.Fatalf("Allow = %q", got)
				}
				var body map[string]string
				decode(t, res, &body)
				if body["error"] != "method not allowed" {
					t.Fatalf("body = %+v", body)
				}
			},
		},
		{
			name:       "allowed origin gets cors header",
			method:     http.MethodGet,
			path:       "/api/venues",
			origin:     allowedOrigin,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, res *httptest.ResponseRecorder) {
				if got := res.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
					t.Fatalf("Access-Control-Allow-Origin = %q", got)
				}
			},
		},
		{
			name:       "other origin gets no cors header",
			method:     http.MethodGet,
			path:       "/api/venues",
			origin:     "https://evil.example",
			wantStatus: http.StatusOK,
			check: func(t *testing.T, res *httptest.ResponseRecorder) {
				if got := res.Header().Get("Access-Control-Allow-Origin"); got != "" {
					t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
				}
			},
		},
		{
			name:       "preflight returns 204",
			method:     http.MethodOptions,
			path:       "/api/venues",
			origin:     allowedOrigin,
			wantStatus: http.StatusNoContent,
			check: func(t *testing.T, res *httptest.ResponseRecorder) {
				if got := res.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
					t.Fatalf("Access-Control-Allow-Origin = %q", got)
				}
				if got := res.Header().Get("Access-Control-Allow-Methods"); got != "GET, OPTIONS" {
					t.Fatalf("Access-Control-Allow-Methods = %q", got)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req)

			if res.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", res.Code, tc.wantStatus)
			}
			if tc.check != nil {
				tc.check(t, res)
			}
		})
	}
}

func checkErrorBody(t *testing.T, res *httptest.ResponseRecorder) {
	t.Helper()

	var body map[string]string
	decode(t, res, &body)
	if body["error"] != "not found" {
		t.Fatalf("body = %+v", body)
	}
}

func decode(t *testing.T, res *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.NewDecoder(res.Body).Decode(target); err != nil {
		t.Fatalf("decode body: %v (body: %q)", err, res.Body.String())
	}
}
