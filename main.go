package main

import (
	"net/http"

	"github.com/markoczy/docserver/db"
	"github.com/markoczy/docserver/server"
	"github.com/markoczy/webapi"
)

func main() {
	// Init DB
	var err error
	conn := db.MustConnect("data/store/store.db")
	if err = db.ProcessUpdates(conn); err != nil {
		panic(err)
	}

	// Create Router
	fallback404 := webapi.NewErrorHandler(http.StatusNotFound, "XPage not found")
	router := webapi.NewRouter(fallback404)
	server.InitViewController(router, conn)
	server.InitAssetController(router)
	http.ListenAndServe(":7890", router)
}
