package main

import (
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"log"
	"net"
	"path/filepath"
	"strings"
	"time"
)

var eventTypeIdMap = map[string]int{
	"no_auth":                   1,
	"password_auth":             2,
	"public_key_auth":           3,
	"keyboard_interactive_auth": 4,
	"connection":                5,
	"connection_close":          6,
	"tcpip_forward":             7,
	"cancel_tcpip_forward":      8,
	"no_more_sessions":          9,
	"host_keys_prove":           10,
	"session":                   11,
	"session_close":             12,
	"session_input":             13,
	"direct_tcpip":              14,
	"direct_tcpip_close":        15,
	"direct_tcpip_input":        16,
	"pty":                       17,
	"shell":                     18,
	"exec":                      19,
	"subsystem":                 20,
	"x11":                       21,
	"env":                       22,
	"window_change":             23,
	"debug_global_request":      24,
	"debug_channel":             25,
	"debug_channel_request":     26,
}

type logEntry interface {
	fmt.Stringer
	eventType() string
}

type addressLog struct {
	Host string `json:"host" bson:"host"`
	Port int    `json:"port" bson:"port"`
}

func (entry addressLog) String() string {
	return net.JoinHostPort(entry.Host, fmt.Sprint(entry.Port))
}

func getAddressLog(host string, port int, cfg *config) interface{} {
	entry := addressLog{
		Host: host,
		Port: port,
	}
	if cfg.Logging.SplitHostPort {
		return entry
	}
	return entry.String()
}

type authAccepted bool

func (accepted authAccepted) String() string {
	if accepted {
		return "accepted"
	}
	return "rejected"
}

type authLog struct {
	User     string       `json:"user" bson:"user"`
	Accepted authAccepted `json:"accepted" bson:"accepted"`
}

type noAuthLog struct {
	authLog
}

func (entry noAuthLog) String() string {
	return fmt.Sprintf("authentication for user %q without credentials %v", entry.User, entry.Accepted)
}
func (entry noAuthLog) eventType() string {
	return "no_auth"
}

type passwordAuthLog struct {
	authLog
	Password string `json:"password" bson:"password"`
}

func (entry passwordAuthLog) String() string {
	return fmt.Sprintf("authentication for user %q with password %q %v", entry.User, entry.Password, entry.Accepted)
}
func (entry passwordAuthLog) eventType() string {
	return "password_auth"
}

type publicKeyAuthLog struct {
	authLog
	PublicKeyFingerprint string `json:"public_key" bson:"public_key"`
}

func (entry publicKeyAuthLog) String() string {
	return fmt.Sprintf("authentication for user %q with public key %q %v", entry.User, entry.PublicKeyFingerprint, entry.Accepted)
}
func (entry publicKeyAuthLog) eventType() string {
	return "public_key_auth"
}

type keyboardInteractiveAuthLog struct {
	authLog
	Answers []string `json:"answers" bson:"answers"`
}

func (entry keyboardInteractiveAuthLog) String() string {
	return fmt.Sprintf("authentication for user %q with keyboard interactive answers %q %v", entry.User, entry.Answers, entry.Accepted)
}
func (entry keyboardInteractiveAuthLog) eventType() string {
	return "keyboard_interactive_auth"
}

type connectionLog struct {
	ClientVersion string `json:"client_version" bson:"client_version"`
}

func (entry connectionLog) String() string {
	return fmt.Sprintf("connection with client version %q established", entry.ClientVersion)
}
func (entry connectionLog) eventType() string {
	return "connection"
}

type connectionCloseLog struct {
}

func (entry connectionCloseLog) String() string {
	return "connection closed"
}
func (entry connectionCloseLog) eventType() string {
	return "connection_close"
}

type tcpipForwardLog struct {
	Address interface{} `json:"address" bson:"address"`
}

func (entry tcpipForwardLog) String() string {
	return fmt.Sprintf("TCP/IP forwarding on %v requested", entry.Address)
}
func (entry tcpipForwardLog) eventType() string {
	return "tcpip_forward"
}

type cancelTCPIPForwardLog struct {
	Address interface{} `json:"address" bson:"address"`
}

func (entry cancelTCPIPForwardLog) String() string {
	return fmt.Sprintf("TCP/IP forwarding on %v canceled", entry.Address)
}
func (entry cancelTCPIPForwardLog) eventType() string {
	return "cancel_tcpip_forward"
}

type noMoreSessionsLog struct {
}

func (entry noMoreSessionsLog) String() string {
	return "rejection of further session channels requested"
}
func (entry noMoreSessionsLog) eventType() string {
	return "no_more_sessions"
}

