package triposter

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"time"

	"slices"

	context "github.com/EcoPowerHub/context/pkg"
	"github.com/EcoPowerHub/shared/pkg/io"
	"github.com/EcoPowerHub/shared/pkg/objects"
	"github.com/rs/zerolog"
)

const (
	BatteryUrl  = "/api/essMeasures"
	MetricUrl   = "/api/metrics"
	StatusUrl   = "/api/statuses"
	SetpointUrl = "/api/setpoints"
	PvUrl       = "/api/pvMeasures"
)

type Triposter struct {
	batteryWaitToPost  []*objects.Battery
	metricWaitToPost   []*objects.Metric
	statusWaitToPost   []*objects.Status
	setpointWaitToPost []*objects.Setpoint
	pvWaitToPost       []*objects.PV
	conf               Configuration
	context            *context.Context
	period             time.Duration
	logger             *zerolog.Logger
}

func New(configuration Configuration, c *context.Context, log zerolog.Logger) Triposter {
	return Triposter{conf: configuration, context: c, logger: &log}
}

func (t *Triposter) Configure() error {
	var err error
	t.period, err = time.ParseDuration(t.conf.Conf.Period)
	if err != nil {
		return fmt.Errorf("error parsing period: %w", err)
	}

	for _, object := range t.conf.Objects {
		_, err := t.context.Get(object.Ref)
		if err != nil {
			return fmt.Errorf("error getting object %s: %w", object.Ref, err)
		}
	}
	return nil
}

func (t *Triposter) Start() {
	for {
		t.Add()
		if len(t.batteryWaitToPost) == 0 {
			t.logger.Debug().Msg("no battery data to send")
		} else {
			t.logger.Debug().Msg("sending battery data")
			t.Post(t.batteryWaitToPost, BatteryUrl)
		}

		if len(t.metricWaitToPost) == 0 {
			t.logger.Debug().Msg("no metric data to send")
		} else {
			t.logger.Debug().Msg("sending metric data")
			t.Post(t.metricWaitToPost, MetricUrl)
		}

		if len(t.statusWaitToPost) == 0 {
			t.logger.Debug().Msg("no status data to send")
		} else {
			t.logger.Debug().Msg("sending status data")
			t.Post(t.statusWaitToPost, StatusUrl)
		}

		if len(t.setpointWaitToPost) == 0 {
			t.logger.Debug().Msg("no setpoint data to send")
		} else {
			t.logger.Debug().Msg("sending setpoint data")
			t.Post(t.setpointWaitToPost, SetpointUrl)
		}

		if len(t.pvWaitToPost) == 0 {
			t.logger.Debug().Msg("no pv data to send")
		} else {
			t.logger.Debug().Msg("sending pv data")
			t.Post(t.pvWaitToPost, PvUrl)
		}

		t.ResetLists()
		time.Sleep(t.period)
	}
}

func (t *Triposter) Post(objectToPost any, url string) {
	var fileName string
	switch url {
	case BatteryUrl:
		fileName = "battery.csv"
	case MetricUrl:
		fileName = "metric.csv"
	case StatusUrl:
		fileName = "status.csv"
	case SetpointUrl:
		fileName = "setpoint.csv"
	case PvUrl:
		fileName = "pv.csv"
	default:
		t.logger.Error().Msgf("unknown url: %s", url)
		return
	}

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.logger.Fatal().Err(err).Msg("error opening CSV file")
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		t.logger.Fatal().Err(err).Msg("error getting file info")
		return
	}
	empty := stat.Size() == 0

	writer := csv.NewWriter(file)
	defer writer.Flush()

	val := reflect.ValueOf(objectToPost)
	if val.Kind() != reflect.Slice || val.Len() == 0 {
		t.logger.Warn().Msg("empty or invalid data for CSV export")
		return
	}

	first := val.Index(0)
	elemType := first.Type()
	numFields := elemType.Elem().NumField()
	headers := make([]string, 0, numFields)

	for i := 0; i < numFields; i++ {
		field := elemType.Elem().Field(i)
		if field.PkgPath == "" { // exporté
			headers = append(headers, field.Name)
		}
	}

	if empty {
		// Annotation for InfluxDB
		annotations := [][]string{
			headers,                      // header row (field names)
			make([]string, len(headers)), // empty datatype annotation (to be defined if known)
			make([]string, len(headers)), // empty group annotation
			make([]string, len(headers)), // empty default annotation
		}

		// Add InfluxDB-specific annotations
		for i := range headers {
			switch headers[i] {
			case "Timestamp":
				annotations[1][i] = "dateTime:RFC3339"
				annotations[2][i] = ""
				annotations[3][i] = ""
			default:
				annotations[1][i] = "double"
				annotations[2][i] = ""
				annotations[3][i] = ""
			}
		}

		// Write annotation rows according to InfluxDB Annotated CSV specification
		datatypeRow := append([]string{"#datatype"}, annotations[1]...)
		groupRow := append([]string{"#group"}, annotations[2]...)
		defaultRow := append([]string{"#default"}, annotations[3]...)

		writer.Write(datatypeRow)
		writer.Write(groupRow)
		writer.Write(defaultRow)

		// Write header row
		writer.Write(headers)
	}

	for i := 0; i < val.Len(); i++ {
		rowVal := val.Index(i).Elem()
		row := make([]string, 0, len(headers))
		for j := 0; j < numFields; j++ {
			field := elemType.Elem().Field(j)
			if field.PkgPath != "" {
				continue
			}
			value := rowVal.Field(j)
			if value.Type().String() == "time.Time" {
				row = append(row, value.Interface().(time.Time).Format(time.RFC3339))
			} else {
				row = append(row, fmt.Sprintf("%v", value.Interface()))
			}
		}
		writer.Write(row)
	}
}

