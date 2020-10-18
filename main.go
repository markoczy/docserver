package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"text/template"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/markoczy/goutil/log"
	"github.com/markoczy/webapi"
)

type AppState struct {
	Header   *HeaderState
	Body     interface{}
	Template *template.Template
}

type HeaderState struct {
	ActivePage string
}

type MarkdownFile struct {
	Filename string
	Content  string
}

func recoverPanic(w http.ResponseWriter) {
	if err := recover(); err != nil {
		errStr := fmt.Sprintf("%v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
	}
}

func removeLBR(text string) string {
	re := regexp.MustCompile(`\x{000D}\x{000A}|[\x{000A}\x{000B}\x{000C}\x{000D}\x{0085}\x{2028}\x{2029}]`)
	return re.ReplaceAllString(text, "\n")
}

func handleError(err error) webapi.Handler {
	return webapi.NewErrorHandler(http.StatusInternalServerError, err.Error())
}

func handleBadRequest(err string) webapi.Handler {
	return webapi.NewErrorHandler(http.StatusBadRequest, err)
}

func handleLogRequest(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
	log.Debugf("Request: %s %s", r.Request.Method, r.Request.RequestURI)
	return next()
}

// could be a way, currently unused
func handleInitTemplate(activePage string) func(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
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

// func handleHeaderTemplate(activePage string) func(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
// 	return func(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
// 		defer recoverPanic(w)
// 		state := r.State.(AppState)
// 		if state.Template == nil {
// 			panic(fmt.Errorf("Main Template is missing"))
// 		}

// 		return next()
// 	}
// }

func LoadDocument(document string) MarkdownFile {
	ret := MarkdownFile{}
	content, err := ioutil.ReadFile("data/document/" + document + "/doc.md")
	if err != nil {
		panic(err)
		// return ret, err
	}

	// Fix Windows Linebreaks
	content = []byte(removeLBR(string(content)))

	// init parser
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	parser := parser.NewWithExtensions(extensions)
	html := markdown.ToHTML(content, parser, nil)

	ret.Content = string(html)
	// _, file := filepath.Split(path)
	ret.Filename = document
	return ret
}

func handleViewDocument(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
	defer recoverPanic(w)
	filename := r.PathParams["file"]
	mdData := LoadDocument(filename)
	state := r.State.(*AppState)
	state.Body = mdData

	tmpl := template.Must(state.Template.ParseFiles(
		"data/template/viewer.html",
	))
	tmpl.Execute(w, state)
	return next()
}

func testPanic(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) (ret webapi.Handler) {
	defer recoverPanic(w)
	i := 1
	if i == 1 {
		log.Debug("Panic reached")
		panic("bad stuff")
	}
	log.Debug("End testPanic")
	return next()
}

func main() {
	fallback404 := webapi.NewErrorHandler(http.StatusNotFound, "Page not found")
	// Create Router
	router := webapi.NewRouter(fallback404)
	// Create Handler by chaining Handler functions
	handleView := webapi.NewHandler(
		handleLogRequest,
		handleInitTemplate("view"),
		handleViewDocument,
	)

	assetServer := http.FileServer(http.Dir("./data/asset"))
	handleAsset := webapi.NewHandler(
		handleLogRequest,
		func(w http.ResponseWriter, r *webapi.ParsedRequest, next func() webapi.Handler) webapi.Handler {
			assetServer.ServeHTTP(w, r.Request)
			return next()
		},
	)

	router.Handle(http.MethodGet, "/view", handleView)
	router.Handle(http.MethodGet, "/view/:file", handleView)
	router.Handle(http.MethodGet, "/testPanic", webapi.NewHandler(testPanic))
	router.Handle(http.MethodGet, "/(.*)", handleAsset)
	// Serve HTTP
	http.ListenAndServe(":7890", router)
}
