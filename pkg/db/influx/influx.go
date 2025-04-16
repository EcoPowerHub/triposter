package influx

import (
	"context"
	"errors"

	"github.com/EcoPowerHub/triposter/pkg/db"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type InfluxClient struct {
	url, token, org, bucket string
	client                  influxdb2.Client
	queryAPI                api.QueryAPI
	writeAPI                api.WriteAPIBlocking
}

func NewInfluxClient(url, token, org, bucket string) *InfluxClient {
	return &InfluxClient{url: url, token: token, org: org, bucket: bucket}
}

func (c *InfluxClient) Connect(ctx context.Context) error {
	c.client = influxdb2.NewClient(c.url, c.token)
	c.queryAPI = c.client.QueryAPI(c.org)
	c.writeAPI = c.client.WriteAPIBlocking(c.org, c.bucket)
	_, err := c.client.Health(ctx)
	return err
}

func (c *InfluxClient) Close() error {
	c.client.Close()
	return nil
}

func (c *InfluxClient) Ping(ctx context.Context) error {
	_, err := c.client.Health(ctx)
	return err
}

func (c *InfluxClient) NewQueryBuilder() db.QueryBuilder {
	return &InfluxQueryBuilder{bucket: c.bucket}
}

func (c *InfluxClient) Query(ctx context.Context, q db.Query) (db.QueryResult, error) {
	r, err := c.queryAPI.Query(ctx, q.Raw)
	if err != nil {
		return db.QueryResult{}, err
	}
	var qr db.QueryResult
	for r.Next() {
		record := r.Record()
		row := []any{record.Time(), record.Field(), record.Value()}
		qr.Rows = append(qr.Rows, row)
	}
	qr.Columns = []string{"time", "field", "value"}
	return qr, nil
}

func (c *InfluxClient) WritePoint(ctx context.Context, point any) error {
	pt, ok := point.(*write.Point)
	if !ok {
		return errors.New("invalid point type for InfluxDB")
	}
	return c.writeAPI.WritePoint(ctx, pt)
}
