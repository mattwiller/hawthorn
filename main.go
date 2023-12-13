package main

import (
	"fmt"
	"net/http"

	"github.com/mattwiller/hawthorn/internal"
)

func main() {
	db, err := internal.NewDB("umls.db")
	if err != nil {
		panic(fmt.Errorf("error opening database file: %w", err))
	}

	http.HandleFunc("/R4/CodeSystem/$lookup", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if !query.Has("system") || !query.Has("code") {
			w.Write([]byte(`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"required","details":{"text":"Coding must be specified using 'system' and 'code' query parameters"}}]}`))
			return
		}

		system := query.Get("system")
		code := query.Get("code")

		results, err := db.Query(`SELECT id,title FROM "CodeSystem" WHERE url = $1`, system)
		if err != nil {
			fmt.Printf("Error: %s", err.Error())
			w.Write([]byte(`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"not-found","details":{"text":"Code system not found"}}]}`))
			return
		}
		codeSystem := results[0]
		systemID := codeSystem["id"].(int64)

		results, err = db.Query(`SELECT id,display FROM "Coding" WHERE "Coding".system = $1 AND "Coding".code = $2;`, systemID, code)
		if err != nil {
			fmt.Printf("Error: %s", err.Error())
			return
		}
		codingID := results[0]["id"].(int64)
		display := results[0]["display"].(string)

		results, err = db.Query(`SELECT "Prop".*, "Code_Prop".value FROM "Coding_Property" "Code_Prop" JOIN "Coding" ON "Code_Prop".coding = "Coding".id
			JOIN "CodeSystem_Property" "Prop" ON "Prop".id = "Code_Prop".property WHERE "Coding".id = $1`, codingID)
		if err != nil {
			fmt.Printf("Error: %s", err.Error())
			return
		}
		fmt.Printf("GOT RESULTS! %s|%s:\n%s (%d)\n\n", system, code, display, codingID)
		for _, property := range results {
			fmt.Printf("%s (%s): %s\n", property["code"], property["description"], property["value"])
		}
	})

	if err := http.ListenAndServe(":29927", nil); err != nil {
		panic(fmt.Errorf("error starting HTTP server: %w", err))
	}
}
