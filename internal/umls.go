package internal

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/xenking/zipstream"
)

//go:embed resources/CodeSystem/snomed.json
var snomed []byte

//go:embed resources/CodeSystem/icd10pcs.json
var icd10pcs []byte

//go:embed resources/CodeSystem/icd10cm.json
var icd10cm []byte

//go:embed resources/CodeSystem/loinc.json
var loinc []byte

//go:embed resources/CodeSystem/rxnorm.json
var rxnorm []byte

//go:embed resources/CodeSystem/cpt.json
var cpt []byte

//go:embed resources/CodeSystem/cvx.json
var cvx []byte

type CodeSystem struct {
	ResourceType     string               `json:"resourceType"`
	Url              string               `json:"url"`
	Title            string               `json:"title"`
	HierarchyMeaning string               `json:"hierarchyMeaning"`
	Property         []CodeSystemProperty `json:"property"`

	// ----- Private fields -----
	dbID int64
}

type CodeSystemProperty struct {
	Code        string `json:"code"`
	Uri         string `json:"uri"`
	Description string `json:"description"`
	Type        string `json:"type"`

	// ----- Private fields -----
	dbID int64
}

func ParseCodeSystem(bytes []byte) *CodeSystem {
	var system CodeSystem
	err := json.Unmarshal(bytes, &system)
	if err != nil {
		panic(err)
	} else if system.ResourceType != "CodeSystem" {
		panic("Invalid resource type: " + system.ResourceType)
	}
	return &system
}

func (system *CodeSystem) GetProperty(name string) *CodeSystemProperty {
	for _, p := range system.Property {
		if p.Code == name || p.Code == mappedProperties[name] {
			return &p
		}
	}
	return nil
}

type umlsSource struct {
	systemID uuid.UUID
	tty      []string
	json     []byte
	resource *CodeSystem
}

var umlsSources = map[string]umlsSource{
	"SNOMEDCT_US": {
		systemID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("http://snomed.info/sct")),
		tty:      []string{"FN", "PT", "SY"},
		json:     snomed,
		resource: ParseCodeSystem(snomed),
	},
	"ICD10PCS": {
		systemID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("http://hl7.org/fhir/sid/icd-10-pcs")),
		tty:      []string{"PT", "HT"},
		json:     icd10pcs,
		resource: ParseCodeSystem(icd10pcs),
	},
	"ICD10CM": {
		systemID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("http://hl7.org/fhir/sid/icd-10-cm")),
		tty:      []string{"PT", "HT"},
		json:     icd10cm,
		resource: ParseCodeSystem(icd10cm),
	},
	"LNC": {
		systemID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("http://loinc.org")),
		tty:      []string{"LC", "LPDN", "LA", "DN", "HC", "LN", "LG"},
		json:     loinc,
		resource: ParseCodeSystem(loinc),
	},
	"CPT": {
		systemID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("http://www.ama-assn.org/go/cpt")),
		tty:      []string{"PT", "HT", "POS", "MP", "GLP"},
		json:     cpt,
		resource: ParseCodeSystem(cpt),
	},
	"RXNORM": {
		systemID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("http://www.nlm.nih.gov/research/umls/rxnorm")),
		tty:      []string{"PSN", "MIN", "SBD", "SCD", "SBDG", "SCDG", "GPCK", "SY"},
		json:     rxnorm,
		resource: ParseCodeSystem(rxnorm),
	},
	"CVX": {
		systemID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("http://hl7.org/fhir/sid/cvx")),
		tty:      []string{"PT"},
		json:     cvx,
		resource: ParseCodeSystem(cvx),
	},
}

