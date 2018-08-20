package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/vharitonsky/iniflags"
)

type statEntry struct {
	bytes    int
	requests int
}

type stats struct {
	api   statEntry
	video statEntry
	image statEntry
	audio statEntry
	other statEntry
	dirty bool
}

var statsInstance = &stats{}
var cIp = make(chan string, 20)
var uniquesMap = make(map[string]int64)

func handleLog(entry LogEntry) {
	if strings.HasPrefix(entry.path, "/_/api.vk.com/") || strings.HasPrefix(entry.path, "/_/imv") || !strings.HasPrefix(entry.path, "/_") {
		statsInstance.api.requests++
		statsInstance.api.bytes += entry.length
	} else if strings.Contains(entry.path, ".mp4") || strings.Contains(entry.path, "vkuservideo") || strings.Contains(entry.path, "vkuserlive") {
		statsInstance.video.requests++
		statsInstance.video.bytes += entry.length
	} else if strings.Contains(entry.path, "vkuseraudio") || strings.Contains(entry.path, ".mp3") {
		statsInstance.audio.requests++
		statsInstance.audio.bytes += entry.length
	} else if strings.HasSuffix(entry.path, ".png") || strings.HasSuffix(entry.path, ".jpg") {
		statsInstance.image.requests++
		statsInstance.image.bytes += entry.length
	} else {
		statsInstance.other.requests++
		statsInstance.other.bytes += entry.length
	}
	statsInstance.dirty = true

	cIp <- entry.ip
}

func main() {
	syslogHost := flag.String("syslog-host", "0.0.0.0:7423", "address to bind syslog")
	influxUrl := flag.String("influx-url", "http://127.0.0.1:8086", "address of InfluxDB")
	influxDatabase := flag.String("influx-database", "vk_proxy", "database name")
	influxRetentionPolicy := flag.String("influx-rp", "a_day", "retention policy")
	influxUsername := flag.String("influx-username", "", "basic auth username")
	influxPassword := flag.String("influx-password", "", "basic auth password")

	iniflags.Parse()

	storage := newStorage(*influxUrl, *influxDatabase, *influxRetentionPolicy, *influxUsername, *influxPassword)

	startOnlineTicker()
	saveTicker := ticker{
		precision: time.Minute,
		callback: func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic: %s", recover())
				}
			}()
			log.Printf("Save initiated")
			online := len(uniquesMap)
			if online > 0 || statsInstance.dirty {
				s := statsInstance
				go func() {
					err := storage.save(s, online)
					if err != nil {
						log.Printf("save error: %s", err)
					}
				}()
				statsInstance = &stats{}
			}
		},
	}
	saveTicker.start()

	startSyslog(*syslogHost).Wait()
}

func startOnlineTicker() {
	ticker := time.Tick(5 * time.Second)
	go func() {
		for {
			select {
			case ip := <-cIp:
				uniquesMap[ip] = time.Now().Unix()
			case <-ticker:
				curr := time.Now().Unix()
				for ip, lastAccess := range uniquesMap {
					if curr-lastAccess > 3*60 {
						delete(uniquesMap, ip)
					}
				}
			}
		}
	}()
}
