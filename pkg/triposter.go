package triposter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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
	StatusUrl   = "/api/status"
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

func New(configuration Configuration, c *context.Context) Triposter {
	return Triposter{conf: configuration, context: c}
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
		t.Post(t.batteryWaitToPost, BatteryUrl)
		t.Post(t.metricWaitToPost, MetricUrl)
		t.Post(t.statusWaitToPost, StatusUrl)
		t.Post(t.setpointWaitToPost, SetpointUrl)
		t.Post(t.pvWaitToPost, PvUrl)
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

	// Vérification de la réponse
	if resp.StatusCode == http.StatusOK {
		t.logger.Info().Msg("data sent successfully")
		t.ResetLists()
	} else {
		t.logger.Error().Msg("data not sent")
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
