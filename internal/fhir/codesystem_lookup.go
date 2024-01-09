package fhir

import (
	"net/http"

	"github.com/mattwiller/hawthorn/internal"
)

// Implements the CodeSystem/$lookup operation endpoint.
// @see http://hl7.org/fhir/R4B/codesystem-operation-lookup.html
func CodeSystemLookupHandler(db *internal.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if !query.Has("system") || !query.Has("code") {
			sendError(w, "required", "Coding must be specified using 'system' and 'code' query parameters")
			return
		}

		system := query.Get("system")
		code := query.Get("code")

		results, err := db.Query(`SELECT id,title FROM "CodeSystem" WHERE url = $1`, system)
		if err != nil {
			sendError(w, "not-found", "Code system not found")
			return
		}
		codeSystem := results[0]
		systemID := codeSystem["id"].(int64)

		results, err = db.Query(`SELECT id,display FROM "Coding" WHERE "Coding".system = $1 AND "Coding".code = $2;`, systemID, code)
		if err != nil {
			return
		}
		codingID := results[0]["id"].(int64)
		display := results[0]["display"].(string)

		results, err = db.Query(`SELECT "Prop".*, "Code_Prop".value, "Code_Prop".target FROM "Coding_Property" "Code_Prop" JOIN "Coding" ON "Code_Prop".coding = "Coding".id
			JOIN "CodeSystem_Property" "Prop" ON "Prop".id = "Code_Prop".property WHERE "Coding".id = $1`, codingID)
		if err != nil {
			return
		}

		output := []map[string]any{
			{"name": "name", "valueString": codeSystem["title"].(string)},
			{"name": "display", "valueString": display},
		}
		for _, property := range results {
			propType := capitalize(property["type"].(string))
			if propType == "Coding" {
				property["value"] = map[string]any{
					"code": property["value"],
				}
			}
			output = append(output, map[string]any{"name": "property", "part": []map[string]any{
				{"name": "code", "valueCode": property["code"]},
				{"name": "description", "valueString": property["description"]},
				{"name": "value", "value" + capitalize(property["type"].(string)): property["value"]},
			}})
		}

		sendOutput(w, output)
	}
}