func LoadUMLS(db *DB) error {
	fmt.Println("Loading UMLS...")
	if err := LoadCodeSystems(db); err != nil {
		return fmt.Errorf("error loading CodeSystems: %w", err)
	}

	archive, err := os.Open("umls-2023AB-full.zip")
	if err != nil {
		return fmt.Errorf("error opening UMLS data archive: %w", err)
	}
	unzip := zipstream.NewReader(archive)
	var concepts map[string]*Concept
	var relationshipProperties map[string]string
	completed := 0
	for {
		file, err := unzip.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println(err.Error())
			buf := make([]byte, 1)
			_, err := unzip.Read(buf)
			fmt.Println(err.Error())
			// return fmt.Errorf("error reading zip file: %w", err)
			continue
		}

		if strings.HasSuffix(file.Name, "/MRCONSO.RRF") {
			if concepts, err = LoadConcepts(db, unzip); err != nil {
				return fmt.Errorf("error loading concepts: %w", err)
			}
			completed++
		} else if strings.HasSuffix(file.Name, "/MRDOC.RRF") {
			relationshipProperties = MapProperties(unzip)
			completed++
		} else if strings.HasSuffix(file.Name, "/MRSAT.RRF") {
			if concepts == nil {
				return errors.New("expected to read concepts before properties (MRCONSO.RRF before MRSAT.RRF)")
			}

			if err := LoadProperties(db, concepts, unzip); err != nil {
				return fmt.Errorf("error loading properties: %w", err)
			}
			completed++
		} else if strings.HasSuffix(file.Name, "/MRREL.RRF") {
			if concepts == nil {
				return errors.New("expected to read concepts before relationship properties (MRCONSO.RRF before MRREL.RRF)")
			} else if relationshipProperties == nil {
				return errors.New("expected to read relationship mapping before relationship properties (MRDOC.RRF before MRREL.RRF)")
			}

			if err := LoadRelationships(db, concepts, relationshipProperties, unzip); err != nil {
				return fmt.Errorf("error loading relationships: %w", err)
			}
			completed++
		} else {
			fmt.Printf("skipping %s", file.Name)
			if !file.FileInfo().IsDir() && file.UncompressedSize64 > 0 {
				fmt.Print("..")
				buf := make([]byte, 2^20)
				var err error
				for _, err = unzip.Read(buf); err == nil; _, err = unzip.Read(buf) {
					continue
				}
				if err != io.EOF {
					return fmt.Errorf("error reading skipped file %s: %w", file.Name, err)
				}
				fmt.Print("done\n")
			} else {
				fmt.Println()
			}
			continue
		}

		if completed >= 4 {
			break
		}
	}
	return nil
}

func LoadCodeSystems(db *DB) error {
	fmt.Println("Loading code system definitions:")
	for key, source := range umlsSources {
		results, err := db.Query(
			`INSERT INTO "CodeSystem" (_id, title, url, json) VALUES ($1, $2, $3, $4) RETURNING id`,
			source.systemID, source.resource.Title, source.resource.Url, source.json,
		)
		if err != nil {
			fmt.Printf("%s ❌\n", key)
			return err
		}

		fmt.Printf("%s ✅\n", key)
		source.resource.dbID = results[0]["id"].(int64)
	}

	fmt.Println()
	return nil
}

