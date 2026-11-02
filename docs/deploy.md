# Deploy: wfc-qr2

Use example.env with the existing WFC database and shared WFC_TOKEN_KEY_FILE.
Never rotate token material or change profile IDs during ordinary service releases.
Dependencies: WFC_GPCM_URL.

```sh
GOWORK=off go test -mod=vendor ./...
GOWORK=off go vet -mod=vendor ./...
docker build -t wfc-qr2 .
```

## Vendored library

Shared wire types, errors, API clients and legacy database helpers are vendored from
wfc-common. The image and tests build from this repository alone. After a shared contract
change, refresh `vendor/` with the sibling library available. Topology, failure/recovery
behaviour and migration: wfc-common `docs/service-split.md`.
