package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/phuslu/iploc"
	"github.com/vharitonsky/iniflags"
)

type statEntry struct {
	bytes    int
	requests int
}

type stats struct {
	api       statEntry
	video     statEntry
	image     statEntry
	audio     statEntry
	other     statEntry
	dirty     bool
	countries map[string]*statEntry
}

func newStats() *stats {
	return &stats{
		countries: make(map[string]*statEntry),
	}
}

var statsInstance = newStats()
var cIp = make(chan string, 20)
var uniquesMap = make(map[string]int64)

func handleLog(entry LogEntry) {
	if strings.HasPrefix(entry.path, "/@") ||
		strings.HasPrefix(entry.path, "/_/api.vk.com") ||
		strings.HasPrefix(entry.path, "/_/imv") ||
		!strings.HasPrefix(entry.path, "/_") {

		statsInstance.api.requests++
		statsInstance.api.bytes += entry.length

	} else if strings.Contains(entry.path, ".mp4") ||
		strings.Contains(entry.path, "vkuservideo") ||
		strings.Contains(entry.path, "vkuserlive") {

		statsInstance.video.requests++
		statsInstance.video.bytes += entry.length

	} else if strings.Contains(entry.path, "vkuseraudio") ||
		strings.Contains(entry.path, ".mp3") {

		statsInstance.audio.requests++
		statsInstance.audio.bytes += entry.length

	} else if strings.HasSuffix(entry.path, ".png") ||
		strings.HasSuffix(entry.path, ".jpg") ||
		strings.HasPrefix(entry.path, "/_/vk.com/sticker") {

		statsInstance.image.requests++
		statsInstance.image.bytes += entry.length

	} else {
		statsInstance.other.requests++
		statsInstance.other.bytes += entry.length
	}

	ip := net.ParseIP(entry.ip)
	if ip != nil {
		country := string(iploc.Country(ip))
		e, ok := statsInstance.countries[country]
		if !ok {
			e = &statEntry{}
			statsInstance.countries[country] = e
		}
		e.requests++
		e.bytes += entry.length
	}

	statsInstance.dirty = true

	cIp <- entry.ip
}

func main() {
	syslogHost := flag.String("syslog-host", "0.0.0.0:7423", "address to bind syslog (UDP)")
	apiHost := flag.String("api-host", "127.0.0.1:8083", "address to bind api server (TCP)")
	pprofHost := flag.String("pprof-host", "", "address to bind pprof (TCP)")
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
				statsInstance = newStats()
				go func() {
					err := storage.save(s, online)
					if err != nil {
						log.Printf("save error: %s", err)
					}
				}()
			}
		},
	}
	saveTicker.start()

	if *pprofHost != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		go func() {
			err := http.ListenAndServe(*pprofHost, mux)
			if err != nil {
				log.Printf("Could not start pprof server: %s", err)
			}
		}()
	}

	startApiServer(*apiHost)
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
