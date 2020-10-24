package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"text/template"
	"time"

	"github.com/bvinc/go-sqlite-lite/sqlite3"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/markoczy/docserver/db"
	"github.com/markoczy/docserver/domain/document"
	"github.com/markoczy/webapi"
	"github.com/pkg/errors"
)

type model struct {
	Header   *headerModel
	Body     interface{}
	Template *template.Template
}

type headerModel struct {
	ActivePage string
}

type viewDocumentModel struct {
	Document document.Document
	Content  string
}

type editDocumentModel struct {
	Document document.Document
	Content  string
}

type viewDocumentsModel struct {
	Documents []document.Document
}

func InitDocumentController(router *webapi.Router, conn *sqlite3.Conn) error {
	router.Handle(http.MethodGet, "/document", webapi.NewHandler(
		handleLogRequest,
		handleInitTemplate("document"),
		handleViewDocuments(conn),
	))
	router.Handle(http.MethodGet, "/document/:uuid/view", webapi.NewHandler(
		handleLogRequest,
		handleInitTemplate("document"),
		handleViewDocument(conn),
	))
	router.Handle(http.MethodGet, "/document/:uuid/edit", webapi.NewHandler(
		handleLogRequest,
		handleInitTemplate("document"),
		handleEditDocument(conn),
	))
	router.Handle(http.MethodPost, "/document/save", webapi.NewHandler(
		handleLogRequest,
		handleSaveDocument(conn),
		webapi.NewNativeHandlerFunc(http.RedirectHandler("/document", http.StatusFound)),
	))
	return nil
}

func InitAssetController(router *webapi.Router) {
	assetServer := http.StripPrefix("/asset", http.FileServer(http.Dir("./data/asset")))
	handleAsset := webapi.NewHandler(
		handleLogRequest,
		func(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
			assetServer.ServeHTTP(w, r.Request)
			return next()
		},
	)
	router.Handle(http.MethodGet, "/asset/(.*)", handleAsset)
}

func handleLogRequest(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
	log.Printf("Request: %s %s\n", r.Request.Method, r.Request.RequestURI)
	return next()
}

func handleError(err error) webapi.Handler {
	return webapi.NewErrorHandler(http.StatusInternalServerError, err.Error())
}

func handleBadRequest(err string) webapi.Handler {
	return webapi.NewErrorHandler(http.StatusBadRequest, err)
}

func handleInitTemplate(activePage string) webapi.HandlerFunc {
	return func(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
		defer recoverPanic(w)
		tmpl := template.Must(template.ParseFiles(
			"data/template/main.html",
			"data/template/header.html",
		))
		r.State = &model{
			Template: tmpl,
			Header: &headerModel{
				ActivePage: activePage,
			},
		}
		return next()
	}
}

func handleViewDocuments(conn *sqlite3.Conn) webapi.HandlerFunc {
	return func(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
		defer recoverPanic(w)
		var err error
		var docs []document.Document

		// Load data from db
		if docs, err = db.ReadDocuments(conn); err != nil {
			panic(errors.Wrap(err, "Failed to read documents from DB"))
		}
		for _, v := range docs {
			log.Printf("Doc: %v\n", v)
		}
		state := r.State.(*model)
		state.Body = viewDocumentsModel{
			Documents: docs,
		}

		tmpl := template.Must(state.Template.ParseFiles(
			"data/template/view-documents.html",
		))
		if err = tmpl.Execute(w, state); err != nil {
			panic(err)
		}
		return next()
	}
}

func handleViewDocument(conn *sqlite3.Conn) webapi.HandlerFunc {
	return func(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
		defer recoverPanic(w)
		var err error
		var doc document.Document
		uuid := r.PathParams["uuid"]

		// Load data from db
		if doc, err = db.ReadDocument(conn, uuid); err != nil {
			panic(errors.Wrap(err, "Failed to read document from DB"))
		}
		// Load content from file
		content := loadDocumentAsHtml(uuid)
		state := r.State.(*model)
		state.Body = viewDocumentModel{
			Document: doc,
			Content:  content,
		}

		tmpl := template.Must(state.Template.ParseFiles(
			"data/template/view-document.html",
		))
		if err = tmpl.Execute(w, state); err != nil {
			panic(err)
		}
		return next()
	}
}

func handleEditDocument(conn *sqlite3.Conn) webapi.HandlerFunc {
	return func(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
		defer recoverPanic(w)
		var err error
		var doc document.Document
		uuid := r.PathParams["uuid"]

		// Load data from db
		if doc, err = db.ReadDocument(conn, uuid); err != nil {
			panic(errors.Wrap(err, "Failed to read document from DB"))
		}
		// Load content from file
		content := loadDocument(uuid)
		state := r.State.(*model)
		state.Body = editDocumentModel{
			Document: doc,
			Content:  content,
		}

		tmpl := template.Must(state.Template.ParseFiles(
			"data/template/edit-document.html",
		))
		if err = tmpl.Execute(w, state); err != nil {
			panic(err)
		}

		return next()
	}
}

func handleSaveDocument(conn *sqlite3.Conn) webapi.HandlerFunc {
	return func(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
		defer recoverPanic(w)
		var err error
		if err = r.Request.ParseForm(); err != nil {
			panic(errors.Wrap(err, "Could not parse request form"))
		}

		uuid := r.Request.PostFormValue("uuid")
		name := r.Request.PostFormValue("name")
		content := r.Request.PostFormValue("content")
		log.Printf("uuid: %s, name: %s", uuid, name)
		doc := document.NewBuilder().
			WithUuid(uuid).
			WithName(name).
			WithLastModified(time.Now()).
			Build()
		if err = db.UpdateDocument(conn, doc); err != nil {
			panic(errors.Wrap(err, "Could not update document"))
		}
		saveDocument(uuid, content)
		return next()
	}
}

func recoverPanic(w http.ResponseWriter) {
	if err := recover(); err != nil {
		errStr := fmt.Sprintf("%v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
	}
}

func replaceLineBreaks(text string) string {
	re := regexp.MustCompile(`\r?\n`)
	return re.ReplaceAllString(text, "\n")
}

func loadDocumentAsHtml(uuid string) string {
	content, err := ioutil.ReadFile("data/document/" + uuid + "/doc.md")
	if err != nil {
		panic(errors.Wrap(err, "Failed to load File: "+uuid))
	}

	// Init markdown parser (cannot be reused!)
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	parser := parser.NewWithExtensions(extensions)

	// Fix Windows Linebreaks
	content = []byte(replaceLineBreaks(string(content)))
	html := markdown.ToHTML(content, parser, nil)
	return string(html)
}

func loadDocument(uuid string) string {
	content, err := ioutil.ReadFile("data/document/" + uuid + "/doc.md")
	if err != nil {
		panic(errors.Wrap(err, "Failed to load File: "+uuid))
	}
	return string(content)
}

func saveDocument(uuid, content string) {
	var err error
	if err = ioutil.WriteFile("data/document/"+uuid+"/doc.md", []byte(content), 0644); err != nil {
		panic(errors.Wrap(err, "Failed to write File: "+uuid))
	}
}
