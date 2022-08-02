package main

import (
	"devops-pet-project/cmd/server/handlers"
	"fmt"
	"log"
	"net/http"

	"devops-pet-project/storage"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func missedMetricNameHandlerFunc(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusNotFound)
	rw.Write(nil)
}

func main() {
	currentStorage := storage.Init()
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	handlers.InitRouter(r, currentStorage)

	fmt.Println("Server Started: http://localhost:8080/")

	log.Fatal(http.ListenAndServe(":8080", r))
}
