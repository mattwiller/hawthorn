package main

import (
	"fmt"

	"github.com/mattwiller/hawthorn/internal"
)

var setup = []string{
	`CREATE TABLE IF NOT EXISTS "CodeSystem" (
		id			INTEGER	PRIMARY KEY AUTOINCREMENT,
		_id			TEXT	NOT NULL,
		title		TEXT	NOT NULL,
		url			TEXT	NOT NULL,
		json		TEXT	NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS "ValueSet" (
		id			INTEGER	PRIMARY KEY AUTOINCREMENT,
		_id			TEXT	NOT NULL,
		url			TEXT	NOT NULL,
		json		TEXT	NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS "Coding" (
		id			INTEGER	PRIMARY KEY AUTOINCREMENT,
		system		INTEGER	NOT NULL, -- reference to "CodeSystem".id
		code		TEXT				NOT NULL,
		display		TEXT
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS "Coding_system_code_idx" ON "Coding" (system, code)`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS "Coding_fts_idx" USING fts5(display, tokenize = 'porter', content='Coding', content_rowid='id')`,
	// Triggers to keep the FTS index up to date.
	`CREATE TRIGGER "Coding_postinsert" AFTER INSERT ON "Coding" BEGIN
		INSERT INTO "Coding_fts_idx" (rowid, display) VALUES (new.id, new.display);
	END`,
	`CREATE TRIGGER "Coding_postdelete" AFTER DELETE ON "Coding" BEGIN
		INSERT INTO "Coding_fts_idx" ("Coding_fts_idx", rowid, display) VALUES ('delete', old.id, old.display);
	END`,
	`CREATE TRIGGER "Coding_postupdate" AFTER UPDATE ON "Coding" BEGIN
		INSERT INTO "Coding_fts_idx" ("Coding_fts_idx", rowid, display) VALUES ('delete', old.id, old.display);
		INSERT INTO "Coding_fts_idx" (rowid, display) VALUES (new.id, new.display);
	END`,

	`CREATE TABLE IF NOT EXISTS "CodeSystem_Property" (
		id			INTEGER	PRIMARY KEY AUTOINCREMENT,
		system		INTEGER	NOT NULL,
		code		TEXT	NOT NULL,
		type		TEXT	NOT NULL,
		uri			TEXT,
		description TEXT
	)`,

	`CREATE TABLE IF NOT EXISTS "Coding_Property" (
		coding		INTEGER	NOT NULL, -- reference to "Coding".id
		property	INTEGER	NOT NULL, -- reference to "CodeSystem_Property".id
		target		INTEGER, -- reference to "Coding".id, for relationship properties
		value		TEXT -- value could be string | integer | boolean | dateTime
	)`,
	`CREATE INDEX IF NOT EXISTS "Coding_Property_idx" ON "Coding_Property" (coding, property)`,
	`CREATE INDEX IF NOT EXISTS "Coding_Property_relationship_idx" ON "Coding_Property" (coding, target, property)
		WHERE target IS NOT NULL`,

	`CREATE TABLE IF NOT EXISTS "ValueSet_Membership" (
		"valueSet"	INTEGER, -- reference to "ValueSet".id
		coding		INTEGER, -- reference to "Coding".id
		PRIMARY KEY ("valueSet", coding)
	) WITHOUT ROWID`,
}

func main() {
	db, err := internal.NewDB("umls.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	fmt.Printf("Connected to database, running setup statements...\n")
	for _, stmt := range setup {
		_, err := db.Query(stmt)
		if err != nil {
			panic(fmt.Errorf(`error executing setup statement: %w`, err))
		}
		fmt.Print(".")
	}
	fmt.Println("✅")

	if err := internal.LoadUMLS(db); err != nil {
		panic(err)
	}
}