type hostKeysProveLog struct {
	HostKeyFiles []string `json:"host_key_files" bson:"host_key_files"`
}

func (entry hostKeysProveLog) String() string {
	baseNames := make([]string, len(entry.HostKeyFiles))
	for i, hostKeyFile := range entry.HostKeyFiles {
		baseNames[i] = fmt.Sprintf("%q", filepath.Base(hostKeyFile))
	}
	return fmt.Sprintf("proof of ownership of host keys %v requested", strings.Join(baseNames, ", "))
}
func (entry hostKeysProveLog) eventType() string {
	return "host_keys_prove"
}

type channelLog struct {
	ChannelID int `json:"channel_id" bson:"channel_id"`
}

type sessionLog struct {
	channelLog
}

func (entry sessionLog) String() string {
	return fmt.Sprintf("[channel %v] session requested", entry.ChannelID)
}
func (entry sessionLog) eventType() string {
	return "session"
}

type sessionCloseLog struct {
	channelLog
}

func (entry sessionCloseLog) String() string {
	return fmt.Sprintf("[channel %v] closed", entry.ChannelID)
}
func (entry sessionCloseLog) eventType() string {
	return "session_close"
}

type sessionInputLog struct {
	channelLog
	Input string `json:"input" bson:"input"`
}

func (entry sessionInputLog) String() string {
	return fmt.Sprintf("[channel %v] input: %q", entry.ChannelID, entry.Input)
}
func (entry sessionInputLog) eventType() string {
	return "session_input"
}

type directTCPIPLog struct {
	channelLog
	From interface{} `json:"from" bson:"from"`
	To   interface{} `json:"to" bson:"to"`
}

func (entry directTCPIPLog) String() string {
	return fmt.Sprintf("[channel %v] direct TCP/IP forwarding from %v to %v requested", entry.ChannelID, entry.From, entry.To)
}
func (entry directTCPIPLog) eventType() string {
	return "direct_tcpip"
}

type directTCPIPCloseLog struct {
	channelLog
}

func (entry directTCPIPCloseLog) String() string {
	return fmt.Sprintf("[channel %v] closed", entry.ChannelID)
}
func (entry directTCPIPCloseLog) eventType() string {
	return "direct_tcpip_close"
}

type directTCPIPInputLog struct {
	channelLog
	Input string `json:"input" bson:"input"`
}

func (entry directTCPIPInputLog) String() string {
	return fmt.Sprintf("[channel %v] input: %q", entry.ChannelID, entry.Input)
}
func (entry directTCPIPInputLog) eventType() string {
	return "direct_tcpip_input"
}

type ptyLog struct {
	channelLog
	Terminal string `json:"terminal" bson:"terminal"`
	Width    uint32 `json:"width" bson:"width"`
	Height   uint32 `json:"height" bson:"height"`
}

func (entry ptyLog) String() string {
	return fmt.Sprintf("[channel %v] PTY using terminal %q (size %vx%v) requested", entry.ChannelID, entry.Terminal, entry.Width, entry.Height)
}
func (entry ptyLog) eventType() string {
	return "pty"
}

type shellLog struct {
	channelLog
}

func (entry shellLog) String() string {
	return fmt.Sprintf("[channel %v] shell requested", entry.ChannelID)
}
func (entry shellLog) eventType() string {
	return "shell"
}

type execLog struct {
	channelLog
	Command string `json:"command" bson:"command"`
}

func (entry execLog) String() string {
	return fmt.Sprintf("[channel %v] command %q requested", entry.ChannelID, entry.Command)
}
func (entry execLog) eventType() string {
	return "exec"
}

type subsystemLog struct {
	channelLog
	Subsystem string `json:"subsystem" bson:"subsystem"`
}

func (entry subsystemLog) String() string {
	return fmt.Sprintf("[channel %v] subsystem %q requested", entry.ChannelID, entry.Subsystem)
}
func (entry subsystemLog) eventType() string {
	return "subsystem"
}

type x11Log struct {
	channelLog
	Screen uint32 `json:"screen" bson:"screen"`
}

func (entry x11Log) String() string {
	return fmt.Sprintf("[channel %v] X11 forwarding on screen %v requested", entry.ChannelID, entry.Screen)
}
func (entry x11Log) eventType() string {
	return "x11"
}

type envLog struct {
	channelLog
	Name  string `json:"name" bson:"name"`
	Value string `json:"value" bson:"value"`
}