type Concept struct {
	// Unique identifier for concept.
	CUI string `json:"conceptID"`
	// Language of term.
	LAT string `json:"language"`
	// Term status.
	TS string `json:"termStatus"`
	// Unique identifier for term.
	LUI string `json:"lexicalID"`
	// String type.
	STT string `json:"stringType"`
	// Unique identifier for string.
	SUI string `json:"stringID"`
	// Atom status - preferred (Y) or not (N) for this string within this concept.
	ISPREF bool `json:"isPreferred"`
	// Unique identifier for atom - variable length field, 8 or 9 characters.
	AUI string `json:"atomID"`
	// Source asserted atom identifier [optional].
	SAUI *string `json:"sourceAtomID"`
	// Source asserted concept identifier [optional].
	SCUI *string `json:"sourceconceptID"`
	// Source asserted descriptor identifier [optional].
	SDUI *string `json:"sourceDescriptorID"`
	// Abbreviated source name. Maximum field length is 20 alphanumeric characters.
	// Official source names, RSABs, and VSABs are included on the
	// [UMLS Source Vocabulary Documentation page](https://www.nlm.nih.gov/research/umls/sourcereleasedocs/index.html).
	SAB string `json:"sourceNameAbbr"`
	// Abbreviation for term type in source vocabulary, for example PN (Metathesaurus Preferred Name) or CD (Clinical Drug).
	// Possible values are listed on the
	// [Abbreviations Used in Data Elements page](http://www.nlm.nih.gov/research/umls/knowledge_sources/metathesaurus/release/abbreviations.html).
	TTY string `json:"termType"`
	// Most useful source asserted identifier, or a generated source entry identifier (if the source has none).
	CODE string `json:"code"`
	// String value.
	STR string `json:"string"`
	// Source restriction level.
	SRL int `json:"sourceRestrictionLevel"`
	// Suppressible flag. Values = O, E, Y, or N.
	// O: All obsolete content, whether they are obsolesced by the source or by NLM.
	// These will include all atoms having obsolete TTYs, and other atoms becoming obsolete that have not acquired an obsolete TTY
	// (e.g. RxNorm SCDs no longer associated with current drugs, LNC atoms derived from obsolete LNC concepts).
	// E: Non-obsolete content marked suppressible by an editor. These do not have a suppressible SAB/TTY combination.
	// Y: Non-obsolete content deemed suppressible during inversion.
	// These can be determined by a specific SAB/TTY combination explicitly listed in MRRANK.
	// N: None of the above
	SUPPRESS string `json:"suppressible"`
	// Content View Flag. Bit field used to flag rows included in Content View.
	// This field is a varchar field to maximize the number of bits available for use.
	CVF uint64 `json:"contentViewFlag"`

	// ----- Private fields -----

	dbID int64
}

var pipeDelimiter = []byte{'|'}

func ParseConcept(row []byte) Concept {
	concept := Concept{}
	fields := bytes.Split(row, pipeDelimiter)
	for n, value := range fields {
		switch n {
		case 0:
			concept.CUI = string(value)
		case 1:
			concept.LAT = string(value)
		case 2:
			concept.TS = string(value)
		case 3:
			concept.LUI = string(value)
		case 4:
			concept.STT = string(value)
		case 5:
			concept.SUI = string(value)
		case 6:
			if len(value) == 1 && value[0] == 'Y' {
				concept.ISPREF = true
			}
		case 7:
			concept.AUI = string(value)
		case 8:
			if len(value) > 0 {
				saui := string(value)
				concept.SAUI = &saui
			}
		case 9:
			if len(value) > 0 {
				scui := string(value)
				concept.SCUI = &scui
			}
		case 10:
			if len(value) > 0 {
				sdui := string(value)
				concept.SDUI = &sdui
			}
		case 11:
			concept.SAB = string(value)
		case 12:
			concept.TTY = string(value)
		case 13:
			concept.CODE = string(value)
		case 14:
			concept.STR = string(value)
		case 15:
			if len(value) > 0 {
				n, _ := strconv.ParseInt(string(value), 10, 0)
				concept.SRL = int(n)
			}
		case 16:
			if len(value) > 0 {
				concept.SUPPRESS = string(value[0])
			}
		case 17:
			if len(value) > 0 {
				n, _ := strconv.ParseUint(string(value), 10, 64)
				concept.CVF = n
			}
		default:
			// Done, ignore leftover fields
			return concept
		}
	}
	return concept
}

