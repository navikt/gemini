BUILDTIME = $(shell date "+%s")
DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git rev-parse --short HEAD)
LDFLAGS := -X github.com/nais/gemini/pkg/version.Revision=$(LAST_COMMIT) -X github.com/nais/gemini/pkg/version.Date=$(DATE) -X github.com/nais/gemini/pkg/version.BuildUnixTime=$(BUILDTIME)

.PHONY: alpine gemini test migration

gemini:
	go build -o bin/gemini -ldflags "-s $(LDFLAGS)" cmd/gemini/*.go

test:
	go test ./...

migration:
	go generate ./...

alpine:
	go build -a -installsuffix cgo -o bin/gemini -ldflags "-s $(LDFLAGS)" cmd/gemini/main.go

docker:
	docker build -t ghcr.io/nais/gemini:latest .
