package factory

import (
	"context"
	"testing"
	"time"

	"github.com/EcoPowerHub/triposter/pkg/db"
	"github.com/EcoPowerHub/triposter/pkg/db/influx"
	"github.com/EcoPowerHub/triposter/pkg/db/mysql"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMySQLIntegration(t *testing.T) {
	ctx := context.Background()

	// Configuration du conteneur MySQL
	mysqlContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mysql:8",
			ExposedPorts: []string{"3306/tcp"},
			Env: map[string]string{
				"MYSQL_ROOT_PASSWORD": "password",
				"MYSQL_DATABASE":      "test",
			},
			WaitingFor: wait.ForLog("port: 3306  MySQL Community Server").WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	})
	assert.NoError(t, err)
	defer mysqlContainer.Terminate(ctx)

	host, err := mysqlContainer.Host(ctx)
	assert.NoError(t, err)
	port, err := mysqlContainer.MappedPort(ctx, "3306")
	assert.NoError(t, err)

	dsn := "root:password@tcp(" + host + ":" + port.Port() + ")/test"
	client, err := NewDatabaseDriver(DriverConfig{
		Type: MySQLDriver,
		Conf: mysql.Conf{
			DSN: dsn,
		},
	})

	assert.NoError(t, err)

	err = client.Connect(ctx)
	assert.NoError(t, err)
	defer client.Close()

	queryBuilder := client.NewQueryBuilder()
	query := queryBuilder.Select("*").From("information_schema.tables").Limit(1).Build()
	result, err := client.Query(ctx, query)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Columns)
}

func TestInfluxDBIntegration(t *testing.T) {
	ctx := context.Background()

	// Configuration du conteneur InfluxDB
	influxContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "influxdb:2.7",
			ExposedPorts: []string{"8086/tcp"},
			Env: map[string]string{
				"DOCKER_INFLUXDB_INIT_MODE":        "setup",
				"DOCKER_INFLUXDB_INIT_USERNAME":    "admin",
				"DOCKER_INFLUXDB_INIT_PASSWORD":    "password",
				"DOCKER_INFLUXDB_INIT_ORG":         "my-org",
				"DOCKER_INFLUXDB_INIT_BUCKET":      "test",
				"DOCKER_INFLUXDB_INIT_ADMIN_TOKEN": "my-token",
			},
			WaitingFor: wait.ForHTTP("/health").WithPort("8086/tcp").WithStatusCodeMatcher(
				func(status int) bool {
					return status == 200
				}),
		},
		Started: true,
	})
	assert.NoError(t, err)
	defer influxContainer.Terminate(ctx)

	// Récupération de l'adresse du conteneur
	host, err := influxContainer.Host(ctx)
	assert.NoError(t, err)
	port, err := influxContainer.MappedPort(ctx, "8086")
	assert.NoError(t, err)

	// Connexion au client InfluxDB
	url := "http://" + host + ":" + port.Port()
	client, err := NewDatabaseDriver(DriverConfig{
		Type: InfluxDBDriver,
		Conf: influx.Conf{
			Url:    url,
			Token:  "my-token",
			Org:    "my-org",
			Bucket: "test",
		},
	})

	assert.NoError(t, err)

	err = client.Connect(ctx)
	assert.NoError(t, err)
	defer client.Close()

	// Test d'écriture et de lecture de points
	point := write.NewPoint(
		"test_measurement", // Nom de la mesure
		map[string]string{ // Tags
			"tag1": "value1",
		},
		map[string]interface{}{ // Champs
			"value": 42,
		},
		time.Now(), // Timestamp
	)

	err = client.(db.PointWriter).WritePoint(ctx, point)
	assert.NoError(t, err)

	queryBuilder := client.NewQueryBuilder()
	query := queryBuilder.Select("*").From("test").Limit(1).Build()
	result, err := client.Query(ctx, query)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Rows)
}
