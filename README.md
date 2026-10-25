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

## Build

```sh
docker build -t wfc-qr2 .
```

## Docs

- [Protocol](docs/protocol.md)
- [Evidence](docs/evidence.md)
- [Deploy](docs/deploy.md)
- [Notes](docs/notes.md)

## Licence

AGPL-3.0-only. See [LICENSE](LICENSE).
