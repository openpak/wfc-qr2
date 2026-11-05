package qr2

import (
	"encoding/binary"
	"net"
	"openpak/wfc-common/common"
	"openpak/wfc-common/database"
	"openpak/wfc-common/httplog"
	"openpak/wfc-common/logging"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/logrusorgru/aurora/v3"
)

const (
	QueryRequest            = 0x00
	ChallengeRequest        = 0x01
	EchoRequest             = 0x02
	HeartbeatRequest        = 0x03
	AddErrorRequest         = 0x04
	EchoResponseRequest     = 0x05
	ClientMessageRequest    = 0x06
	ClientMessageAckRequest = 0x07
	KeepAliveRequest        = 0x08
	AvailableRequest        = 0x09
	ClientRegisteredReply   = 0x0A

	// Added by WiiLink
	ClientExploitReply = 0x10
)

var (
	masterConn net.PacketConn
	inShutdown atomic.Bool
	waitGroup  = sync.WaitGroup{}
)

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()

	address := *config.GameSpyAddress + ":27900"
	// WFC_PROXY_LISTEN: also take datagrams relayed by wfc-front, framed with the player's real
	// address (PROXY v2, prds#149); replies to those players go back through the relay.
	conn, err := common.ListenGameUDP(address, os.Getenv("WFC_PROXY_LISTEN"))
	if err != nil {
		panic(err)
	}

	// Connect to database for event logging
	if config.EventReporting.LogToDatabase {
		db := database.Start(config)
		db.RegisterEvents(config, []string{
			"group_created",
			"group_deleted",
			"group_joined",
			"group_left",
			"group_host_changed",
			"natneg_succeeded",
			"natneg_failed",
		})
	}

	masterConn = conn
	inShutdown.Store(false)

	if reload {
		err := loadSessions()
		if err != nil {
			panic(err)
		}

		logging.Notice("QR2", "Loaded", aurora.Cyan(len(sessions)), "sessions")

		err = loadLogins()
		if err != nil {
			panic(err)
		}

		logging.Notice("QR2", "Loaded", aurora.Cyan(len(logins)), "logins")

		err = loadGroups()
		if err != nil {
			panic(err)
		}

		logging.Notice("QR2", "Loaded", aurora.Cyan(len(groups)), "groups")
	}

	startRewards()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		// Close the listener when the application closes.
		defer func() {
			_ = conn.Close()
		}()
		logging.Notice("QR2", "Listening on", aurora.BrightCyan(address))

		for {
			if inShutdown.Load() {
				return
			}

			buf := make([]byte, 1024)
			n, addr, err := conn.ReadFrom(buf)
			if err != nil || n == 0 {
				continue
			}

			waitGroup.Add(1)

			go handleConnection(conn, *addr.(*net.UDPAddr), buf)
		}
	}()
}

func Shutdown() {
	inShutdown.Store(true)
	common.ShouldNotError(masterConn.Close())
	waitGroup.Wait()
	stopRewards()

	mutex.Lock()
	defer mutex.Unlock()

	err := saveSessions()
	if err != nil {
		logging.Error("QR2", "Failed to save sessions:", err)
	}

	logging.Notice("QR2", "Saved", aurora.Cyan(len(sessions)), "sessions")

	err = saveLogins()
	if err != nil {
		logging.Error("QR2", "Failed to save logins:", err)
	}

	logging.Notice("QR2", "Saved", aurora.Cyan(len(logins)), "logins")

	err = saveGroups()
	if err != nil {
		logging.Error("QR2", "Failed to save groups:", err)
	}

	logging.Notice("QR2", "Saved", aurora.Cyan(len(groups)), "groups")
}

