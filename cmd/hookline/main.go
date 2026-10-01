package main

import (
	"log"
	"net/http"

	"github.com/ngaunpot/hookline/internal/api"
)

func main() {
	srv := api.NewServer()

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", srv))
}
