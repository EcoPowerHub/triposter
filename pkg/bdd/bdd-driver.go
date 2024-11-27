package bdd

type DatabaseDriver interface {
	// Connexion et déconnexion
	Connect(connectionString string) error
	Disconnect() error

	// Opérations CRUD de base
	Create(table string, data map[string]interface{}) (int64, error)
	Read(table string, filters map[string]interface{}) ([]map[string]interface{}, error)
	Delete(table string, filters map[string]interface{}) (int64, error)

	Configure(conf any) error
}
