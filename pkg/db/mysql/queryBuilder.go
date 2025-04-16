package mysql

import (
	"fmt"
	"strings"

	"github.com/EcoPowerHub/triposter/pkg/db"
)

type MySQLQueryBuilder struct {
	selects  []string
	from     string
	where    string
	groupBy  []string
	orderBy  string
	orderAsc bool
	limit    int
}

func (b *MySQLQueryBuilder) Select(fields ...string) db.QueryBuilder {
	b.selects = fields
	return b
}
func (b *MySQLQueryBuilder) From(source string) db.QueryBuilder {
	b.from = source
	return b
}
func (b *MySQLQueryBuilder) Where(condition string) db.QueryBuilder {
	b.where = condition
	return b
}
func (b *MySQLQueryBuilder) GroupBy(fields ...string) db.QueryBuilder {
	b.groupBy = fields
	return b
}
func (b *MySQLQueryBuilder) OrderBy(field string, asc bool) db.QueryBuilder {
	b.orderBy = field
	b.orderAsc = asc
	return b
}
func (b *MySQLQueryBuilder) Limit(n int) db.QueryBuilder {
	b.limit = n
	return b
}
func (b *MySQLQueryBuilder) Build() db.Query {
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(b.selects, ", "), b.from)
	if b.where != "" {
		query += " WHERE " + b.where
	}
	if len(b.groupBy) > 0 {
		query += " GROUP BY " + strings.Join(b.groupBy, ", ")
	}
	if b.orderBy != "" {
		dir := "DESC"
		if b.orderAsc {
			dir = "ASC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", b.orderBy, dir)
	}
	if b.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", b.limit)
	}
	return db.Query{Raw: query, Args: nil}
}
