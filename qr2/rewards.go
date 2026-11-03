package qr2

// Online sessions for rewards: a player counts once they have shared a matchmaking group with at
// least one other player for SessionMin of unbroken time. The group is the server-side proof of
// presence in a room, not a finished race. One event per (group, player); the group name plus its
// creation time is the event id and the distinct key, because group names are random and recycled.
// Players are identified by the console user id (the id the website links), not the profile id.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	emit "github.com/openpak/rewards-emit"
)

// sessionMetrics maps a game name to its rewards metric and event id prefix.
var sessionMetrics = map[string][2]string{
	"mariokartwii":  {"kart_sessions.mkwii", "mkwii"},
	"mariokartds":   {"kart_sessions.mkds", "mkds"},
	"mariokartdsam": {"kart_sessions.mkds", "mkds"},
	"mariokartkods": {"kart_sessions.mkds", "mkds"},
}

// SessionMin is how long two players must share a group before it counts.
func SessionMin() time.Duration {
	if d, err := time.ParseDuration(os.Getenv("REWARDS_SESSION_MIN")); err == nil && d > 0 {
		return d
	}
	return 3 * time.Minute
}

// sessionGroup is one group as the watcher sees it. Players holds the console user id of every
// member; 0 is a member whose console is unknown (counted as present, never reported).
type sessionGroup struct {
	Key     string // groupname:createUnix
	Game    string
	Created int64
	Players []uint64
}

type sessionPair struct {
	g      sessionGroup
	userID uint64
}

type sessionClock struct {
	min     time.Duration
	since   map[string]time.Time
	emitted map[string]bool
}

func newSessionClock(min time.Duration) *sessionClock {
	return &sessionClock{min: min, since: map[string]time.Time{}, emitted: map[string]bool{}}
}

// clockState is the clock as stored in Postgres through rewards-emit (SaveState / LoadState), so a
// restart loses no half-served session.
type clockState struct {
	Since   map[string]int64 `json:"since"` // unix nanoseconds
	Emitted []string         `json:"emitted"`
}

func (c *sessionClock) state() []byte {
	st := clockState{Since: map[string]int64{}, Emitted: []string{}}
	for k, t := range c.since {
		st.Since[k] = t.UnixNano()
	}
	for k := range c.emitted {
		st.Emitted = append(st.Emitted, k)
	}
	sort.Strings(st.Emitted)
	b, _ := json.Marshal(st)
	return b
}

func (c *sessionClock) restore(b []byte) error {
	var st clockState
	if err := json.Unmarshal(b, &st); err != nil {
		return err
	}
	for k, n := range st.Since {
		c.since[k] = time.Unix(0, n)
	}
	for _, k := range st.Emitted {
		c.emitted[k] = true
	}
	return nil
}

const clockStateName = "session_clock"

// tick takes the groups now and returns the (group, player) pairs that just crossed the minimum.
// A group with fewer than two players resets its players.
func (c *sessionClock) tick(now time.Time, gs []sessionGroup) (done []sessionPair) {
	seen := map[string]bool{}
	for _, g := range gs {
		if len(g.Players) < 2 {
			continue
		}
		for _, uid := range g.Players {
			if uid == 0 {
				continue
			}
			k := g.Key + "/" + strconv.FormatUint(uid, 10)
			seen[k] = true
			if _, ok := c.since[k]; !ok {
				c.since[k] = now
			}
			if !c.emitted[k] && now.Sub(c.since[k]) >= c.min {
				c.emitted[k] = true
				done = append(done, sessionPair{g, uid})
			}
		}
	}
	for k := range c.since {
		if !seen[k] {
			delete(c.since, k)
		}
	}
	for k := range c.emitted {
		if !seen[k] {
			delete(c.emitted, k)
		}
	}
	sort.Slice(done, func(i, j int) bool {
		if done[i].g.Key != done[j].g.Key {
			return done[i].g.Key < done[j].g.Key
		}
		return done[i].userID < done[j].userID
	})
	return done
}

// sessionSnapshot copies the rewarded games' groups under the package mutex.
func sessionSnapshot() []sessionGroup {
	mutex.Lock()
	defer mutex.Unlock()
	var out []sessionGroup
	for _, g := range groups {
		if _, ok := sessionMetrics[g.GameName]; !ok {
			continue
		}
		sg := sessionGroup{
			Key:     g.GroupName + ":" + strconv.FormatInt(g.CreateTime.Unix(), 10),
			Game:    g.GameName,
			Created: g.CreateTime.Unix(),
		}
		for s := range g.players {
			var uid uint64
			if s.login != nil {
				uid = s.login.UserID
			}
			sg.Players = append(sg.Players, uid)
		}
		out = append(out, sg)
	}
	return out
}

func emitSession(rw *emit.Client, p sessionPair) {
	m := sessionMetrics[p.g.Game]
	rw.Event(emit.Event{
		ID:     fmt.Sprintf("%s:%s:%d", m[1], p.g.Key, p.userID),
		Metric: m[0],
		Player: &emit.Identity{Namespace: "wfc", Subject: strconv.FormatUint(p.userID, 10)},
		Keys:   []string{p.g.Key},
	})
}

var (
	rewardsClient *emit.Client
	rewardsStop   func()
)

// watchSessions reports online sessions to rewards until stop is called. With rewards
// unconfigured it does nothing.
func watchSessions(rw *emit.Client, snapshot func() []sessionGroup, min, every time.Duration) (stop func()) {
	if !rw.Enabled() {
		return func() {}
	}
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		clock := newSessionClock(min)
		if b, ok := rw.LoadState(clockStateName); ok {
			if err := clock.restore(b); err != nil {
				log.Printf("wfc-qr2: session clock state unreadable, starting empty: %v", err)
			} else {
				log.Printf("wfc-qr2: session clock restored (%d running, %d counted)", len(clock.since), len(clock.emitted))
			}
		}
		var saved []byte
		save := func() {
			if b := clock.state(); !bytes.Equal(b, saved) {
				rw.SaveState(clockStateName, b)
				saved = b
			}
		}
		defer save()
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			for _, p := range clock.tick(time.Now(), snapshot()) {
				emitSession(rw, p)
			}
			save()
			select {
			case <-t.C:
			case <-done:
				return
			}
		}
	}()
	return func() { close(done); <-finished }
}

func startRewards() {
	rewardsClient = emit.FromEnv("wfc-qr2")
	rewardsStop = watchSessions(rewardsClient, sessionSnapshot, SessionMin(), 20*time.Second)
}

func stopRewards() {
	if rewardsStop != nil {
		rewardsStop()
		rewardsStop = nil
	}
	if rewardsClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		rewardsClient.Close(ctx)
		cancel()
		rewardsClient = nil
	}
}