func handleConnection(conn net.PacketConn, addr net.UDPAddr, buffer []byte) {
	defer waitGroup.Done()

	packetType := buffer[0]
	moduleName := "QR2:" + addr.String()

	start, result := time.Now(), "ok"
	defer func() { httplog.Command("qr2", packetName(packetType), result, start) }()

	var session *Session
	if packetType != HeartbeatRequest && packetType != AvailableRequest {
		mutex.Lock()

		var ok bool
		session, ok = sessions[makeLookupAddr(addr.String())]
		if !ok {
			mutex.Unlock()
			result = "no_session"
			logging.Error(moduleName, "Cannot find session for this IP address")
			return
		}

		session.SessionID = binary.BigEndian.Uint32(buffer[1:5])

		mutex.Unlock()
	}

	switch packetType {
	case QueryRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("QUERY"))

	case ChallengeRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("CHALLENGE"))

		mutex.Lock()
		if session.Challenge != "" {
			// TODO: Verify the challenge
			session.Authenticated = true
			mutex.Unlock()

			_, _ = conn.WriteTo(createResponseHeader(ClientRegisteredReply, session.SessionID), &addr)
		} else {
			mutex.Unlock()
		}

	case EchoRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("ECHO"))

	case HeartbeatRequest:
		// logging.Info(moduleName, "Command:", aurora.Yellow("HEARTBEAT"))
		heartbeat(moduleName, conn, addr, buffer)

	case AddErrorRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("ADDERROR"))

	case EchoResponseRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("ECHO_RESPONSE"))

	case ClientMessageRequest:
		logging.Info(moduleName, "Command:", aurora.Yellow("CLIENT_MESSAGE"))
		return

	case ClientMessageAckRequest:
		// logging.Info(moduleName, "Command:", aurora.Yellow("CLIENT_MESSAGE_ACK"))

		// In case ClientExploitReply is lost, this can be checked as well
		// This would be sent either after the payload is downloaded, or the client is already patched
		session.ExploitReceived = true
		if login := session.login; login != nil {
			login.NeedsExploit = false
		}

		session.messageAckWaker.Assert()
		return

	case KeepAliveRequest:
		// logging.Info(moduleName, "Command:", aurora.Yellow("KEEPALIVE"))
		_, _ = conn.WriteTo(createResponseHeader(KeepAliveRequest, 0), &addr)

		session.LastKeepAlive = time.Now().UTC().Unix()
		return

	case AvailableRequest:
		logging.Info("QR2", "Command:", aurora.Yellow("AVAILABLE"))
		_, _ = conn.WriteTo(createResponseHeader(AvailableRequest, 0), &addr)
		return

	case ClientRegisteredReply:
		logging.Info(moduleName, "Command:", aurora.Yellow("CLIENT_REGISTERED"))

	case ClientExploitReply:
		logging.Info(moduleName, "Command:", aurora.Yellow("CLIENT_EXPLOIT_ACK"))

		session.ExploitReceived = true
		if login := session.login; login != nil {
			login.NeedsExploit = false
		}

	default:
		result = "unknown"
		logging.Error(moduleName, "Unknown command:", aurora.Yellow(buffer[0]))
		return
	}
}

func createResponseHeader(command byte, sessionId uint32) []byte {
	return binary.BigEndian.AppendUint32([]byte{0xfe, 0xfd, command}, sessionId)
}

// packetName is the log name of a QR2 packet type; unrecognised types all log as "unknown".
func packetName(t byte) string {
	switch t {
	case QueryRequest:
		return "query"
	case ChallengeRequest:
		return "challenge"
	case EchoRequest:
		return "echo"
	case HeartbeatRequest:
		return "heartbeat"
	case AddErrorRequest:
		return "add_error"
	case EchoResponseRequest:
		return "echo_response"
	case ClientMessageRequest:
		return "client_message"
	case ClientMessageAckRequest:
		return "client_message_ack"
	case KeepAliveRequest:
		return "keepalive"
	case AvailableRequest:
		return "available"
	case ClientRegisteredReply:
		return "client_registered"
	case ClientExploitReply:
		return "client_exploit_ack"
	}
	return "unknown"
}