func LoadConcepts(db *DB, file io.Reader) (map[string]*Concept, error) {
    scan := bufio.NewScanner(file)
    n := 0
    codings := make(map[string]int, 8)

    fmt.Println("Loading concepts:")
    var concepts = make(map[string]*Concept, 2^20)

    // Obtain a connection from the pool at the start of the function
    conn, err := db.GetConnection()
    if err != nil {
        return nil, fmt.Errorf("failed to get database connection: %w", err)
    }
    defer db.PutConnection(conn) // Ensure to return the connection back to the pool at the end

    // Start the transaction
    if err := db.Batch(conn); err != nil {
        return nil, fmt.Errorf("failed to start batch operation: %w", err)
    }

    for scan.Scan() {
        line := scan.Bytes()
        concept := ParseConcept(line)
        source, ok := umlsSources[concept.SAB]
        if !ok || concept.LAT != "ENG" || concept.SUPPRESS != "N" {
            continue
        } else if !slices.Contains(source.tty, concept.TTY) {
            continue
        }

        key := concept.SAB + "|" + concept.CODE
        if ex, exists := concepts[key]; exists {
            if slices.Index(source.tty, concept.TTY) >= slices.Index(source.tty, ex.TTY) {
                continue
            }
        } else {
            codings[concept.SAB]++
        }

        // Ensure to use the obtained connection for the query
        results, err := db.QueryWithConnection(conn, `INSERT INTO "Coding" (system, code, display) VALUES ($1, $2, $3) ON CONFLICT (system, code) DO UPDATE SET display = EXCLUDED.display RETURNING id`, source.resource.dbID, concept.CODE, concept.STR)
        if err != nil {
            return nil, fmt.Errorf("failed to insert coding: %w", err)
        }

        concept.dbID = results[0]["id"].(int64)
        concepts[concept.AUI] = &concept
        concepts[key] = &concept

        n++
        if n%500 == 0 {
            // Flush and start a new batch using the same connection
            if err := db.Flush(conn); err != nil {
                return nil, fmt.Errorf("failed to flush batch operation: %w", err)
            }
            if err := db.Batch(conn); err != nil {
                return nil, fmt.Errorf("failed to start new batch operation: %w", err)
            }
            fmt.Print(".")
        }
    }

    // Final flush to commit any remaining operations
    if err := db.Flush(conn); err != nil {
        return nil, fmt.Errorf("failed to finalize batch operation: %w", err)
    }

    fmt.Println("✅")
    fmt.Printf("Processed %d rows\n======================\n", n)
    total := 0
    for system, codes := range codings {
        total += codes
        fmt.Printf("%s: %d\n", system, codes)
    }
    fmt.Printf("======================\n(total %d unique concepts)\n\n", total)
    return concepts, nil
}

func MapProperties(file io.Reader) map[string]string {
	scan := bufio.NewScanner(file)

	mappings := make(map[string]struct {
		rel  string
		rela string
	}, 32)
	for scan.Scan() {
		line := scan.Bytes()
		parts := bytes.Split(line, pipeDelimiter)
		dockey, value, explType, expl := parts[0], string(parts[1]), parts[2], string(parts[3])
		if bytes.Equal(dockey, []byte("REL")) && bytes.Equal(explType, []byte("snomedct_rel_mapping")) {
			m := mappings[value]
			m.rel = expl
			mappings[value] = m
		} else if bytes.Equal(dockey, []byte("RELA")) && bytes.Equal(explType, []byte("snomedct_rela_mapping")) {
			m := mappings[value]
			m.rela = expl
			mappings[value] = m
		}
	}

	output := make(map[string]string, len(mappings))
	for property, mapping := range mappings {
		output["SNOMEDCT_US/"+mapping.rel+"/"+mapping.rela] = property
	}
	return output
}

