package db

import (
	"fmt"

	"github.com/bvinc/go-sqlite-lite/sqlite3"
	"github.com/pkg/errors"
)

// ALL UPDATES MUST BE IDEMPOTENT ON SECOND CALL
var dbUpdates = map[string]string{
	"0.0.1-create-document-table": `CREATE TABLE IF NOT EXISTS document(
		id    			INTEGER PRIMARY KEY,
		uuid 			TEXT UNIQUE,
		name    		TEXT UNIQUE,
		created 		INTEGER,
		last_modified 	INTEGER
	);`,
	"0.0.2-insert-test-document": `REPLACE INTO document(uuid, name, created, last_modified) VALUES(
		"be48a070-451c-4622-8265-4d51aab78a71",
		"test",
		1603042213,
		1603042213
	);`,
}

func ProcessUpdates(conn *sqlite3.Conn) error {
	for k, v := range dbUpdates {
		err := conn.WithTx(func() error {
			return conn.Exec(v)
		})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to process update %s", k))
		}
	}
	return nil
}
