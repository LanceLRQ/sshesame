package main

import (
	"bufio"
	"encoding/json"
	"go.mongodb.org/mongo-driver/v2/bson"
	"os"
	"strconv"
	"strings"
	"time"
)

var eventTypeCounter = map[string]int{
	"no_auth":                   0,
	"password_auth":             0,
	"public_key_auth":           0,
	"keyboard_interactive_auth": 0,
	"connection":                0,
	"connection_close":          0,
	"tcpip_forward":             0,
	"cancel_tcpip_forward":      0,
	"no_more_sessions":          0,
	"host_keys_prove":           0,
	"session":                   0,
	"session_close":             0,
	"session_input":             0,
	"direct_tcpip":              0,
	"direct_tcpip_close":        0,
	"direct_tcpip_input":        0,
	"pty":                       0,
	"shell":                     0,
	"exec":                      0,
	"subsystem":                 0,
	"x11":                       0,
	"env":                       0,
	"window_change":             0,
	"debug_global_request":      0,
	"debug_channel":             0,
	"debug_channel_request":     0,
}

func parseOldLogToMongo(cfg *config, filePath string, isJSON bool, dryRun bool) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		errorLogger.Fatalf("log file %v not exists", err)
	} else if err != nil {
		errorLogger.Fatalf("Failed to read log file: %v", err)
	}

	fp, err := os.Open(filePath)
	if err != nil {
		errorLogger.Fatalf("Failed to read log file: %v", err)
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		line := scanner.Text()
		if isJSON {
			var logObj struct {
				Time      time.Time       `json:"time"`
				Source    string          `json:"source"`
				EventType string          `json:"event_type"`
				Event     json.RawMessage `json:"event"`
			}
			err = json.Unmarshal([]byte(line), &logObj)
			if err == nil {
				eventTypeCounter[logObj.EventType]++
				var entry logEntry
				if logObj.EventType == "no_auth" {
					entry = noAuthLog{}
					err = json.Unmarshal(logObj.Event, &entry)
				} else if logObj.EventType == "password_auth" {
					entry = passwordAuthLog{}
					err = json.Unmarshal(logObj.Event, &entry)
				} else if logObj.EventType == "public_key_auth" {
					entry = publicKeyAuthLog{}
					err = json.Unmarshal(logObj.Event, &entry)
				} else if logObj.EventType == "keyboard_interactive_auth" {
					entry = keyboardInteractiveAuthLog{}
					err = json.Unmarshal(logObj.Event, &entry)
				} else if logObj.EventType == "session_input" {
					entry = sessionInputLog{}
					err = json.Unmarshal(logObj.Event, &entry)
				}
				if err == nil && entry != nil {
					eventTypeId := eventTypeIdMap[logObj.EventType]
					ipAddrSp := strings.Split(logObj.Source, ":")
					ipAddr := logObj.Source
					ipPort := int64(0)
					if len(ipAddrSp) == 2 {
						ipAddr = ipAddrSp[0]
						ipPort, _ = strconv.ParseInt(ipAddrSp[1], 10, 32)
					}
					logRecord := &bson.M{
						"time":        logObj.Time,
						"session_id":  0,
						"event_type":  eventTypeId,
						"source_ip":   ipAddr,
						"source_port": ipPort,
					}
					if !dryRun {
						LogEventToMongo(cfg.mongoRecorder, logObj.EventType, logRecord, entry)
					}
				}
			}
		}
	}
	if dryRun {
		infoLogger.Printf("%s", ObjectToJSONString(eventTypeCounter))
	}
	if err := scanner.Err(); err != nil {
		errorLogger.Fatalf("Error reading file:", err)
	}
	infoLogger.Println("Done")
}
