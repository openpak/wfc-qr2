package qr2

import (
	"net/http"
	"net/http/httptest"
	"openpak/wfc-common/service"
	"openpak/wfc-common/wfcapi"
	"strings"
	"sync"
	"testing"
	"time"
)

func resetRemoteState(t *testing.T) {
	t.Helper()
	mutex.Lock()
	logins = map[uint32]*LoginInfo{}
	retired = map[string]time.Time{}
	sessions = map[uint64]*Session{}
	mutex.Unlock()
	generations = sync.Map{}
	lastSnapshot.Store(0)
}
func TestReconciliationAndGenerationFencing(t *testing.T) {
	resetRemoteState(t)
	now := time.Now()
	v := wfcapi.Login{ProfileID: 7, Generation: "old", Revision: 1, GameCode: "ABCD"}
	applyLogin(v, now)
	marker := &Session{}
	mutex.Lock()
	logins[7].session = marker
	mutex.Unlock()
	applyLogin(v, now.Add(time.Second))
	if logins[7].session != marker {
		t.Fatal("same-generation reconcile discarded room association")
	}
	// Detach synthetic marker before exercising real removal behavior.
	mutex.Lock()
	logins[7].session = nil
	mutex.Unlock()
	newer := v
	newer.Generation = "new"
	newer.Revision = 2
	applyLogin(newer, now.Add(2*time.Second))
	removeLogin(7, "old")
	if logins[7] == nil || logins[7].Generation != "new" {
		t.Fatal("late logout removed a newer login")
	}
	applyLogin(v, now.Add(3*time.Second))
	if logins[7].Generation != "new" {
		t.Fatal("older snapshot replaced new generation")
	}
	removeLogin(7, "new")
	applyLogin(newer, now.Add(4*time.Second))
	if logins[7] != nil {
		t.Fatal("delayed snapshot resurrected logged-out generation")
	}
	fresh := newer
	fresh.Generation = "fresh"
	fresh.Revision = 3
	applyLogin(fresh, now.Add(5*time.Second))
	// An unavailable authority may retain a lease briefly, but may not retain ghosts indefinitely.
	syncLogins(nil, now, now.Add(10*time.Second), false)
	if logins[7] == nil {
		t.Fatal("temporary outage removed an unexpired session")
	}
	syncLogins(nil, now, now.Add(40*time.Second), false)
	if logins[7] != nil {
		t.Fatal("expired login survived authority outage")
	}
}
func TestSnapshotsDoNotLoseConcurrentNewLogins(t *testing.T) {
	resetRemoteState(t)
	start := time.Now()
	applyLogin(wfcapi.Login{ProfileID: 9, Generation: "concurrent", Revision: 1}, start.Add(time.Second))
	syncLogins(nil, start, start.Add(2*time.Second), true)
	if logins[9] == nil {
		t.Fatal("snapshot removed a login newer than snapshot request")
	}
	syncLogins(nil, start.Add(3*time.Second), start.Add(4*time.Second), true)
	if logins[9] != nil {
		t.Fatal("authoritative removal was ignored")
	}
}
func TestInternalAPIRejectsUnauthenticatedAndStaleMutations(t *testing.T) {
	resetRemoteState(t)
	key := strings.Repeat("k", 32)
	t.Setenv("WFC_INTERNAL_KEY", key)
	mux := http.NewServeMux()
	RegisterInternal(mux)
	handler := service.Guard(mux)
	request := func(op, body, auth string) int {
		r := httptest.NewRequest("POST", "/internal/v1/qr2/"+op, strings.NewReader(body))
		if auth != "" {
			r.Header.Set("X-WFC-Internal-Key", auth)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}
	if code := request("servers", "{}", ""); code != 401 {
		t.Fatal("missing internal auth", code)
	}
	if code := request("servers", "{}", key); code != 503 {
		t.Fatal("missing authority returned an apparently valid empty directory", code)
	}
	applyLogin(wfcapi.Login{ProfileID: 7, Generation: "new", Revision: 2}, time.Now())
	if code := request("status", `{"ProfileID":7,"Generation":"old","Status":"1"}`, key); code != 409 {
		t.Fatal("stale generation accepted", code)
	}
	if code := request("login", `{"Login":{"ProfileID":8,"Generation":"","Revision":1}}`, key); code != 400 {
		t.Fatal("invalid generation accepted", code)
	}
	if code := request("logins", "{} {}", key); code != 400 {
		t.Fatal("trailing request object accepted", code)
	}
}
func TestServerDirectoryResponseIsDetachedFromMutableState(t *testing.T) {
	resetRemoteState(t)
	sessions[1] = &Session{Authenticated: true, LastKeepAlive: time.Now().Unix(), Data: map[string]string{"gamename": "test"}}
	rows := GetSessionServers()
	rows[0]["gamename"] = "changed"
	if sessions[1].Data["gamename"] != "test" {
		t.Fatal("directory reader can modify QR2-owned state")
	}
}
