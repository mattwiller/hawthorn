package fhir

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func sendError(w http.ResponseWriter, code string, details string) {
	w.Write([]byte(fmt.Sprintf(`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"%s","details":{"text":"%s"}}]}`, code, details)))
}

func sendOutput(w http.ResponseWriter, parameters []map[string]any) {
	w.Write([]byte(formatParameters(parameters)))
}

func formatParameters(parameters []map[string]any) string {
	output, err := json.Marshal(parameters)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{"resourceType":"Parameters","parameter":%s}`, output)
}

func capitalize(s string) string {
	if len(s) < 1 {
		return ""
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}
