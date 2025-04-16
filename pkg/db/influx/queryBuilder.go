package influx

import (
	"fmt"
	"strings"

	"github.com/EcoPowerHub/triposter/pkg/db"
)

type InfluxQueryBuilder struct {
	selects  []string
	bucket   string
	where    string
	groupBy  []string
	orderBy  string
	orderAsc bool
	limit    int
}

func (b *InfluxQueryBuilder) Select(fields ...string) db.QueryBuilder {
	b.selects = fields
	return b
}
func (b *InfluxQueryBuilder) From(source string) db.QueryBuilder {
	b.bucket = source
	return b
}
func (b *InfluxQueryBuilder) Where(condition string) db.QueryBuilder {
	b.where = condition
	return b
}
func (b *InfluxQueryBuilder) GroupBy(fields ...string) db.QueryBuilder {
	b.groupBy = fields
	return b
}
func (b *InfluxQueryBuilder) OrderBy(field string, asc bool) db.QueryBuilder {
	b.orderBy = field
	b.orderAsc = asc
	return b
}
func (b *InfluxQueryBuilder) Limit(n int) db.QueryBuilder {
	b.limit = n
	return b
}
func (b *InfluxQueryBuilder) Build() db.Query {
	query := fmt.Sprintf(`from(bucket: "%s") |> range(start: -1h)`, b.bucket)
	if b.where != "" {
		query += fmt.Sprintf(" |> filter(fn: (r) => %s)", b.where)
	}
	if len(b.groupBy) > 0 {
		query += fmt.Sprintf(" |> group(columns: [%s])", strings.Join(b.groupBy, ", "))
	}
	if b.orderBy != "" {
		query += fmt.Sprintf(" |> sort(columns:[\"%s\"], desc:%v)", b.orderBy, !b.orderAsc)
	}
	if b.limit > 0 {
		query += fmt.Sprintf(" |> limit(n:%d)", b.limit)
	}
	return db.Query{Raw: query, Args: nil}
}
