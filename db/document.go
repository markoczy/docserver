package db

import (
	"fmt"
	"time"

	"github.com/bvinc/go-sqlite-lite/sqlite3"
	"github.com/markoczy/docserver/domain/document"
	"github.com/pkg/errors"
)

const querySelectDocument = `SELECT id, uuid, name, created, last_modified FROM document WHERE uuid = ?`
const queryInsertDocument = `INSERT INTO document(uuid, name, created, last_modified) VALUES (?, ?, ?, ?)`
const queryUpdateDocument = `UPDATE document SET name = ?, created = ?, last_modified = ? WHERE id = ?`
const queryDeleteDocument = `DELETE FROM document WHERE uuid = ?`

const errPrepareFailed = "Failed to create PreparedStatement"
const errExecuteFailed = "Failed to execute PreparedStatement"
const errStepFailed = "Failed to step through result record"

func Connect(file string) (*sqlite3.Conn, error) {
	return sqlite3.Open(file)
}

func MustConnect(file string) *sqlite3.Conn {
	conn, err := sqlite3.Open(file)
	if err != nil {
		panic(err)
	}
	return conn
}

func CreateDocument(conn *sqlite3.Conn, doc document.Document) error {
	return conn.WithTx(func() error {
		var err error
		var stmt *sqlite3.Stmt
		if stmt, err = conn.Prepare(queryInsertDocument); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to create Document: %s", errPrepareFailed))
		}
		defer stmt.Close()

		if err = stmt.Exec(doc.Uuid(), doc.Name(), doc.Created(), doc.LastModified()); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to create Document: %s", errExecuteFailed))
		}
		return nil
	})
}

func ReadDocument(conn *sqlite3.Conn, uuid string) (document.Document, error) {
	var doc document.Document
	err := conn.WithTx(func() error {
		var (
			err    error
			stmt   *sqlite3.Stmt
			hasRow bool
		)
		if stmt, err = conn.Prepare(querySelectDocument); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to read Document %s: %s", uuid, errPrepareFailed))
		}
		defer stmt.Close()

		if err = stmt.Exec(uuid); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to read Document %s: %s", uuid, errExecuteFailed))
		}
		if hasRow, err = stmt.Step(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to read Document %s: %s", uuid, errStepFailed))
		}
		if !hasRow {
			return fmt.Errorf("No record found for uuid %s", uuid)
		}
		doc, err = readDocument(stmt)
		return err
	})
	return doc, err
}

func UpdateDocument(conn *sqlite3.Conn, doc document.Document) error {
	return conn.WithTx(func() error {
		var err error
		var stmt *sqlite3.Stmt
		if stmt, err = conn.Prepare(queryUpdateDocument); err != nil {
			return err
		}
		defer stmt.Close()

		if err = stmt.Exec(doc.Name(), doc.Created(), doc.LastModified(), doc.Id()); err != nil {
			return err
		}
		return nil
	})
}

func DeleteDocument(conn *sqlite3.Conn, uuid string) error {
	return conn.WithTx(func() error {
		var err error
		var stmt *sqlite3.Stmt
		if stmt, err = conn.Prepare(queryDeleteDocument); err != nil {
			return err
		}
		defer stmt.Close()

		if err = stmt.Exec(uuid); err != nil {
			return err
		}
		return nil
	})
}

func readDocument(stmt *sqlite3.Stmt) (doc document.Document, err error) {
	var (
		id, created, lastModified int64
		uuid, name                string
	)
	if err = stmt.Scan(&id, &uuid, &name, &created, &lastModified); err != nil {
		return
	}
	doc = document.NewBuilder().
		WithId(id).
		WithUuid(uuid).
		WithName(name).
		WithCreated(time.Unix(created, 0)).
		WithLastModified(time.Unix(lastModified, 0)).
		Build()
	return
}
