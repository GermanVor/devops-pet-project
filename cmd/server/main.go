package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/GermanVor/devops-pet-project/cmd/server/handlers"
	"github.com/GermanVor/devops-pet-project/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	currentStorage := storage.Init()
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	handlers.InitRouter(r, currentStorage)

	fmt.Println("Server Started: http://localhost:8080/")

	log.Fatal(http.ListenAndServe(":8080", r))
}