func (entry envLog) String() string {
	return fmt.Sprintf("[channel %v] environment variable %q with value %q requested", entry.ChannelID, entry.Name, entry.Value)
}
func (entry envLog) eventType() string {
	return "env"
}

type windowChangeLog struct {
	channelLog
	Width  uint32 `json:"width" bson:"width"`
	Height uint32 `json:"height" bson:"height"`
}

func (entry windowChangeLog) String() string {
	return fmt.Sprintf("[channel %v] window size change to %vx%v requested", entry.ChannelID, entry.Width, entry.Height)
}
func (entry windowChangeLog) eventType() string {
	return "window_change"
}

type debugGlobalRequestLog struct {
	RequestType string `json:"request_type" bson:"request_type"`
	WantReply   bool   `json:"want_reply" bson:"want_reply"`
	Payload     string `json:"payload" bson:"payload"`
}

func (entry debugGlobalRequestLog) String() string {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		warningLogger.Printf("Failed to log event: %v", err)
		return ""
	}
	return fmt.Sprintf("DEBUG global request received: %v\n", string(jsonBytes))
}
func (entry debugGlobalRequestLog) eventType() string {
	return "debug_global_request"
}

type debugChannelLog struct {
	channelLog
	ChannelType string `json:"channel_type" bson:"channel_type"`
	ExtraData   string `json:"extra_data" bson:"extra_data"`
}

func (entry debugChannelLog) String() string {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		warningLogger.Printf("Failed to log event: %v", err)
		return ""
	}
	return fmt.Sprintf("DEBUG new channel requested: %v\n", string(jsonBytes))
}
func (entry debugChannelLog) eventType() string {
	return "debug_channel"
}

type debugChannelRequestLog struct {
	channelLog
	RequestType string `json:"request_type" bson:"request_type"`
	WantReply   bool   `json:"want_reply" bson:"want_reply"`
	Payload     string `json:"payload" bson:"payload"`
}

func (entry debugChannelRequestLog) String() string {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		warningLogger.Printf("Failed to log event: %v", err)
		return ""
	}
	return fmt.Sprintf("DEBUG channel request received: %v\n", string(jsonBytes))
}
func (entry debugChannelRequestLog) eventType() string {
	return "debug_channel_request"
}

func (context connContext) logEvent(entry logEntry) {
	if strings.HasPrefix(entry.eventType(), "debug_") && !context.cfg.Logging.Debug {
		return
	}
	if context.cfg.MongoDBConfig.Enable && context.cfg.mongoRecorder.isConnected {
		context.logEventToMongo(entry)
	}
	if context.cfg.Logging.JSON {
		var jsonEntry interface{}
		tcpSource := context.RemoteAddr().(*net.TCPAddr)
		source := getAddressLog(tcpSource.IP.String(), tcpSource.Port, context.cfg)
		if context.cfg.Logging.Timestamps {
			jsonEntry = struct {
				SessionId int64       `json:"session_id"`
				Time      int64       `json:"time"`
				Source    interface{} `json:"source"`
				EventType string      `json:"event_type"`
				Event     logEntry    `json:"event"`
			}{
				context.sessionId,
				time.Now().Unix(),
				source,
				entry.eventType(),
				entry,
			}
		} else {
			jsonEntry = struct {
				SessionId int64       `json:"session_id"`
				Source    interface{} `json:"source"`
				EventType string      `json:"event_type"`
				Event     logEntry    `json:"event"`
			}{
				context.sessionId,
				source,
				entry.eventType(),
				entry,
			}
		}
		logBytes, err := json.Marshal(jsonEntry)
		if err != nil {
			warningLogger.Printf("Failed to log event: %v", err)
			return
		}
		log.Print(string(logBytes))
	} else {
		log.Printf("[%v] %v", context.RemoteAddr().String(), entry)
	}
}

func mergeBSONM(bson1, bson2 bson.M) *bson.M {
	for k, v := range bson2 {
		bson1[k] = v
	}
	return &bson1
}

func (context connContext) logEventToMongo(entry logEntry) {
	eventType := entry.eventType()
	eventTypeId, ok := eventTypeIdMap[eventType]
	tcpSource := context.RemoteAddr().(*net.TCPAddr)
	if !ok {
		eventTypeId = 0
	}
	logRecord := &bson.M{
		"time":        time.Now(),
		"session_id":  context.sessionId,
		"event_type":  eventTypeId,
		"source_ip":   tcpSource.IP.String(),
		"source_port": tcpSource.Port,
	}
	LogEventToMongo(context.cfg.mongoRecorder, eventType, logRecord, entry)
}
