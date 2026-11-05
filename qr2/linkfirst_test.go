package qr2

import (
	"testing"
	"time"

	"openpak/wfc-common/wfcapi"
)

func TestUnlinkedLoginIsNeverListed(t *testing.T) {
	now := time.Now()
	applyLogin(wfcapi.Login{ProfileID: 4242, Generation: "g1", Revision: 1, GameCode: "RSBE", Unlinked: true}, now)
	mutex.Lock()
	l := logins[4242]
	mutex.Unlock()
	if l == nil || !l.Unlinked {
		t.Fatalf("login projection lost Unlinked: %+v", l)
	}
	s := &Session{Authenticated: true, LastKeepAlive: now.UTC().Unix(), Data: map[string]string{"gamename": "x"}, login: l}
	mutex.Lock()
	sessions[999001] = s
	mutex.Unlock()
	defer func() {
		mutex.Lock()
		delete(sessions, 999001)
		delete(logins, 4242)
		mutex.Unlock()
	}()
	for _, row := range GetSessionServers() {
		if row["gamename"] == "x" {
			t.Fatal("unlinked session listed")
		}
	}
	l.Unlinked = false
	found := false
	for _, row := range GetSessionServers() {
		found = found || row["gamename"] == "x"
	}
	if !found {
		t.Fatal("linked session not listed")
	}
}
