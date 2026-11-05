package wfcapi

import "openpak/wfc-common/common"

type Login struct {
	ProfileID                                     uint32
	UserID                                        uint64
	Generation                                    string
	Revision                                      int64
	GameCode, InGameName                          string
	ConsoleFriendCode                             uint64
	FriendKeyGame, GPPublicIP                     string
	NeedsExploit, DeviceAuthenticated, Restricted bool
	// Unlinked marks the session of a Wii on the DNS-exploit path that is not linked to an
	// OpenPak account: it logged in only so QR2 can deliver the payload, and its exlogin will be
	// refused. QR2 gives it the exploit and nothing else. False for every other login (and for
	// gpcm builds that predate link-first).
	Unlinked bool `json:",omitempty"`
}
type Session struct {
	Login                Login
	SessionKey           int32
	GameName, UniqueNick string
}
type QR2Request struct {
	Login                                  *Login
	ProfileID                              uint32
	Generation                             string
	Address, DestAddress                   uint64
	DestProfileID                          uint32
	Status, Key, Value, SenderIP, IP1, IP2 string
	Packet                                 []byte
	SearchID                               uint64
	JoinType, Result                       byte
	MatchVersion                           int
	Reservation                            common.MatchCommandDataReservation
	ResvOK                                 common.MatchCommandDataResvOK
	Games, Groups                          []string
	Sorted                                 bool
}
type QR2Response struct {
	Logins   []Login
	Servers  []map[string]string
	Groups   []GroupInfo
	SearchID uint64
	Allowed  string
	Accepted bool
}
type GPCMRequest struct {
	ProfileID        uint32
	UserID           uint64 `json:",omitempty"` // kick-user: every session of this console
	Generation       string
	SessionKey       int32
	GameName, Reason string
	Custom           string
}
type GPCMResponse struct {
	Sessions   []Session
	Kicked     int `json:",omitempty"` // kick-user: sessions closed
	UniqueNick string
	Valid      bool
}
