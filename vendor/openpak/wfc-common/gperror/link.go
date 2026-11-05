package gperror

import "strconv"

// Link-first refusals (prds repos/wfc-link-first-design.md). Codes 22100-22102 are OpenPak's,
// outside WiiLink's 22000-22009. Mario Kart Wii shows the text (the payload's message 6602);
// every other Wii title shows only the number in wl:err, so for them the number IS the console's
// link number; the DS shows its stock DWC error number.

var (
	// WWFCMsgConsoleNotLinked: %[3]s is the pairing code and link number.
	WWFCMsgConsoleNotLinked = WWFCErrorMessage{
		ErrorCode: 22100,
		MessageRMC: map[byte]string{
			LangEnglish: "" +
				"This Wii is not linked to an\n" +
				"OpenPak account yet.\n" +
				"Go to openpak.org/link\n" +
				"and enter this code:\n" +
				"%[3]s\n" +
				"\n" +
				"Error Code: %[1]d",
			LangJapanese: "" +
				"このWiiはまだOpenPakアカウントに\n" +
				"連携されていません。\n" +
				"openpak.org/link で\n" +
				"このコードを入力してください：\n" +
				"%[3]s\n" +
				"\n" +
				"エラーコード: %[1]d",
			LangFrench: "" +
				"Cette Wii n'est pas encore liée\n" +
				"à un compte OpenPak.\n" +
				"Allez sur openpak.org/link\n" +
				"et saisissez ce code :\n" +
				"%[3]s\n" +
				"\n" +
				"Code d'erreur : %[1]d",
			LangGerman: "" +
				"Diese Wii ist noch nicht mit\n" +
				"einem OpenPak-Konto verknüpft.\n" +
				"Gehe auf openpak.org/link\n" +
				"und gib diesen Code ein:\n" +
				"%[3]s\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"Esta Wii aún no está vinculada\n" +
				"a una cuenta de OpenPak.\n" +
				"Ve a openpak.org/link\n" +
				"e introduce este código:\n" +
				"%[3]s\n" +
				"\n" +
				"Código de error: %[1]d",
			LangItalian: "" +
				"Questa Wii non è ancora collegata\n" +
				"a un account OpenPak.\n" +
				"Vai su openpak.org/link\n" +
				"e inserisci questo codice:\n" +
				"%[3]s\n" +
				"\n" +
				"Codice di errore: %[1]d",
			LangDutch: "" +
				"Deze Wii is nog niet gekoppeld\n" +
				"aan een OpenPak-account.\n" +
				"Ga naar openpak.org/link\n" +
				"en voer deze code in:\n" +
				"%[3]s\n" +
				"\n" +
				"Foutcode: %[1]d",
		},
	}

	WWFCMsgConsoleUnlinked = WWFCErrorMessage{
		ErrorCode: 22101,
		MessageRMC: map[byte]string{
			LangEnglish: "" +
				"This Wii was unlinked from\n" +
				"its OpenPak account.\n" +
				"Connect again to get a new\n" +
				"code to link it.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangJapanese: "" +
				"このWiiとOpenPakアカウントの\n" +
				"連携が解除されました。\n" +
				"もう一度接続すると新しい\n" +
				"コードが表示されます。\n" +
				"\n" +
				"エラーコード: %[1]d",
			LangFrench: "" +
				"Cette Wii a été déliée de\n" +
				"son compte OpenPak.\n" +
				"Reconnectez-vous pour obtenir\n" +
				"un nouveau code.\n" +
				"\n" +
				"Code d'erreur : %[1]d",
			LangGerman: "" +
				"Diese Wii wurde von ihrem\n" +
				"OpenPak-Konto getrennt.\n" +
				"Verbinde dich erneut, um einen\n" +
				"neuen Code zu erhalten.\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"Esta Wii se ha desvinculado\n" +
				"de su cuenta de OpenPak.\n" +
				"Conéctate de nuevo para\n" +
				"obtener un código nuevo.\n" +
				"\n" +
				"Código de error: %[1]d",
			LangItalian: "" +
				"Questa Wii è stata scollegata\n" +
				"dal suo account OpenPak.\n" +
				"Connettiti di nuovo per\n" +
				"ricevere un nuovo codice.\n" +
				"\n" +
				"Codice di errore: %[1]d",
			LangDutch: "" +
				"Deze Wii is ontkoppeld van\n" +
				"zijn OpenPak-account.\n" +
				"Maak opnieuw verbinding voor\n" +
				"een nieuwe code.\n" +
				"\n" +
				"Foutcode: %[1]d",
		},
	}

	WWFCMsgLinkUnavailable = WWFCErrorMessage{
		ErrorCode: 22102,
		MessageRMC: map[byte]string{
			LangEnglish: "" +
				"OpenPak cannot check this\n" +
				"Wii's account link right now.\n" +
				"Please try again later.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangJapanese: "" +
				"現在、このWiiのOpenPak連携を\n" +
				"確認できません。\n" +
				"しばらくしてからお試しください。\n" +
				"\n" +
				"エラーコード: %[1]d",
			LangFrench: "" +
				"OpenPak ne peut pas vérifier\n" +
				"le lien de cette Wii.\n" +
				"Réessayez plus tard.\n" +
				"\n" +
				"Code d'erreur : %[1]d",
			LangGerman: "" +
				"OpenPak kann die Verknüpfung\n" +
				"dieser Wii gerade nicht prüfen.\n" +
				"Bitte versuche es später.\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"OpenPak no puede comprobar\n" +
				"el vínculo de esta Wii ahora.\n" +
				"Inténtalo más tarde.\n" +
				"\n" +
				"Código de error: %[1]d",
			LangItalian: "" +
				"OpenPak non riesce a verificare\n" +
				"il collegamento di questa Wii.\n" +
				"Riprova più tardi.\n" +
				"\n" +
				"Codice di errore: %[1]d",
			LangDutch: "" +
				"OpenPak kan de koppeling van\n" +
				"deze Wii nu niet controleren.\n" +
				"Probeer het later opnieuw.\n" +
				"\n" +
				"Foutcode: %[1]d",
		},
	}
)

