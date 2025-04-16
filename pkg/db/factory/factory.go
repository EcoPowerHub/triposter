package factory

import (
	"fmt"

	"github.com/EcoPowerHub/triposter/pkg/db"
	"github.com/EcoPowerHub/triposter/pkg/db/influx"
	"github.com/EcoPowerHub/triposter/pkg/db/mysql"
)

type DatabaseDriver interface {
	Connect() error
	Close() error
}

type DriverType string

const (
	MySQLDriver    DriverType = "mysql"
	InfluxDBDriver DriverType = "influxdb"
)

type DriverConfig struct {
	Type DriverType `json:"type"`
	Conf any        `json:"conf"`
}

func NewDatabaseDriver(config DriverConfig) (db.DatabaseReader, error) {
	switch config.Type {
	case MySQLDriver:
		conf, ok := config.Conf.(mysql.Conf)
		if !ok {
			return nil, fmt.Errorf("invalid configuration for MySQL driver")
		}
		return mysql.NewMySQLClient(conf), nil

	case InfluxDBDriver:
		conf, ok := config.Conf.(influx.Conf)
		if !ok {
			return nil, fmt.Errorf("invalid configuration for InfluxDB driver")
		}
		return influx.NewInfluxClient(conf), nil

	default:
		return nil, fmt.Errorf("unsupported driver type: %s", config.Type)
	}
}
