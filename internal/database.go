package internal

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db *sqlx.DB
)

type CatData struct {
	ID       int    `db:"id"`
	CatID    string `db:"cat_id"`
	FileType string `db:"file_type"`
	Data     []byte `db:"data"`
}

const schema string = `
    CREATE TABLE IF NOT EXISTS cats (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        cat_id TEXT NOT NULL UNIQUE,
        data BLOB NOT NULL,
        file_type TEXT NOT NULL
    );
`

func NewDatabase(path string) (*sqlx.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sqlx.ConnectContext(ctx, "sqlite3", path)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	log.Info("connected to database", "dbpath", path)

	return db, nil
}
