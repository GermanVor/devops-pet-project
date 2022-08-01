package main

import (
	"fmt"
	"net/http"

	"github.com/GermanVor/devops-pet-project/cmd/server/handlers"
	storage "github.com/GermanVor/devops-pet-project/storage"
)

func main() {
	currentStorage := storage.Init()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.UpdateStorageFunc(w, r, currentStorage)
	})

	fmt.Println("Server Started: http://localhost:8080/")
	http.ListenAndServe(":8080", nil)
}
