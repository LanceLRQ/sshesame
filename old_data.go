package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"os"
	"regexp"
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

func TrimAndRemoveQuote(str string) string {
	str = strings.Trim(str, " ")
	str = strings.Trim(str, "\"")
	return str
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

	counter := 0
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		counter++
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
				flag := false
				var entry logEntry
				if logObj.EventType == "no_auth" {
					a1 := noAuthLog{}
					err = json.Unmarshal(logObj.Event, &a1)
					entry = a1
					flag = true
				} else if logObj.EventType == "password_auth" {
					p1 := passwordAuthLog{}
					err = json.Unmarshal(logObj.Event, &p1)
					entry = p1
					flag = true
				} else if logObj.EventType == "public_key_auth" {
					p2 := publicKeyAuthLog{}
					err = json.Unmarshal(logObj.Event, &p2)
					entry = p2
					flag = true
				} else if logObj.EventType == "keyboard_interactive_auth" {
					k1 := keyboardInteractiveAuthLog{}
					err = json.Unmarshal(logObj.Event, &k1)
					entry = k1
					flag = true
				} else if logObj.EventType == "session_input" {
					s1 := sessionInputLog{}
					err = json.Unmarshal(logObj.Event, &s1)
					entry = s1
					flag = true
				}
				infoLogger.Println(err)
				if err == nil && flag {
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
		} else {
			lineSplited := strings.Split(line, " ")
			if len(lineSplited) >= 4 {
				timeStr := fmt.Sprintf("%s %s", lineSplited[0], lineSplited[1])
				ipStr := strings.TrimRight(strings.TrimLeft(lineSplited[2], "["), "]")
				ipAddrSp := strings.Split(ipStr, ":")
				ipAddr := ipStr
				ipPort := int64(0)
				if len(ipAddrSp) == 2 {
					ipAddr = ipAddrSp[0]
					ipPort, _ = strconv.ParseInt(ipAddrSp[1], 10, 32)
				}
				contentStr := strings.Join(lineSplited[3:], " ")
				eventType := ""
				var entry logEntry

				reInput, _ := regexp.Compile("\\[channel (\\d+)] input:")
				reAuth, _ := regexp.Compile("authentication for user")
				if reInput.Match([]byte(contentStr)) {
					eventType = "session_input"
					lIndex := strings.Index(contentStr, "input:") + 6
					var s1 sessionInputLog
					if cid, err := strconv.ParseInt(reInput.FindStringSubmatch(contentStr)[1], 10, 32); err == nil {
						s1.ChannelID = int(cid)
					}
					s1.Input = TrimAndRemoveQuote(contentStr[lIndex:])
					entry = s1
				} else if reAuth.Match([]byte(contentStr)) {
					lIndex := strings.Index(contentStr, "authentication for user") + 25
					l2Index := strings.Index(contentStr, "without credentials")
					if l2Index > -1 {
						eventType = "no_auth"
						var a1 noAuthLog
						a1.User = TrimAndRemoveQuote(contentStr[lIndex:l2Index])
						a1.Accepted = lineSplited[len(lineSplited)-1] == "accepted"
						entry = a1
					}
					l2Index = strings.Index(contentStr, "with password")
					if l2Index > -1 {
						eventType = "password_auth"
						l3Index := l2Index + 13
						l4Index := strings.LastIndex(contentStr, " ")
						var a1 passwordAuthLog
						a1.User = TrimAndRemoveQuote(contentStr[lIndex:l2Index])
						a1.Password = TrimAndRemoveQuote(contentStr[l3Index:l4Index])
						a1.Accepted = lineSplited[len(lineSplited)-1] == "accepted"
						entry = a1
					}
					l2Index = strings.Index(contentStr, "with public key")
					if l2Index > -1 {
						eventType = "public_key_auth"
						l3Index := l2Index + 17
						l4Index := strings.LastIndex(contentStr, " ")
						var a1 publicKeyAuthLog
						a1.User = TrimAndRemoveQuote(contentStr[lIndex:l2Index])
						a1.PublicKeyFingerprint = TrimAndRemoveQuote(contentStr[l3Index:l4Index])
						a1.Accepted = lineSplited[len(lineSplited)-1] == "accepted"
						entry = a1
					}
					// I don't want to parse the keyboard interactive auth, fuck!
				}

				eventTypeCounter[eventType]++
				eventTypeId := eventTypeIdMap[eventType]
				logRecord := &bson.M{
					"time":        timeStr,
					"session_id":  0,
					"event_type":  eventTypeId,
					"source_ip":   ipAddr,
					"source_port": ipPort,
				}
				if !dryRun {
					LogEventToMongo(cfg.mongoRecorder, eventType, logRecord, entry)
				}
			}
		}
		if counter%1000 == 0 {
			infoLogger.Printf("Processed %d lines", counter)
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
