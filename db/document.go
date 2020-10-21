package db

import (
	"fmt"
	"time"

	"github.com/bvinc/go-sqlite-lite/sqlite3"
	"github.com/markoczy/docserver/domain/document"
	"github.com/pkg/errors"
)

const querySelectDocuments = `SELECT id, uuid, name, created, last_modified FROM document`
const querySelectDocument = `SELECT id, uuid, name, created, last_modified FROM document WHERE uuid = ?`
const queryInsertDocument = `INSERT INTO document(uuid, name, created, last_modified) VALUES (?, ?, ?, ?)`
const queryUpdateDocument = `UPDATE document SET name = ?, created = ?, last_modified = ? WHERE id = ?`
const queryDeleteDocument = `DELETE FROM document WHERE uuid = ?`

const errPrepareFailed = "Failed to create PreparedStatement"
const errExecuteFailed = "Failed to execute PreparedStatement"
const errStepFailed = "Failed to step through result set"
const errScanFailed = "Failed to scan result set"

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

func ReadDocuments(conn *sqlite3.Conn) ([]document.Document, error) {
	var (
		doc    document.Document
		err    error
		stmt   *sqlite3.Stmt
		hasRow bool
		ret    []document.Document
	)
	err = conn.WithTx(func() error {
		if stmt, err = conn.Prepare(querySelectDocuments); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to read Documents: %s", errPrepareFailed))
		}
		defer stmt.Close()

		if err = stmt.Exec(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to read Documents: %s", errExecuteFailed))
		}

		for {
			hasRow, err = stmt.Step()
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to read Documents: %s", errStepFailed))
			}
			if !hasRow {
				break
			}
			if doc, err = readDocument(stmt); err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to read documents: %s", errStepFailed))
			}
			ret = append(ret, doc)
		}
		return nil
	})
	return ret, err
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

func readDocument(stmt *sqlite3.Stmt) (document.Document, error) {
	var (
		err                       error
		id, created, lastModified int64
		uuid, name                string
	)
	if err = stmt.Scan(&id, &uuid, &name, &created, &lastModified); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to read document: %s", errScanFailed))
	}
	doc := document.NewBuilder().
		WithId(id).
		WithUuid(uuid).
		WithName(name).
		WithCreated(time.Unix(created, 0)).
		WithLastModified(time.Unix(lastModified, 0)).
		Build()
	return doc, nil
}
