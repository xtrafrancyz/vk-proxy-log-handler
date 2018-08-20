package main

import (
	"log"
	"time"
)

type ticker struct {
	precision time.Duration
	callback  func()
}

func (t *ticker) start() {
	now := time.Now()
	startAt := now.Round(t.precision)
	if startAt.Before(now) {
		startAt = startAt.Add(t.precision)
	}
	sleepDuration := startAt.Sub(now)

	log.Printf("First save will be after %s", sleepDuration)

	time.AfterFunc(sleepDuration, func() {
		t.callback()
		ticker := time.NewTicker(t.precision)
		go func() {
			for range ticker.C {
				t.callback()
			}
		}()
	})

	time.NewTimer(time.Second)
}
