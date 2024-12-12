package config

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type DB struct {
	Conn         string        `env:"CONN,required"`
	MaxIdleConns int           `env:"MAX_IDLE_CONNS,default=2"`
	MaxOpenConns int           `env:"MAX_OPEN_CONNS,default=5"`
	MaxLifeConns time.Duration `env:"MAX_CONN_LIFE,default=10s"`
}

func OpenSQLXConn(cfg *DB) (db *sqlx.DB, err error) {
	db, err = sqlx.Open("postgres", cfg.Conn)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to open connection to '%s'", cfg.Conn)
	}

	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetConnMaxLifetime(cfg.MaxLifeConns)

	err = db.Ping()
	if err != nil {
		return nil, errors.Wrap(err, "db ping fail")
	}

	return db, nil
}
