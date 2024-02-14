package triposter

import (
	"bytes"
	"encoding/json"
	"fmt"
	context "github.com/EcoPowerHub/context/pkg"
	"github.com/EcoPowerHub/shared/pkg/io"
	"github.com/EcoPowerHub/shared/pkg/objects"
	"net/http"
	"time"
)

type Triposter struct {
	batteryWaitToPost  []*objects.Battery
	metricWaitToPost   []*objects.Metric
	statusWaitToPost   []*objects.Status
	setpointWaitToPost []*objects.Setpoint
	conf               Configuration
	context            *context.Context
	processIsDone      bool
}

func New(configuration Configuration, c *context.Context) Triposter {
	return Triposter{conf: Configuration{}, context: c}
}

func (t *Triposter) Initialize() {
	t.processIsDone = false
	for _, object := range t.conf.Objects {
		fmt.Printf("object: %s\n", object.Ref)
		_, err := t.context.Get(object.Ref)
		if err != nil {
			return
		}
	}
	t.processListObject()
}

func (t *Triposter) processListObject() {
	t.StartInterval(t.conf.Conf.PostPeriodS)
	for _, object := range t.conf.Objects {
		switch object.Type {
		case io.KeyBattery:
			battery, err := t.context.Battery(object.Ref)
			if err != nil {
				return
			}
			t.batteryWaitToPost = append(t.batteryWaitToPost, battery)
		case io.KeyMetric:
			metric, err := t.context.Metric(object.Ref)
			if err != nil {
				return
			}
			t.metricWaitToPost = append(t.metricWaitToPost, metric)
		case io.KeyStatus:
			status, err := t.context.Status(object.Ref)
			if err != nil {
				return
			}
			t.statusWaitToPost = append(t.statusWaitToPost, status)
		case io.KeySetpoint:
			setPoint, err := t.context.Setpoint(object.Ref)
			if err != nil {
				return
			}
			t.setpointWaitToPost = append(t.setpointWaitToPost, setPoint)
		}
	}
	t.processIsDone = true

}

func (t *Triposter) StartInterval(intervalTime int) {
	//Toutes les intervalTime on post
	ticker := time.NewTicker(time.Duration(intervalTime))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.PostListsObject()
			if t.processIsDone {
				ticker.Stop()
			}
		}
	}
}

func (t *Triposter) ResetLists() {
	t.batteryWaitToPost = []*objects.Battery{}
	t.metricWaitToPost = []*objects.Metric{}
	t.statusWaitToPost = []*objects.Status{}
	t.setpointWaitToPost = []*objects.Setpoint{}
}

func (t *Triposter) PostListsObject() {
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
}