func (t *Triposter) Add() {
	for _, object := range t.conf.Objects {
		switch object.Type {
		case io.KeyBattery:
			battery, err := t.context.Battery(object.Ref)
			if err != nil {
				t.logger.Fatal().Str("ref", object.Ref).Err(err).Msg("error getting battery")
				continue
			}
			if slices.Contains(t.batteryWaitToPost, battery) {
				continue
			}
			battery.Source = object.Source
			battery.Site = t.conf.Conf.Site
			battery.Name = object.Name
			battery.Timestamp = time.Now()
			t.batteryWaitToPost = append(t.batteryWaitToPost, battery)

		case io.KeyMetric:
			metric, err := t.context.Metric(object.Ref)
			if err != nil {
				t.logger.Fatal().Str("ref", object.Ref).Err(err).Msg("error getting metric")
				continue
			}
			if slices.Contains(t.metricWaitToPost, metric) {
				continue
			}
			metric.Source = object.Source
			metric.Site = t.conf.Conf.Site
			metric.Name = object.Name
			metric.Timestamp = time.Now()
			t.metricWaitToPost = append(t.metricWaitToPost, metric)

		case io.KeyStatus:
			status, err := t.context.Status(object.Ref)
			if err != nil {
				t.logger.Fatal().Str("ref", object.Ref).Err(err).Msg("error getting status")
				continue
			}
			if slices.Contains(t.statusWaitToPost, status) {
				continue
			}
			status.Source = object.Source
			status.Site = t.conf.Conf.Site
			status.Name = object.Name
			status.Timestamp = time.Now()
			t.statusWaitToPost = append(t.statusWaitToPost, status)

		case io.KeySetpoint:
			setPoint, err := t.context.Setpoint(object.Ref)
			if err != nil {
				t.logger.Fatal().Str("ref", object.Ref).Err(err).Msg("error getting setpoint")
				continue
			}
			if slices.Contains(t.setpointWaitToPost, setPoint) {
				continue
			}
			setPoint.Source = object.Source
			setPoint.Site = t.conf.Conf.Site
			setPoint.Name = object.Name
			setPoint.Timestamp = time.Now()
			t.setpointWaitToPost = append(t.setpointWaitToPost, setPoint)

		case io.KeyPV:
			pv, err := t.context.Pv(object.Ref)
			if err != nil {
				t.logger.Fatal().Str("ref", object.Ref).Err(err).Msg("error getting pv")
				continue
			}
			if slices.Contains(t.pvWaitToPost, pv) {
				continue
			}
			pv.Source = object.Source
			pv.Site = t.conf.Conf.Site
			pv.Name = object.Name
			pv.Timestamp = time.Now()
			t.pvWaitToPost = append(t.pvWaitToPost, pv)
		}
	}
}

func (t *Triposter) ResetLists() {
	t.batteryWaitToPost = []*objects.Battery{}
	t.metricWaitToPost = []*objects.Metric{}
	t.statusWaitToPost = []*objects.Status{}
	t.setpointWaitToPost = []*objects.Setpoint{}
}
