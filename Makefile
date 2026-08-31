.PHONY: test test-api test-web run-api run-web build-api build-web

test: test-api test-web

test-api:
	cd api && go test ./...

test-web:
	cd web && npm ci && npm run check

run-api:
	cd api && go run ./cmd/server

run-web:
	cd web && npm install && npm run dev

build-api:
	cd api && CGO_ENABLED=0 go build -trimpath -o ../bin/kubevista-api ./cmd/server

build-web:
	cd web && npm ci && npm run build
