package triposter

import (
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
	var measurement string
	switch url {
	case BatteryUrl:
		measurement = "battery"
	case MetricUrl:
		measurement = "metric"
	case StatusUrl:
		measurement = "status"
	case SetpointUrl:
		measurement = "setpoint"
	case PvUrl:
		measurement = "pv"
	default:
		t.logger.Error().Msgf("unknown url: %s", url)
		return
	}

	val := reflect.ValueOf(objectToPost)
	if val.Kind() != reflect.Slice || val.Len() == 0 {
		t.logger.Warn().Msg("empty or invalid data for Line Protocol export")
		return
	}

	fileName := measurement + ".lp"
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.logger.Fatal().Err(err).Msg("error opening Line Protocol file")
		return
	}
	defer file.Close()

	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i).Elem()
		elemType := elem.Type()

		// Prepare tags and fields
		tags := make(map[string]string)
		fieldsNumeric := make(map[string]string)
		fieldsString := make(map[string]string)
		var timestamp time.Time

		for j := 0; j < elem.NumField(); j++ {
			field := elemType.Field(j)
			if field.PkgPath != "" { // unexported
				continue
			}
			fieldName := field.Name
			fieldValue := elem.Field(j).Interface()

			switch fieldName {
			case "Site":
				tags["site"] = fmt.Sprintf("%v", fieldValue)
			case "Source":
				tags["source"] = fmt.Sprintf("%v", fieldValue)
			case "Name":
				tags["name"] = fmt.Sprintf("%v", fieldValue)
			case "Timestamp":
				if ts, ok := fieldValue.(time.Time); ok {
					timestamp = ts
				}
			default:
				// Determine field type to separate numeric and string fields
				switch val := fieldValue.(type) {
				case string:
					// Escape double quotes and backslashes
					escaped := stringReplaceAll(val, "\\", "\\\\")
					escaped = stringReplaceAll(escaped, "\"", "\\\"")
					fieldsString[fieldName] = fmt.Sprintf("\"%s\"", escaped)
				case int, int8, int16, int32, int64:
					fieldsNumeric[fieldName] = fmt.Sprintf("%di", val)
				case uint, uint8, uint16, uint32, uint64:
					fieldsNumeric[fieldName] = fmt.Sprintf("%di", val)
				case float32, float64:
					fieldsNumeric[fieldName] = fmt.Sprintf("%f", val)
				case bool:
					fieldsNumeric[fieldName] = fmt.Sprintf("%t", val)
				case time.Time:
					// Represent as UnixNano integer
					fieldsNumeric[fieldName] = fmt.Sprintf("%di", val.UnixNano())
				default:
					fieldsString[fieldName] = fmt.Sprintf("\"%v\"", val)
				}
			}
		}

		// Build tag set string
		tagSet := ""
		for k, v := range tags {
			// Escape comma, space, and equals in tag values
			escapedValue := v
			escapedValue = stringReplaceAll(escapedValue, ",", "\\,")
			escapedValue = stringReplaceAll(escapedValue, " ", "\\ ")
			escapedValue = stringReplaceAll(escapedValue, "=", "\\=")
			if tagSet == "" {
				tagSet = fmt.Sprintf("%s=%s", k, escapedValue)
			} else {
				tagSet = fmt.Sprintf("%s,%s=%s", tagSet, k, escapedValue)
			}
		}

		// Build field set string, numeric fields first, then string fields
		fieldSet := ""
		for k, v := range fieldsNumeric {
			if fieldSet == "" {
				fieldSet = fmt.Sprintf("%s=%s", k, v)
			} else {
				fieldSet = fmt.Sprintf("%s,%s=%s", fieldSet, k, v)
			}
		}
		for k, v := range fieldsString {
			if fieldSet == "" {
				fieldSet = fmt.Sprintf("%s=%s", k, v)
			} else {
				fieldSet = fmt.Sprintf("%s,%s=%s", fieldSet, k, v)
			}
		}

		// Use timestamp in nanoseconds, fallback to current time if zero
		tsInt := timestamp.UnixNano()
		if tsInt == 0 {
			tsInt = time.Now().UnixNano()
		}

		line := fmt.Sprintf("%s", measurement)
		if tagSet != "" {
			line += "," + tagSet
		}
		if fieldSet != "" {
			line += " " + fieldSet
		} else {
			// No fields, skip line
			continue
		}
		line += fmt.Sprintf(" %d\n", tsInt)

		if _, err := file.WriteString(line); err != nil {
			t.logger.Error().Err(err).Msg("error writing to Line Protocol file")
		}
	}
}

func stringReplaceAll(s, old, new string) string {
	// simple wrapper to avoid importing strings package
	// since only used here
	r := []rune{}
	oldRunes := []rune(old)
	newRunes := []rune(new)
	for i := 0; i < len(s); {
		if i+len(oldRunes) <= len(s) && s[i:i+len(oldRunes)] == old {
			r = append(r, newRunes...)
			i += len(oldRunes)
		} else {
			r = append(r, rune(s[i]))
			i++
		}
	}
	return string(r)
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
