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

## Deploy

Production: **https://bandung-sport-be.vercel.app** (Vercel Go preset,
`vercel.json` sets `"framework": "go"`). Hobby plan, no database: the JSON in
`data/` is embedded at build time, so updating data means commit + push.

| Setting | Value |
|---|---|
| Framework Preset | Go (from `vercel.json`) |
| Root Directory | `./` |
| Environment | `ALLOWED_ORIGINS=http://localhost:5173,https://bandung-sport-web.vercel.app` |
| Region | Singapore (`sin1`) — default is `iad1`, ~200 ms extra from Bandung |

```sh
curl https://bandung-sport-be.vercel.app/healthz
# {"offerings":8,"status":"ok","venues":22}
```

Adding a new frontend origin (preview deployments) means editing
`ALLOWED_ORIGINS` and redeploying: env changes only apply to new deployments.

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
