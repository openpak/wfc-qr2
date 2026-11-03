# Wii/DS QR2 Server

OpenPak's Wii/DS QR2 Server.

| | |
| --- | --- |
| Network | Wii, DS |
| Role | QR2 server registration, heartbeat, server directory, room state and generation-fenced GPCM projections |
| Depends on | `WFC_GPCM_URL` |
| Image | `ghcr.io/openpak/wfc-qr2` |
| Based on | WiiLink WFC (AGPL-3.0) |
| Licence | AGPL-3.0-only |

## What it serves

| Endpoint | What |
| --- | --- |
| `GET /healthz` | Health check, unauthenticated |
| Protocol and internal paths | Require `WFC_INTERNAL_KEY` |
| `27900/udp` | QR2 registration and heartbeat |

## Rewards

Set `REWARDS_EVENTS_URL` and `REWARDS_EVENTS_KEY` (or `REWARDS_EVENTS_KEY_FILE`) to report Mario Kart
Wii and DS online sessions; `REWARDS_SESSION_MIN` (default `3m`) is how long two players must share
a group. See [Deploy](docs/deploy.md).

## Build

```sh
docker build -t wfc-qr2 .
```

## Docs

- [Protocol](docs/protocol.md)
- [Deploy](docs/deploy.md)
- [Provenance](docs/provenance.md)

## Licence

AGPL-3.0-only. See [LICENSE](LICENSE).
