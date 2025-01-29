package config

import (
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"time"
)

type DB struct {
	Conn string `env:"CONN,required"`
}

const (
	maxIdleConns = 2
	maxOpenConns = 5
	maxLifeConns = 10
)

func OpenSQLXConn(cfg *DB) (db *sqlx.DB, err error) {
	db, err = sqlx.Open("postgres", cfg.Conn)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to open connection to '%s'", cfg.Conn)
	}

	db.SetMaxIdleConns(maxIdleConns)
	db.SetMaxOpenConns(maxOpenConns)
	db.SetConnMaxLifetime(maxLifeConns * time.Second)

	err = db.Ping()
	if err != nil {
		return nil, errors.Wrap(err, "db ping fail")
	}

	return db, nil
}
