# Changelog

## v0.3.2 - 2026-11-03

- Internal `/internal/` polls are counted and summarised once a minute instead of logged per call (wfc-common v0.3.1).

## v0.3.1 - 2026-11-03

- Every qr2 message now logs one line to stderr, `qr2 cmd <name> -> <result> <ms>ms` (verb and result only, never payload values) (every UDP packet type, including heartbeat and keepalive). Internal HTTP logs as above through wfc-common v0.3.0 `httplog`.

## v0.3.0 - 2026-11-03

- The counted-session clock is kept in Postgres through rewards-emit v0.3.0 (`REWARDS_EMIT_DATABASE_URL`), restored on start and saved after every change, so a restart loses no half-served session and counts no player twice. Rewards events also queue in Postgres.

## v0.2.0

- Rewards: reports Mario Kart Wii (`kart_sessions.mkwii`) and Mario Kart DS (`kart_sessions.mkds`)
  online sessions. A player counts once, per group, after sharing a matchmaking group with at
  least one other player for `REWARDS_SESSION_MIN` (default 3m) without a break. The identity is
  the console user id (namespace `wfc`). Off unless `REWARDS_EVENTS_URL` and a key are set.
- `LoginInfo.UserID`, carried from the GPCM session projection and kept in the saved logins.
