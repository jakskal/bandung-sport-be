# Bandung Sport API

Read-only Go API for [Bandung Sport](https://github.com/jakskal/bandung-sport-web):
venues and offerings for sports in Bandung, with real prices.

Data lives in `data/*.json` and is embedded into the binary (`go:embed`) —
no database. Updating data means editing the JSON and redeploying.

## Layout

```
cmd/api/            HTTP entrypoint (S7)
internal/catalog/   model, loader, validation, tier + price helpers
data/               venues.json, offerings.json (single source of data)
```

## Run

```sh
go run ./cmd/api      # S7
```

## Test

```sh
make test             # go test ./...
make vet              # go vet ./...
```

`TestDataIsValid` fails when the embedded data breaks a rule from the model
spec, so bad data cannot ship.
