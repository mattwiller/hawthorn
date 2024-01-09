package main

import (
	"fmt"
	"net/http"

	"github.com/mattwiller/hawthorn/internal"
	"github.com/mattwiller/hawthorn/internal/fhir"
)

func main() {
	db, err := internal.NewDB("umls.db")
	if err != nil {
		panic(fmt.Errorf("error opening database file: %w", err))
	}

	http.HandleFunc("/R4/CodeSystem/$lookup", fhir.CodeSystemLookupHandler(db))

	fmt.Println("Listening on :29927")
	if err := http.ListenAndServe(":29927", nil); err != nil {
		panic(fmt.Errorf("error starting HTTP server: %w", err))
	}
}
