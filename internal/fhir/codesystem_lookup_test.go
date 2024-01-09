package fhir_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/mattwiller/hawthorn/internal"
	"github.com/mattwiller/hawthorn/internal/fhir"
	"github.com/stretchr/testify/require"
)

func TestCodeSystemLookup(t *testing.T) {
	require := require.New(t)

	db, err := internal.NewDB("../../umls.db")
	require.NoError(err)
	srv := fhir.CodeSystemLookupHandler(db)

	req := httptest.NewRequest("GET", "/R4/CodeSystem/$lookup?system=http://loinc.org&code=79741-5", nil)
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, req)

	require.Equal(200, res.Result().StatusCode)

	body, err := io.ReadAll(res.Result().Body)
	require.NoError(err)

	expected := `{
		"resourceType": "Parameters",
		"parameter": [
			{"name": "name", "valueString": "LOINC Code System"},
			{"name": "display", "valueString": "Eye-related brain MRI findings"},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "parent"},
				{"name": "description", "valueString": "A parent code in the Component Hierarchy by System"},
				{"name": "value", "valueCode": "MTHU000341"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "parent"},
				{"name": "description", "valueString": "A parent code in the Component Hierarchy by System"},
				{"name": "value", "valueCode": "LP408570-2"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "COMMON_TEST_RANK"},
				{"name": "description", "valueString": "Ranking of approximately 2000 common tests performed by laboratories in USA."},
				{"name": "value", "valueString": "0"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "COMMON_ORDER_RANK"},
				{"name": "description", "valueString": "Ranking of approximately 300 common orders performed by laboratories in USA."},
				{"name": "value", "valueString": "0"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "SYSTEM"},
				{"name": "description", "valueString": "Fourth major axis-type of specimen or system: System (Sample) Type"},
				{"name": "value", "valueString": "^Patient"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "CLASSTYPE"},
				{"name": "description", "valueString": "1=Laboratory class; 2=Clinical class; 3=Claims attachments; 4=Surveys"},
				{"name": "value", "valueString": "2"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "SCALE_TYP"},
				{"name": "description", "valueString": "Fifth major axis-scale of measurement: Type of Scale"},
				{"name": "value", "valueString": "Nom"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "ORDER_OBS"},
				{"name": "description", "valueString": "Provides users with an idea of the intended use of the term by categorizing it as an order only, observation only, or both"},
				{"name": "value", "valueString": "Observation"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "PROPERTY"},
				{"name": "description", "valueString": "Second major axis-property observed: Kind of Property (also called kind of quantity)"},
				{"name": "value", "valueString": "Find"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "CLASS"},
				{"name": "description", "valueString": "An arbitrary classification of terms for grouping related observations together"},
				{"name": "value", "valueString": "EYE.HX.NEI"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "TIME_ASPCT"},
				{"name": "description", "valueString": "Third major axis-timing of the measurement: Time Aspect (Point or moment in time vs. time interval)"},
				{"name": "value", "valueString": "Pt"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "COMPONENT"},
				{"name": "description", "valueString": "First major axis-component or analyte: Analyte Name, Analyte sub-class, Challenge"},
				{"name": "value", "valueString": "Eye-related brain MRI findings"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "STATUS"},
				{"name": "description", "valueString": "Status of the term. Within LOINC, codes with STATUS=DEPRECATED are considered inactive. Current values: ACTIVE, TRIAL, DISCOURAGED, and DEPRECATED"},
				{"name": "value", "valueString": "ACTIVE"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "CHNG_TYPE"},
				{"name": "description", "valueString": "DEL = delete (deprecate); ADD = add; PANEL = addition or removal of child elements or change in the conditionality of child elements in the panel or in sub-panels contained by the panel; NAM = change to Analyte/Component (field #2); MAJ = change to name field other than #2 (#3 - #7); MIN = change to field other than name; UND = undelete"},
				{"name": "value", "valueString": "MIN"}
			]},
			{"name": "property", "part": [
				{"name": "code", "valueCode": "RELATEDNAMES2"},
				{"name": "description", "valueString": "This field was introduced in version 2.05. It contains synonyms for each of the parts of the fully specified LOINC name (component, property, time, system, scale, method)."},
				{"name": "value", "valueString": "Eye; EYE.HX; EYE.HX.NEI; Eye-rel brain MRI find; Finding; Findings; Nominal; Ophthalmology; Ophtho; Ophthy; Point in time; Rad; Radiology; Random"}
			]}
		]
	}`
	require.JSONEq(expected, string(body))
}

func BenchmarkCodeSystemLookup(b *testing.B) {
	db, _ := internal.NewDB("../../umls.db")
	srv := fhir.CodeSystemLookupHandler(db)

	req := httptest.NewRequest("GET", "/R4/CodeSystem/$lookup?system=http://loinc.org&code=79741-5", nil)

	for i := 0; i < b.N; i++ {
		res := httptest.NewRecorder()
		srv.ServeHTTP(res, req)
	}
}
