package main

import (
	"log"
	"os"

	influx "github.com/influxdata/influxdb1-client/v2"
)

type storage struct {
	client            influx.Client
	batchPointsConfig influx.BatchPointsConfig
}

func newStorage(url, database, retentionPolicy, username, password string) *storage {
	influxClient, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     url,
		Username: username,
		Password: password,
	})
	if err != nil {
		log.Println("Error creating InfluxDB Client: ", err.Error())
		os.Exit(0)
	}

	return &storage{
		client: influxClient,
		batchPointsConfig: influx.BatchPointsConfig{
			Precision:       "s",
			Database:        database,
			RetentionPolicy: retentionPolicy,
		},
	}
}

func (s *storage) save(stats *stats, online int) error {
	bp, err := influx.NewBatchPoints(s.batchPointsConfig)
	if err != nil {
		return err
	}
	bp.AddPoint(s.createTrafficPoint("api", stats.api))
	bp.AddPoint(s.createTrafficPoint("audio", stats.audio))
	bp.AddPoint(s.createTrafficPoint("image", stats.image))
	bp.AddPoint(s.createTrafficPoint("video", stats.video))
	bp.AddPoint(s.createTrafficPoint("other", stats.other))

	point, _ := influx.NewPoint("online", nil, map[string]interface{}{
		"value": online,
	})
	bp.AddPoint(point)

	return s.client.Write(bp)
}

func (s *storage) createTrafficPoint(typeStr string, entry statEntry) *influx.Point {
	point, _ := influx.NewPoint("traffic", map[string]string{
		"type": typeStr,
	}, map[string]interface{}{
		"requests": entry.requests,
		"bytes":    entry.bytes,
	})
	return point
}
