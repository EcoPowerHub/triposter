package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/EcoPowerHub/triposter/pkg/db"
)

// --- MySQL Implementation ---

type MySQLClient struct {
	dsn string
	db  *sql.DB
}

type Conf struct {
	DSN string `json:"dsn"`
}

func NewMySQLClient(c Conf) *MySQLClient {
	return &MySQLClient{dsn: c.DSN}
}

func (c *MySQLClient) Connect(ctx context.Context) error {
	db, err := sql.Open("mysql", c.dsn)
	if err != nil {
		return fmt.Errorf("MySQL connect error: %w", err)
	}
	c.db = db
	return c.db.PingContext(ctx)
}

func (c *MySQLClient) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *MySQLClient) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *MySQLClient) NewQueryBuilder() db.QueryBuilder {
	return &MySQLQueryBuilder{}
}

func (c *MySQLClient) Query(ctx context.Context, q db.Query) (db.QueryResult, error) {
	rows, err := c.db.QueryContext(ctx, q.Raw, q.Args...)
	if err != nil {
		return db.QueryResult{}, fmt.Errorf("MySQL query error: %w", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	result := db.QueryResult{Columns: cols}

	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return result, err
		}
		result.Rows = append(result.Rows, vals)
	}
	return result, nil
}

func (c *MySQLClient) Write(ctx context.Context, q db.Query) error {
	_, err := c.db.ExecContext(ctx, q.Raw, q.Args...)
	return err
}
