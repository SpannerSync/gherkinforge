.PHONY: test lint-go lint-features scaffold-demo ci build tidy

## build — compile the gforge CLI binary
build:
	go build -o bin/gforge ./cmd/gforge

## tidy — sync go.mod and go.sum
tidy:
	go mod tidy

## test — run all unit and BDD tests
test:
	go test -race -count=1 ./...

## bdd — run only the godog BDD suite
bdd:
	go test -race -count=1 -run TestFeatures ./tests/...

## lint-go — run golangci-lint
lint-go:
	golangci-lint run ./...

## lint-features — validate .feature files with gforge lint
lint-features:
	go run ./cmd/gforge lint features/

## scaffold-demo — generate a skeleton from the pilot feature into /tmp/demo
scaffold-demo:
	go run ./cmd/gforge scaffold \
		--feature features/business/create_order.feature \
		--out /tmp/gherkinforge-demo

## ci — full local CI sequence (same as GitHub Actions)
ci: lint-go lint-features test
