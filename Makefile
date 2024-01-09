.PHONY: build clean run test

# ===== Component files =====

umls.db: umls-2023AB-full.zip
	go run cmd/build.go

umls-2023AB-full.zip:
	curl "https://uts-ws.nlm.nih.gov/download?url=https://download.nlm.nih.gov/umls/kss/2023AB/umls-2023AB-metathesaurus-full.zip&apiKey=$(UMLS_API_KEY)" -o umls-2023AB-full.zip

# ===== Commands =====

build: umls.db
	go build

clean:
	rm -f umls.db*

run: umls.db
	go run main.go

test: umls.db
	go test -bench=. -benchmem ./...