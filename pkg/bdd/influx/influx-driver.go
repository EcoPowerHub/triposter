package influx

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

type InfluxDBDriver struct {
	token  string
	client influxdb2.Client
	conf   Conf
}

type Conf struct {
	Org    string `json:"org"`
	Bucket string `json:"bucket"`
	Url    string `json:"url"`
}