// NotLinkedError is the login refusal for an unlinked console. Mario Kart Wii gets the text
// with the code and number; any other title gets the link number as its error code (wl:err),
// which is all the payload can show there. dsErr is the GP err value a DS is sent (see the
// design, section 5.3); 0 keeps the generic login error.
func NotLinkedError(gameName, formattedCode string, number int, dsErr int) GPError {
	e := GPError{ErrorCode: ErrLogin.ErrorCode, ErrorString: "This console is not linked to an OpenPak account.", Fatal: true}
	if gameName == "mariokartwii" {
		e.WWFCMessage = WWFCMsgConsoleNotLinked
		e.Reason = formattedCode + "  or  " + strconv.Itoa(number)
		return e
	}
	e.WWFCMessage = WWFCErrorMessage{ErrorCode: number}
	if dsErr != 0 {
		e.ErrorCode = dsErr
	}
	return e
}

// UnlinkedKickError ends the session of a console unlinked while online.
func UnlinkedKickError() GPError {
	return GPError{ErrorCode: ErrForcedDisconnect.ErrorCode, ErrorString: "This console was unlinked from its OpenPak account.",
		Fatal: true, WWFCMessage: WWFCMsgConsoleUnlinked, Reason: "unlinked"}
}

// LinkUnavailableError refuses a login whose link state cannot be read right now.
func LinkUnavailableError() GPError {
	return GPError{ErrorCode: ErrLogin.ErrorCode, ErrorString: "The account link cannot be checked right now.",
		Fatal: true, WWFCMessage: WWFCMsgLinkUnavailable, Reason: "unavailable"}
}
