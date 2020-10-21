package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"text/template"

	"github.com/bvinc/go-sqlite-lite/sqlite3"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/markoczy/docserver/db"
	"github.com/markoczy/docserver/domain/document"
	"github.com/markoczy/webapi"
	"github.com/pkg/errors"
)

type AppState struct {
	Header   *HeaderState
	Body     interface{}
	Template *template.Template
}

type HeaderState struct {
	ActivePage string
}

type DocumentData struct {
	Document document.Document
	Content  string
}

func InitViewController(router *webapi.Router, conn *sqlite3.Conn) error {
	handleView := webapi.NewHandler(
		handleLogRequest,
		handleInitTemplate("view"),
		handleViewDocument(conn),
	)
	router.Handle(http.MethodGet, "/view/:uuid", handleView)
	return nil
}

func InitAssetController(router *webapi.Router) {
	assetServer := http.FileServer(http.Dir("./data/asset"))
	handleAsset := webapi.NewHandler(
		handleLogRequest,
		func(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
			assetServer.ServeHTTP(w, r.Request)
			return next()
		},
	)
	router.Handle(http.MethodGet, "/(.*)", handleAsset)
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
		r.State = &AppState{
			Template: tmpl,
			Header: &HeaderState{
				ActivePage: activePage,
			},
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
			panic(err)
		}
		// Load content from file
		content := loadDocumentAsHtml(doc.Name())
		state := r.State.(*AppState)
		state.Body = DocumentData{
			Document: doc,
			Content:  content,
		}

		tmpl := template.Must(state.Template.ParseFiles(
			"data/template/viewer.html",
		))
		if err = tmpl.Execute(w, state); err != nil {
			panic(err)
		}
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

func loadDocumentAsHtml(name string) string {
	content, err := ioutil.ReadFile("data/document/" + name + "/doc.md")
	if err != nil {
		panic(errors.Wrap(err, "Failed to load File:"+name))
	}

	// Init markdown parser (cannot be reused!)
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	parser := parser.NewWithExtensions(extensions)

	// Fix Windows Linebreaks
	content = []byte(replaceLineBreaks(string(content)))
	html := markdown.ToHTML(content, parser, nil)
	return string(html)
}
