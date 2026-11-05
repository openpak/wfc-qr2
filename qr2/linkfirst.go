package qr2

import (
	"sync/atomic"
	"time"

	"openpak/wfc-common/logging"
)

// unlinkedDrops counts QR2 sessions of unlinked consoles dropped (link-first); summarised once
// a minute instead of a line per heartbeat.
var unlinkedDrops atomic.Int64

func init() {
	go func() {
		for range time.Tick(time.Minute) {
			if n := unlinkedDrops.Swap(0); n > 0 {
				logging.Notice("QR2", "link gate: refused", n, "heartbeats of unlinked consoles in the last minute")
			}
		}
	}()
}