// Represents a UMLS attribute (coding property).
// @see https://www.ncbi.nlm.nih.gov/books/NBK9685/table/ch03.T.simple_concept_and_atom_attribute
type Attribute struct {
	// Unique identifier for concept (if METAUI is a relationship identifier, this will be CUI1 for that relationship).
	CUI string
	// Unique identifier for term (optional - present for atom attributes, but not for relationship attributes).
	LUI string
	// Unique identifier for string (optional - present for atom attributes, but not for relationship attributes).
	SUI string
	// Metathesaurus atom identifier (will have a leading A) or Metathesaurus relationship identifier (will have a leading R) or blank if it is a concept attribute.
	METAUI string
	// The name of the column in MRCONSO.RRF or MRREL.RRF that contains the identifier to which the attribute is attached, i.e. AUI, CODE, CUI, RUI, SCUI, SDUI.
	STYPE string
	// Most useful source asserted identifier (if the source vocabulary contains more than one) or a Metathesaurus-generated source entry identifier (if the source vocabulary has none). Optional - present if METAUI is an AUI.
	CODE string
	// Unique identifier for attribute.
	ATUI string
	// Source asserted attribute identifier (optional - present if it exists).
	SATUI string
	// Attribute name.
	// @see http://www.nlm.nih.gov/research/umls/knowledge_sources/metathesaurus/release/attribute_names.html
	ATN string
	// Source abbreviation.  This uniquely identifies the underlying source vocabulary.
	// @see https://www.nlm.nih.gov/research/umls/sourcereleasedocs/index.html
	SAB string
	// Attribute value described under specific attribute name on the Attributes Names page.
	// @see http://www.nlm.nih.gov/research/umls/knowledge_sources/metathesaurus/release/abbreviations.html
	ATV string
	// Suppressible flag.
	//
	// O = All obsolete content, whether they are obsolesced by the source or by NLM
	// E = Non-obsolete content marked suppressible by an editor
	// Y = Non-obsolete content deemed suppressible during inversion
	// N = None of the above (not suppressible)
	SUPPRESS string
}

func ParseAttribute(row []byte) Attribute {
	attribute := Attribute{}
	fields := bytes.Split(row, pipeDelimiter)
	for n, value := range fields {
		switch n {
		case 0:
			attribute.CUI = string(value)
		case 1:
			attribute.LUI = string(value)
		case 2:
			attribute.SUI = string(value)
		case 3:
			attribute.METAUI = string(value)
		case 4:
			attribute.STYPE = string(value)
		case 5:
			attribute.CODE = string(value)
		case 6:
			attribute.ATUI = string(value)
		case 7:
			attribute.SATUI = string(value)
		case 8:
			attribute.ATN = string(value)
		case 9:
			attribute.SAB = string(value)
		case 10:
			attribute.ATV = string(value)
		case 11:
			attribute.SUPPRESS = string(value)
		default:
			// Done, ignore leftover fields
			return attribute
		}
	}
	return attribute
}

var mappedProperties = map[string]string{
	"LOINC_COMPONENT":   "COMPONENT",
	"LOINC_METHOD_TYP":  "METHOD_TYP",
	"LOINC_PROPERTY":    "PROPERTY",
	"LOINC_SCALE_TYP":   "SCALE_TYP",
	"LOINC_SYSTEM":      "SYSTEM",
	"LOINC_TIME_ASPECT": "TIME_ASPCT",
	"LOR":               "ORDER_OBS",
	"LQS":               "SURVEY_QUEST_SRC",
	"LQT":               "SURVEY_QUEST_TEXT",
	"LRN2":              "RELATEDNAMES2",
	"LCL":               "CLASS",
	"LCN":               "CLASSTYPE",
	"LCS":               "STATUS",
	"LCT":               "CHNG_TYPE",
	"LEA":               "EXMPL_ANSWERS",
	"LFO":               "FORMULA",
	"LMP":               "MAP_TO",
	"LUR":               "UNITSREQUIRED",
	"LC":                "LONG_COMMON_NAME",
}

