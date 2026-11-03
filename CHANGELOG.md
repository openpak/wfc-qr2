# Changelog

## v0.2.0

- Rewards: reports Mario Kart Wii (`kart_sessions.mkwii`) and Mario Kart DS (`kart_sessions.mkds`)
  online sessions. A player counts once, per group, after sharing a matchmaking group with at
  least one other player for `REWARDS_SESSION_MIN` (default 3m) without a break. The identity is
  the console user id (namespace `wfc`). Off unless `REWARDS_EVENTS_URL` and a key are set.
- `LoginInfo.UserID`, carried from the GPCM session projection and kept in the saved logins.
