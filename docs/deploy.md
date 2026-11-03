# Deploy: wfc-qr2

Use example.env with the existing WFC database and shared WFC_TOKEN_KEY_FILE.
Never rotate token material or change profile IDs during ordinary service releases.
Dependencies: WFC_GPCM_URL.

## Rewards (optional)

Online Mario Kart Wii and Mario Kart DS sessions are reported to the rewards service as
`kart_sessions.mkwii` and `kart_sessions.mkds`. Unset means off.

| Variable | Meaning |
| --- | --- |
| `REWARDS_EVENTS_URL` | Base URL of rewards, for example `http://openpak-rewards:20180` |
| `REWARDS_EVENTS_KEY` | The events key (or `REWARDS_EVENTS_KEY_FILE`, a file holding it) |
| `REWARDS_SESSION_MIN` | Time two players must share a group to count, default `3m` |

A session is a player in a server-side matchmaking group with at least one other player for the
minimum, unbroken. Identity is the console user id in namespace `wfc`.

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