func LoadProperties(db *DB, concepts map[string]*Concept, file io.Reader) error {
    scan := bufio.NewScanner(file)
    n := 0
    propertyCounts := make(map[string]int, 64)

    fmt.Println("Loading properties:")
    
    conn, err := db.GetConnection()
    if err != nil {
        return fmt.Errorf("failed to get database connection: %w", err)
    }
    defer db.PutConnection(conn) 

	if err := db.Batch(conn); err != nil {
        return fmt.Errorf("failed to start batch operation: %w", err)
    }

    for scan.Scan() {
        line := scan.Bytes()
        attribute := ParseAttribute(line)
        source, ok := umlsSources[attribute.SAB]
        if !ok || attribute.SUPPRESS != "N" {
            continue
        }

        property := source.resource.GetProperty(attribute.ATN)
        if property == nil {
            continue
        }
        if property.dbID == 0 {
            results, err := db.QueryWithConnection(conn, `INSERT INTO "CodeSystem_Property" (system, code, type, uri, description) VALUES ($1, $2, $3, $4, $5) RETURNING id`, source.resource.dbID, property.Code, property.Type, property.Uri, property.Description)
            if err != nil {
                return fmt.Errorf("failed to insert code system property: %w", err)
            }
            property.dbID = results[0]["id"].(int64)
        }

        concept := concepts[attribute.SAB+"|"+attribute.CODE]
        if concept == nil {
            return fmt.Errorf("unknown code: %s|%s", attribute.SAB, attribute.CODE)
        }

        _, err = db.QueryWithConnection(conn, `INSERT INTO "Coding_Property" (coding, property, value) VALUES ($1, $2, $3)`, concept.dbID, property.dbID, attribute.ATV)
        if err != nil {
            return fmt.Errorf("failed to insert coding property: %w", err)
        }

        propertyCounts[attribute.SAB+"|"+attribute.ATN]++
        n++

        if n%500 == 0 {
            if err := db.Flush(conn); err != nil {
                return fmt.Errorf("failed to flush batch operation: %w", err)
            }
            if err := db.Batch(conn); err != nil {
                return fmt.Errorf("failed to start new batch operation: %w", err)
            }
            fmt.Print(".")
        }
    }

    if err := db.Flush(conn); err != nil {
        return fmt.Errorf("failed to finalize batch operation: %w", err)
    }

    fmt.Println("✅")
    for property, count := range propertyCounts {
        fmt.Printf("%s: %d\n", property, count)
    }
    fmt.Printf("======================\n(total %d properties)\n\n", n)
    return nil
}

// Represents a relationship between two UMLS Concepts.
// @see https://www.ncbi.nlm.nih.gov/books/NBK9685/table/ch03.T.related_concepts_file_mrrel_rrf
type Relationship struct {
	// Unique identifier of first concept.
	CUI1 string
	// Unique identifier of first atom.
	AUI1 string
	// The name of the column in MRCONSO.RRF that contains the identifier used for the first element in the relationship, i.e. AUI, CODE, CUI, SCUI, SDUI.
	STYPE1 string
	// Relationship of second concept or atom to first concept or atom.
	REL string
	// Unique identifier of second concept.
	CUI2 string
	// Unique identifier of second atom.
	AUI2 string
	// The name of the column in MRCONSO.RRF that contains the identifier used for the second element in the relationship, i.e. AUI, CODE, CUI, SCUI, SDUI.
	STYPE2 string
	// Additional (more specific) relationship label (optional).
	RELA string
	// Unique identifier of relationship.
	RUI string
	// Source asserted relationship identifier, if present.
	SRUI string
	// Source abbreviation.  This uniquely identifies the underlying source vocabulary.
	// @see https://www.nlm.nih.gov/research/umls/sourcereleasedocs/index.html
	SAB string
	// Source of relationship labels.
	SL string
	// Relationship group. Used to indicate that a set of relationships should be looked at in conjunction.
	RG string
	// Source asserted directionality flag.
	// 'Y' indicates that this is the direction of the relationship in its source; 'N' indicates that it is not;
	// a blank indicates that it is not important or has not yet been determined.
	DIR string
	// Suppressible flag.
	// O = All obsolete content, whether they are obsolesced by the source or by NLM
	// E = Non-obsolete content marked suppressible by an editor
	// Y = Non-obsolete content deemed suppressible during inversion
	// N = None of the above (not suppressible)
	SUPPRESS string
}

func ParseRelationship(row []byte) Relationship {
	relationship := Relationship{}
	fields := bytes.Split(row, pipeDelimiter)
	for n, value := range fields {
		switch n {
		case 0:
			relationship.CUI1 = string(value)
		case 1:
			relationship.AUI1 = string(value)
		case 2:
			relationship.STYPE1 = string(value)
		case 3:
			relationship.REL = string(value)
		case 4:
			relationship.CUI2 = string(value)
		case 5:
			relationship.AUI2 = string(value)
		case 6:
			relationship.STYPE2 = string(value)
		case 7:
			relationship.RELA = string(value)
		case 8:
			relationship.RUI = string(value)
		case 9:
			relationship.SRUI = string(value)
		case 10:
			relationship.SAB = string(value)
		case 11:
			relationship.SL = string(value)
		case 12:
			relationship.RG = string(value)
		case 13:
			relationship.DIR = string(value)
		case 14:
			relationship.SUPPRESS = string(value)
		default:
			// Done, ignore leftover fields
			return relationship
		}
	}
	return relationship
}

