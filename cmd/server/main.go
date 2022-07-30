package main

import (
	"fmt"
	"net/http"
)

func HelloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method, r.RequestURI)
}

func main() {
	http.HandleFunc("/update/", HelloWorld)

	fmt.Println("Server Started: http://localhost:8080/")
	http.ListenAndServe(":8080", nil)
}
