# Bandung Sport API

Read-only Go API for [Bandung Sport](https://github.com/jakskal/bandung-sport-web):
venues and offerings for sports in Bandung, with real prices.

Data lives in `data/*.json` and is embedded into the binary (`go:embed`) —
no database. Updating data means editing the JSON and redeploying.

## Endpoints

| Method & path | Response |
|---|---|
| `GET /healthz` | `{"status":"ok","venues":22,"offerings":8}` |
| `GET /api/venues` | `VenueSummary[]` |
| `GET /api/venues/{id}` | `VenueDetail` · `404` |

Errors are `{"error":"not found"}`. No filtering, auth or versioning: the
catalog is small enough to filter in the browser.

## Layout

```
cmd/api/            HTTP entrypoint
internal/catalog/   model, loader, validation, tier + price helpers
internal/httpapi/   routes, JSON views, CORS, handler tests
data/               venues.json, offerings.json (single source of data)
```

## Run

```sh
make run              # go run ./cmd/api, listens on :8080
```

Environment:

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `8080` | Set by Vercel in production. |
| `ALLOWED_ORIGINS` | empty | Comma separated. Empty means browsers get no CORS header, so the web app cannot call the API. Local: `http://localhost:5173`. |

```sh
ALLOWED_ORIGINS=http://localhost:5173 make run
curl localhost:8080/api/venues
```

## Test

```sh
make test             # go test ./...
make vet              # go vet ./...
```

`TestDataIsValid` fails when the embedded data breaks a rule from the model
spec, so bad data cannot ship.

## Docker

```sh
docker build -t bandung-sport-be .
docker run -p 8080:8080 -e ALLOWED_ORIGINS=http://localhost:5173 bandung-sport-be
```

Multi-stage: `golang:1.23-alpine` builder, `gcr.io/distroless/static:nonroot`
runtime, `CGO_ENABLED=0`. Vercel deploys the same code with its Go preset;
the Dockerfile stays for local runs and as a fallback host.
