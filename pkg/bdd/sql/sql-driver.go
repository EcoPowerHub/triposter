package sql

import (
	sql "database/sql"
)

type SQLDriver struct {
	db   *sql.DB
	conf Conf
}

type Conf struct {
	Bdd string `json:"bdd"`
	Url string `json:"url"`
}
