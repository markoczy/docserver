package db

import (
	"fmt"
	"time"

	"github.com/bvinc/go-sqlite-lite/sqlite3"
	"github.com/markoczy/docserver/types/document"
	"github.com/pkg/errors"
)

const querySelectDocument = `SELECT id, uuid, name, created, last_modified FROM document WHERE uuid = ?`
const queryInsertDocument = `INSERT INTO document(uuid, name, created, last_modified) VALUES (?, ?, ?, ?)`
const queryUpdateDocument = `UPDATE document SET name = ?, created = ?, last_modified = ? WHERE id = ?`
const queryDeleteDocument = `DELETE FROM document WHERE uuid = ?`

type Handler interface {
	Init() error
	Close() error
	ProcessUpdates() error
	// CRUD For document
	CreateDocument(doc document.Document) error
	ReadDocument(uuid string) (document.Document, error)
	UpdateDocument(doc document.Document) error
	DeleteDocument(uuid string) error
}

func NewHandler(file string) Handler {
	return &dbHandler{
		file: file,
	}
}

type dbHandler struct {
	file string
	conn *sqlite3.Conn
}

func (db *dbHandler) Init() error {
	var err error
	db.conn, err = sqlite3.Open(db.file)
	return err
}

func (db *dbHandler) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

func (db *dbHandler) ProcessUpdates() error {
	for k, v := range dbUpdates {
		err := db.conn.WithTx(func() error {
			return db.conn.Exec(v)
		})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to process update %s", k))
		}
	}
	return nil
}

func (db *dbHandler) CreateDocument(doc document.Document) error {
	return db.conn.WithTx(func() error {
		var err error
		var stmt *sqlite3.Stmt
		if stmt, err = db.conn.Prepare(queryInsertDocument); err != nil {
			return err
		}
		defer stmt.Close()

		if err = stmt.Exec(doc.Uuid(), doc.Name(), doc.Created(), doc.LastModified()); err != nil {
			return err
		}
		return nil
	})
}

func (db *dbHandler) ReadDocument(uuid string) (document.Document, error) {
	var doc document.Document
	err := db.conn.WithTx(func() error {
		var (
			err    error
			stmt   *sqlite3.Stmt
			hasRow bool
		)
		if stmt, err = db.conn.Prepare(querySelectDocument); err != nil {
			return err
		}
		defer stmt.Close()

		if err = stmt.Exec(uuid); err != nil {
			return err
		}
		if hasRow, err = stmt.Step(); err != nil {
			return err
		}
		if !hasRow {
			return fmt.Errorf("No record found for uuid %s", uuid)
		}
		doc, err = db.readDocument(stmt)
		return err
	})
	return doc, err
}

func (db *dbHandler) UpdateDocument(doc document.Document) error {
	return db.conn.WithTx(func() error {
		var err error
		var stmt *sqlite3.Stmt
		if stmt, err = db.conn.Prepare(queryUpdateDocument); err != nil {
			return err
		}
		defer stmt.Close()

		if err = stmt.Exec(doc.Name(), doc.Created(), doc.LastModified(), doc.Id()); err != nil {
			return err
		}
		return nil
	})
}

func (db *dbHandler) DeleteDocument(uuid string) error {
	return db.conn.WithTx(func() error {
		var err error
		var stmt *sqlite3.Stmt
		if stmt, err = db.conn.Prepare(queryDeleteDocument); err != nil {
			return err
		}
		defer stmt.Close()

		if err = stmt.Exec(uuid); err != nil {
			return err
		}
		return nil
	})
}

func (db *dbHandler) readDocument(stmt *sqlite3.Stmt) (doc document.Document, err error) {
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
