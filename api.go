package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type BadgeResponse struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

func startApiServer(host string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/onlineBadge", handleOnlineBadge)
	go func() {
		err := http.ListenAndServe(host, mux)
		if err != nil {
			log.Printf("Could not start api server: %s", err)
		}
	}()
}

func handleOnlineBadge(w http.ResponseWriter, r *http.Request) {
	online := len(uniquesMap)
	res := BadgeResponse{
		SchemaVersion: 1,
		Label:         "online",
		Message:       strconv.Itoa(online),
		Color:         "brightgreen",
	}
	if online == 0 {
		res.Color = "yellow"
	}
	bytes, _ := json.Marshal(res)
	_, _ = w.Write(bytes)
}
