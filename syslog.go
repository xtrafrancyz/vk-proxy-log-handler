package main

import (
	"regexp"
	"strconv"
	"time"

	"gopkg.in/mcuadros/go-syslog.v2"
)

type LogEntry struct {
	time    time.Time
	message string
	length  int
	method  string
	path    string
	ip      string
}

var logRegex = regexp.MustCompile(`([^ ]+) [^"]+"([A-Z]+) ([^ ]+) [^"]+" [0-9]+ ([0-9]+)`)

func startSyslog(host string) *syslog.Server {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC3164)
	server.SetHandler(handler)
	server.ListenUDP(host)
	server.Boot()

	go func(channel syslog.LogPartsChannel) {
		for record := range channel {
			entry := LogEntry{}
			entry.message = record["content"].(string)
			entry.time = record["timestamp"].(time.Time)
			match := logRegex.FindStringSubmatch(entry.message)
			if len(match) == 5 {
				entry.ip = match[1]
				entry.method = match[2]
				entry.path = match[3]
				entry.length, _ = strconv.Atoi(match[4])
				handleLog(entry)
			}
		}
	}(channel)

	return server
}
