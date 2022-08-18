package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/GermanVor/devops-pet-project/cmd/server/handlers"
	"github.com/GermanVor/devops-pet-project/common"
	"github.com/GermanVor/devops-pet-project/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/joho/godotenv"
)

var Address string

func init() {
	godotenv.Load(".env")
	Address = common.InitConfig().Address
}

func main() {
	currentStorage := storage.Init()
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	handlers.InitRouter(r, currentStorage)

	fmt.Println("Server Started: http://" + Address)

	log.Fatal(http.ListenAndServe(Address, r))
}
