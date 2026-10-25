package gpcmclient

import (
	"context"
	"openpak/wfc-common/gperror"
	"openpak/wfc-common/wfcapi"
)

type GPError = gperror.GPError
type WWFCErrorMessage = gperror.WWFCErrorMessage

var ErrParse = gperror.ErrParse
var ErrSearch = gperror.ErrSearch
var ErrDatabase = gperror.ErrDatabase
var ErrNotLoggedIn = gperror.ErrNotLoggedIn
var WWFCMsgKickedCustom = gperror.WWFCMsgKickedCustom
var WWFCMsgProfileRestrictedCustom = gperror.WWFCMsgProfileRestrictedCustom

func VerifyPlayerSearch(pid uint32, key int32, game string) (string, bool) {
	var out wfcapi.GPCMResponse
	err := wfcapi.Call(context.Background(), "WFC_GPCM_URL", "/internal/v1/gpcm/verify", wfcapi.GPCMRequest{ProfileID: pid, SessionKey: key, GameName: game}, &out)
	return out.UniqueNick, err == nil && out.Valid
}
func KickPlayerCustomMessage(pid uint32, reason string, message WWFCErrorMessage) error {
	custom := "kick"
	if message.ErrorCode == WWFCMsgProfileRestrictedCustom.ErrorCode {
		custom = "restricted"
	}
	return wfcapi.Call(context.Background(), "WFC_GPCM_URL", "/internal/v1/gpcm/kick", wfcapi.GPCMRequest{ProfileID: pid, Reason: reason, Custom: custom}, nil)
}
func Sessions(ctx context.Context) ([]wfcapi.Session, error) {
	var out wfcapi.GPCMResponse
	e := wfcapi.Call(ctx, "WFC_GPCM_URL", "/internal/v1/gpcm/sessions", wfcapi.GPCMRequest{}, &out)
	return out.Sessions, e
}
func Callback(ctx context.Context, pid uint32, generation, reason string) error {
	return wfcapi.Call(ctx, "WFC_GPCM_URL", "/internal/v1/gpcm/callback", wfcapi.GPCMRequest{ProfileID: pid, Generation: generation, Reason: reason}, nil)
}
