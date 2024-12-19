package bdd

type DatabaseDriver interface {
	// Connexion et déconnexion
	Connect(connectionString string) error
	Disconnect() error

	// Opérations CRUD de base
	Create(table string, data map[string]interface{}) (int64, error)
	Read(table string, filters map[string]interface{}) ([]map[string]interface{}, error)
	Update(table string, filters map[string]interface{}, data map[string]interface{}) (int64, error)
	Delete(table string, filters map[string]interface{}) (int64, error)

	// Fonctions utilitaires
	ExecuteQuery(query string, args ...interface{}) (interface{}, error)
	Ping() error
	BeginTransaction() (Transaction, error)
}

// Interface pour les transactions
type Transaction interface {
	Commit() error
	Rollback() error
	ExecuteQuery(query string, args ...interface{}) (interface{}, error)
}
