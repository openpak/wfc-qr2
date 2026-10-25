package wfcapi

import "time"

type MiiInfo struct {
	MiiData string `json:"data"`
	MiiName string `json:"name"`
}

type PlayerInfo struct {
	Count      string `json:"count"`
	ProfileID  string `json:"pid"`
	InGameName string `json:"name"`
	ConnMap    string `json:"conn_map"`
	ConnFail   string `json:"conn_fail"`
	Suspend    string `json:"suspend"`

	// Mario Kart Wii-specific fields
	FriendCode string    `json:"fc,omitempty"`
	VersusELO  string    `json:"ev,omitempty"`
	BattleELO  string    `json:"eb,omitempty"`
	Mii        []MiiInfo `json:"mii,omitempty"`
}

type GroupInfo struct {
	GroupName   string    `json:"id"`
	GameName    string    `json:"game"`
	CreateTime  time.Time `json:"created"`
	MatchType   string    `json:"type"`
	Suspend     bool      `json:"suspend"`
	ServerIndex string    `json:"host,omitempty"`
	MKWRegion   string    `json:"rk,omitempty"`

	Players  map[string]PlayerInfo `json:"players"`
	RaceInfo *RaceInfo             `json:"race,omitempty"`

	PlayersRaw      map[string]map[string]string `json:"-"`
	SortedJoinIndex []string                     `json:"-"`
}

type RaceInfo struct {
	RaceNumber    int `json:"num"`
	CourseID      int `json:"course"`
	EngineClassID int `json:"cc"`
}
