package qr2

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	emit "github.com/openpak/rewards-emit"
)

func grp(key string, uids ...uint64) sessionGroup {
	return sessionGroup{Key: key, Game: "mariokartwii", Players: uids}
}

func TestSessionClock(t *testing.T) {
	c := newSessionClock(3 * time.Minute)
	t0 := time.Now()
	// a lone player counts nothing, however long
	if d := c.tick(t0, []sessionGroup{grp("a:1", 10)}); len(d) != 0 {
		t.Fatal(d)
	}
	if d := c.tick(t0.Add(10*time.Minute), []sessionGroup{grp("a:1", 10)}); len(d) != 0 {
		t.Fatal(d)
	}
	// two players: nothing before the minimum, each once after it
	two := []sessionGroup{grp("b:2", 10, 20)}
	c.tick(t0, two)
	if d := c.tick(t0.Add(2*time.Minute), two); len(d) != 0 {
		t.Fatal(d)
	}
	d := c.tick(t0.Add(3*time.Minute), two)
	if len(d) != 2 || d[0].userID != 10 || d[1].userID != 20 {
		t.Fatalf("%v", d)
	}
	if d := c.tick(t0.Add(9*time.Minute), two); len(d) != 0 {
		t.Fatalf("re-emitted %v", d)
	}
	// a player who leaves resets; the one who stays alone also resets
	c.tick(t0.Add(10*time.Minute), []sessionGroup{grp("b:2", 10)})
	c.tick(t0.Add(11*time.Minute), two)
	if d := c.tick(t0.Add(13*time.Minute), two); len(d) != 0 {
		t.Fatalf("counted without an unbroken minimum %v", d)
	}
	if d := c.tick(t0.Add(14*time.Minute), two); len(d) != 2 {
		t.Fatalf("%v", d)
	}
	// unknown consoles are never reported
	c2 := newSessionClock(time.Minute)
	g3 := []sessionGroup{grp("c:3", 0, 30)}
	c2.tick(t0, g3)
	if d := c2.tick(t0.Add(time.Minute), g3); len(d) != 1 || d[0].userID != 30 {
		t.Fatalf("%v", d)
	}
}

func TestSessionSnapshot(t *testing.T) {
	mutex.Lock()
	old := groups
	groups = map[string]*Group{}
	a, b := &Session{login: &LoginInfo{UserID: 5}}, &Session{login: &LoginInfo{UserID: 6}}
	groups["AAAAAA"] = &Group{GroupName: "AAAAAA", GameName: "mariokartwii", CreateTime: time.Unix(1000, 0), players: map[*Session]bool{a: true, b: true}}
	groups["BBBBBB"] = &Group{GroupName: "BBBBBB", GameName: "other", CreateTime: time.Unix(1000, 0), players: map[*Session]bool{a: true, b: true}}
	mutex.Unlock()
	defer func() { mutex.Lock(); groups = old; mutex.Unlock() }()
	s := sessionSnapshot()
	if len(s) != 1 || s[0].Key != "AAAAAA:1000" || len(s[0].Players) != 2 {
		t.Fatalf("%+v", s)
	}
}

func TestWatcherReachesRewards(t *testing.T) {
	var mu sync.Mutex
	var got []map[string]any
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var in struct{ Events []map[string]any }
		_ = json.Unmarshal(b, &in)
		mu.Lock()
		got = append(got, in.Events...)
		mu.Unlock()
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer sink.Close()
	rw := emit.New(emit.Config{URL: sink.URL, Key: "k", Source: "t", Flush: 10 * time.Millisecond})

	var smu sync.Mutex
	cur := []sessionGroup{{Key: "XYZXYZ:77", Game: "mariokartds", Players: []uint64{1}}}
	snap := func() []sessionGroup { smu.Lock(); defer smu.Unlock(); return append([]sessionGroup(nil), cur...) }
	stop := watchSessions(rw, snap, 400*time.Millisecond, 50*time.Millisecond)

	time.Sleep(300 * time.Millisecond)
	mu.Lock()
	if len(got) != 0 {
		t.Fatalf("a lobby of one reported %v", got)
	}
	mu.Unlock()
	smu.Lock()
	cur = []sessionGroup{{Key: "XYZXYZ:77", Game: "mariokartds", Players: []uint64{1, 2}}}
	smu.Unlock()
	time.Sleep(900 * time.Millisecond)
	stop()
	rw.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	subs := map[any]bool{}
	for _, e := range got {
		id := e["identity"].(map[string]any)
		keys := e["keys"].([]any)
		if e["metric"] != "kart_sessions.mkds" || id["namespace"] != "wfc" || keys[0] != "XYZXYZ:77" {
			t.Fatalf("%v", e)
		}
		if e["event_id"] != "mkds:XYZXYZ:77:"+id["subject"].(string) {
			t.Fatalf("%v", e)
		}
		subs[id["subject"]] = true
	}
	if len(got) != 2 || len(subs) != 2 {
		t.Fatalf("want one event per player, got %v", got)
	}
}
