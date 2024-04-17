package triposter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	stdio "io"
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
	data := struct {
		Objects any `json:"objects"`
	}{
		Objects: objectToPost,
	}
	// Encodage des données en JSON
	objectJson, err := json.Marshal(data)
	if err != nil {
		t.logger.Fatal().Err(err).Msg("error encoding JSON")
		return
	}

	// Envoi de la requête POST avec les données JSON
	resp, err := http.Post(t.conf.Conf.Host+url, "application/json", bytes.NewBuffer(objectJson))
	if err != nil {
		t.logger.Fatal().Err(err).Msg("error sending POST request")
		return
	}
	defer resp.Body.Close()

	_, err = stdio.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	// Vérification de la réponse
	if resp.StatusCode == http.StatusCreated {
		t.logger.Info().Msg("data sent successfully")
		t.ResetList(objectToPost)
	} else {
		t.logger.Error().Msgf("request failed with status code %d", resp.StatusCode)
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
			t.pvWaitToPost = append(t.pvWaitToPost, pv)
		}
	}
}

func (t *Triposter) ResetList(TypeOfObject any) {
	TypeOfObject = make([]any, 0)
}

func (t *Triposter) ResetLists() {
	t.batteryWaitToPost = []*objects.Battery{}
	t.metricWaitToPost = []*objects.Metric{}
	t.statusWaitToPost = []*objects.Status{}
	t.setpointWaitToPost = []*objects.Setpoint{}
	t.pvWaitToPost = []*objects.PV{}
}
