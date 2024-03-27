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

type Triposter struct {
	batteryWaitToPost  []*objects.Battery
	metricWaitToPost   []*objects.Metric
	statusWaitToPost   []*objects.Status
	setpointWaitToPost []*objects.Setpoint
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
		// Création d'une structure pour contenir les trois listes
		data := struct {
			ListBattery  []*objects.Battery  `json:"listBattery"`
			ListMetric   []*objects.Metric   `json:"listMetric"`
			ListStatus   []*objects.Status   `json:"listStatus"`
			ListSetpoint []*objects.Setpoint `json:"listSetpoint"`
		}{
			ListBattery:  t.batteryWaitToPost,
			ListMetric:   t.metricWaitToPost,
			ListStatus:   t.statusWaitToPost,
			ListSetpoint: t.setpointWaitToPost,
		}

		// Encodage des données en JSON
		jsonData, err := json.Marshal(data)
		if err != nil {
			fmt.Println("Erreur lors de l'encodage JSON:", err)
			return
		}

		// Envoi de la requête POST avec les données JSON
		resp, err := http.Post("URL", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("Erreur lors de l'envoi de la requête POST:", err)
			return
		}
		defer resp.Body.Close()

		// Vérification de la réponse
		if resp.StatusCode == http.StatusOK {
			fmt.Println("Données envoyées avec succès.")
			t.ResetLists()
		} else {
			fmt.Println("La requête POST a échoué avec le code de statut:", resp.StatusCode)
		}
		time.Sleep(t.period)
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
			t.setpointWaitToPost = append(t.setpointWaitToPost, setPoint)
		}
	}
}

func (t *Triposter) ResetLists() {
	t.batteryWaitToPost = []*objects.Battery{}
	t.metricWaitToPost = []*objects.Metric{}
	t.statusWaitToPost = []*objects.Status{}
	t.setpointWaitToPost = []*objects.Setpoint{}
}
