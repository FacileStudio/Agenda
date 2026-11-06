package main

import (
	"net/http"

	"github.com/FacileStudio/tronc/httpjson"
)

var version = "dev"

func versionHandler(w http.ResponseWriter, _ *http.Request) {
	httpjson.WriteJSON(w, http.StatusOK, map[string]string{"version": version})
}