const PARENT_URI = "http://hl7.org/fhir/concept-properties#parent"
const CHILD_URI = "http://hl7.org/fhir/concept-properties#child"

func LoadRelationships(db *DB, concepts map[string]*Concept, relationshipProperties map[string]string, file io.Reader) error {
    scan := bufio.NewScanner(file)
    n := 0
    propertyCounts := make(map[string]int, 64)

    fmt.Println("Loading relationships:")
    
    conn, err := db.GetConnection()
    if err != nil {
        return fmt.Errorf("failed to get database connection: %w", err)
    }
    defer db.PutConnection(conn)

    if err := db.Batch(conn); err != nil {
        return fmt.Errorf("failed to start batch operation: %w", err)
    }

    for scan.Scan() {
        line := scan.Bytes()
        relationship := ParseRelationship(line)
        source, ok := umlsSources[relationship.SAB]
        if !ok || relationship.SUPPRESS != "N" {
            continue
        }

        mappedRelationshipProperty := relationshipProperties[relationship.SAB+"/"+relationship.REL+"/"+relationship.RELA]
        var propertyName string
        var property CodeSystemProperty
        if mappedRelationshipProperty != "" {
            propertyName = mappedRelationshipProperty
            for _, p := range source.resource.Property {
                if p.Code == propertyName {
                    property = p
                    break
                }
            }
        } else if relationship.REL == "PAR" {
            for _, p := range source.resource.Property {
                if p.Uri == PARENT_URI {
                    propertyName = p.Code
                    property = p
                    break
                }
            }
        } else if relationship.REL == "CHD" {
            for _, p := range source.resource.Property {
                if p.Uri == CHILD_URI {
                    propertyName = p.Code
                    property = p
                    break
                }
            }
        }
        if propertyName == "" {
            continue
        }
        if property.dbID == 0 {
            results, err := db.QueryWithConnection(conn, `INSERT INTO "CodeSystem_Property" (system, code, type, uri, description) VALUES ($1, $2, $3, $4, $5) RETURNING id`, source.resource.dbID, property.Code, property.Type, property.Uri, property.Description)
            if err != nil {
                return fmt.Errorf("failed to insert code system property: %w", err)
            }
            property.dbID = results[0]["id"].(int64)
        }

        srcConcept := concepts[relationship.AUI1]
        dstConcept := concepts[relationship.AUI2]
        if srcConcept == nil || dstConcept == nil {
            continue
        }

        key := fmt.Sprintf(`%s|%s (%s/%s)`, source.resource.Url, propertyName, relationship.REL, relationship.RELA)

        _, err = db.QueryWithConnection(conn, `INSERT INTO "Coding_Property" (coding, property, target, value) VALUES ($1, $2, $3, $4)`, srcConcept.dbID, property.dbID, dstConcept.dbID, dstConcept.CODE)
        if err != nil {
            return fmt.Errorf("failed to insert coding property: %w", err)
        }

        propertyCounts[key]++
        n++

        if n%500 == 0 {
            if err := db.Flush(conn); err != nil {
                return fmt.Errorf("failed to flush batch operation: %w", err)
            }
            if err := db.Batch(conn); err != nil {
                return fmt.Errorf("failed to start new batch operation: %w", err)
            }
            fmt.Print(".")
        }
    }

    if err := db.Flush(conn); err != nil {
        return fmt.Errorf("failed to finalize batch operation: %w", err)
    }

    fmt.Println("✅")
    for property, count := range propertyCounts {
        fmt.Printf("%s: %d\n", property, count)
    }
    fmt.Printf("======================\n(total %d relationships)\n\n", n)
    return nil
}
