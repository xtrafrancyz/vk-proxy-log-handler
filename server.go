package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vharitonsky/iniflags"
)

type ContentType struct {
	Api   int `json:"api"`
	Video int `json:"video"`
	Image int `json:"image"`
	Audio int `json:"audio"`
	Other int `json:"other"`
	Total int `json:"total"`
}

type StatEntry struct {
	Online   int         `json:"online"`
	Traffic  ContentType `json:"traffic"`
	Requests ContentType `json:"requests"`
}

var data = make(map[int64]*StatEntry)
var cIp = make(chan string, 20)
var uniquesMap = make(map[string]int64)

func handleLog(entry LogEntry) {
	key := entry.time.Unix() - int64(entry.time.Second())

	stats, ok := data[key]
	if !ok {
		stats = &StatEntry{}
		for k := range data {
			if key-k > 2*24*60*60 {
				delete(data, k)
			}
		}
		data[key] = stats
	}
	stats.Requests.Total++
	stats.Traffic.Total += entry.length
	if strings.HasPrefix(entry.path, "/_/api.vk.com/") || strings.HasPrefix(entry.path, "/_/imv") || !strings.HasPrefix(entry.path, "/_") {
		stats.Requests.Api++
		stats.Traffic.Api += entry.length
	} else if strings.Contains(entry.path, "vkuservideo") || strings.Contains(entry.path, "vkuserlive") {
		stats.Requests.Video++
		stats.Traffic.Video += entry.length
	} else if strings.Contains(entry.path, "vkuseraudio") || strings.Contains(entry.path, ".mp3") {
		stats.Requests.Audio++
		stats.Traffic.Audio += entry.length
	} else if strings.Contains(entry.path, ".png") {
		stats.Requests.Image++
		stats.Traffic.Image += entry.length
	} else {
		stats.Requests.Other++
		stats.Traffic.Other += entry.length
	}

	cIp <- entry.ip
	stats.Online = len(uniquesMap)
}

func handleWeb(w http.ResponseWriter, r *http.Request) {
	encoded, _ := json.Marshal(data)
	fmt.Fprintf(w, string(encoded))
}

func main() {
	webHost := flag.String("web-host", "0.0.0.0:13554", "address to bind web")
	syslogHost := flag.String("syslog-host", "0.0.0.0:7423", "address to bind syslog")

	iniflags.Parse()

	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case ip := <-cIp:
				uniquesMap[ip] = time.Now().Unix()
			case <-ticker.C:
				curr := time.Now().Unix()
				for ip, lastAccess := range uniquesMap {
					if curr-lastAccess > 3*60 {
						delete(uniquesMap, ip)
					}
				}
			}
		}
	}()

	http.HandleFunc("/getData", handleWeb)
	go http.ListenAndServe(*webHost, nil)
	startSyslog(*syslogHost).Wait()
}
