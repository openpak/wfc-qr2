package wfcapi

import "openpak/wfc-common/common"

type Login struct {
	ProfileID                                     uint32
	Generation                                    string
	Revision                                      int64
	GameCode, InGameName                          string
	ConsoleFriendCode                             uint64
	FriendKeyGame, GPPublicIP                     string
	NeedsExploit, DeviceAuthenticated, Restricted bool
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
	Generation       string
	SessionKey       int32
	GameName, Reason string
	Custom           string
}
type GPCMResponse struct {
	Sessions   []Session
	UniqueNick string
	Valid      bool
}
