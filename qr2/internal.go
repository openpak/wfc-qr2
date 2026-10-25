package qr2

import (
	"context"
	"log"
	"net/http"
	"openpak/wfc-common/gpcmclient"
	"openpak/wfc-common/wfcapi"
	"sync"
	"sync/atomic"
	"time"
)

var generations sync.Map
var lastSnapshot atomic.Int64
var retired = map[string]time.Time{}

// applyLogin is idempotent for the same session generation and preserves its room pointer.
func applyLogin(v wfcapi.Login, now time.Time) {
	mutex.Lock()
	defer mutex.Unlock()
	if until, ok := retired[v.Generation]; ok && until.After(now) {
		return
	}
	for g, until := range retired {
		if !until.After(now) {
			delete(retired, g)
		}
	}
	old := logins[v.ProfileID]
	if old != nil && v.Revision < old.Revision {
		return
	}
	if old != nil && old.Generation != v.Generation {
		if old.session != nil {
			removeSession(makeLookupAddr(old.session.Addr.String()))
		}
		old = nil
	}
	if old == nil {
		old = &LoginInfo{}
		logins[v.ProfileID] = old
	}
	old.ProfileID = v.ProfileID
	old.Generation = v.Generation
	old.Revision = v.Revision
	old.GameCode = v.GameCode
	old.InGameName = v.InGameName
	old.ConsoleFriendCode = v.ConsoleFriendCode
	old.FriendKeyGame = v.FriendKeyGame
	old.GPPublicIP = v.GPPublicIP
	old.NeedsExploit = v.NeedsExploit
	old.DeviceAuthenticated = v.DeviceAuthenticated
	old.Restricted = v.Restricted
	old.RemoteUntil = now.Add(30 * time.Second).UnixNano()
	old.UpdatedAt = now.UnixNano()
	generations.Store(v.ProfileID, v.Generation)
}
func removeLogin(pid uint32, generation string) {
	mutex.Lock()
	defer mutex.Unlock()
	old := logins[pid]
	if old == nil || old.Generation != generation {
		return
	}
	if old.session != nil {
		removeSession(makeLookupAddr(old.session.Addr.String()))
	}
	retired[generation] = time.Now().Add(time.Minute)
	delete(logins, pid)
	generations.Delete(pid)
}
func syncLogins(rows []wfcapi.Session, snapshotStart, now time.Time, authoritative bool) {
	seen := map[uint32]bool{}
	for _, r := range rows {
		seen[r.Login.ProfileID] = true
		applyLogin(r.Login, now)
	}
	type removal struct {
		pid        uint32
		generation string
	}
	var remove []removal
	mutex.Lock()
	for pid, l := range logins {
		if l.RemoteUntil <= now.UnixNano() || (authoritative && !seen[pid] && l.UpdatedAt <= snapshotStart.UnixNano()) {
			remove = append(remove, removal{pid, l.Generation})
		}
	}
	mutex.Unlock()
	for _, r := range remove {
		removeLogin(r.pid, r.generation)
	}
}
func RegisterInternal(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/qr2/{op}", func(w http.ResponseWriter, r *http.Request) {
		var in wfcapi.QR2Request
		if !wfcapi.Decode(w, r, &in) {
			return
		}
		out := wfcapi.QR2Response{}
		op := r.PathValue("op")
		switch op {
		case "search-id", "status", "tell-address", "reservation-allowed", "reservation-ok", "user", "selection", "device-authenticated":
			mutex.Lock()
			l := logins[in.ProfileID]
			valid := l != nil && l.Generation == in.Generation && l.RemoteUntil > time.Now().UnixNano()
			mutex.Unlock()
			if !valid {
				http.Error(w, "stale or unavailable player session", 409)
				return
			}
		}

		switch r.PathValue("op") {
		case "servers":
			if time.Since(time.Unix(0, lastSnapshot.Load())) > 30*time.Second {
				http.Error(w, "session authority unavailable", 503)
				return
			}
			out.Servers = GetSessionServers()
		case "groups":
			if time.Since(time.Unix(0, lastSnapshot.Load())) > 30*time.Second {
				http.Error(w, "session authority unavailable", 503)
				return
			}
			out.Groups = GetGroups(in.Games, in.Groups, in.Sorted)
		case "logins":
			mutex.Lock()
			for _, l := range logins {
				out.Logins = append(out.Logins, wfcapi.Login{ProfileID: l.ProfileID, Generation: l.Generation, Revision: l.Revision, DeviceAuthenticated: l.DeviceAuthenticated})
			}
			mutex.Unlock()
		case "search-id":
			out.SearchID = GetSearchID(in.Address)
		case "login":
			if in.Login == nil || in.Login.ProfileID == 0 || in.Login.Generation == "" || len(in.Login.Generation) > 64 || in.Login.Revision <= 0 {
				http.Error(w, "invalid session", 400)
				return
			}
			applyLogin(*in.Login, time.Now())
		case "logout":
			removeLogin(in.ProfileID, in.Generation)
		case "device-authenticated":
			mutex.Lock()
			if l := logins[in.ProfileID]; l != nil && l.Generation == in.Generation {
				l.DeviceAuthenticated = true
				if l.session != nil {
					l.session.Data["+deviceauth"] = "1"
				}
			}
			mutex.Unlock()
		case "status":
			ProcessGPStatusUpdate(in.ProfileID, in.Address, in.Status)
		case "tell-address":
			ProcessGPTellAddr(in.ProfileID, in.Address, in.DestProfileID, in.DestAddress)
		case "reservation-allowed":
			out.Allowed = CheckGPReservationAllowed(in.Address, in.ProfileID, in.DestProfileID, in.JoinType)
		case "reservation-ok":
			out.Accepted = ProcessGPResvOK(in.MatchVersion, in.Reservation, in.ResvOK, in.Address, in.ProfileID, in.DestAddress, in.DestProfileID)
		case "user":
			ProcessUSER(in.ProfileID, in.Address, in.Packet)
		case "selection":
			ProcessMKWSelectRecord(in.ProfileID, in.Key, in.Value)
		case "client-message":
			if len(in.Packet) > 4096 {
				http.Error(w, "packet too large", 400)
				return
			}
			SendClientMessage(in.SenderIP, in.SearchID, in.Packet)
		case "natneg-report":
			ProcessNATNEGReport(in.Result, in.IP1, in.IP2)
		default:
			http.NotFound(w, r)
			return
		}
		wfcapi.Reply(w, out)
	})
}

// Callbacks are queued without waiting while the room lock is held. Generation
// checking at GPCM prevents a delayed room error from kicking a newer login.
func StartRemote(ctx context.Context) {
	type event struct {
		pid                uint32
		generation, reason string
	}
	queue := make(chan event, 256)
	SetGPErrorCallback(func(pid uint32, reason string) {
		g, ok := generations.Load(pid)
		if !ok {
			return
		}
		select {
		case queue <- event{pid, g.(string), reason}:
		default:
			log.Printf("QR2 callback queue full; retaining player session")
		}
	})
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-queue:
				for attempt := 0; attempt < 4; attempt++ {
					if err := gpcmclient.Callback(ctx, e.pid, e.generation, e.reason); err == nil {
						break
					} else {
						log.Printf("QR2 callback delivery failed: %v", err)
					}
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
					}
				}
			}
		}
	}()
	go func() {
		for ctx.Err() == nil {
			start := time.Now()
			rows, err := gpcmclient.Sessions(ctx)
			syncLogins(rows, start, time.Now(), err == nil)
			if err == nil {
				lastSnapshot.Store(time.Now().UnixNano())
			}
			if err != nil {
				log.Printf("QR2 session reconciliation failed: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}()
}
