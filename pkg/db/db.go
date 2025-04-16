// --- Interfaces affinées ---

package db

import (
	"context"

	_ "github.com/go-sql-driver/mysql"
)

type Query struct {
	Raw  string
	Args []any
}

type QueryResult struct {
	Columns []string
	Rows    [][]any
}

type QueryBuilder interface {
	Select(fields ...string) QueryBuilder
	From(source string) QueryBuilder
	Where(condition string) QueryBuilder
	GroupBy(fields ...string) QueryBuilder
	OrderBy(field string, asc bool) QueryBuilder
	Limit(n int) QueryBuilder
	Build() Query
}

type DatabaseReader interface {
	Connect(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error
	NewQueryBuilder() QueryBuilder
	Query(ctx context.Context, q Query) (QueryResult, error)
}

type DatabaseWriter interface {
	Write(ctx context.Context, q Query) error
}

type PointWriter interface {
	WritePoint(ctx context.Context, point any) error
}

// --- InfluxDB Implementation ---

// --- Influx QueryBuilder ---
