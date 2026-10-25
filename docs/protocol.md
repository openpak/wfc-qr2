# Protocol: wfc-qr2

Independent OpenPak Wii/DS protocol service: QR2 server registration, heartbeat, server directory, room state and generation-fenced GPCM projections.
Own repository, binary, image and release; AGPL-3.0, derived from WiiLink WFC.

Internal HTTP defaults to WFC_HTTP_ADDR=:20203. GET /healthz is unauthenticated;
protocol/internal paths require WFC_INTERNAL_KEY. Console HTTP reaches this owner
through wfc-nas; TCP/UDP ports belong directly to their protocol's container.
Runtime frontend/backend net/rpc endpoints stay on loopback inside each TCP
service container. Restarting another protocol does not restart these sockets.
