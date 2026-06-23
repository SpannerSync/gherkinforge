.PHONY: test bdd lint-go lint-features lint integration mutation scaffold-demo ci build tidy

## build — compile the gforge CLI binary
build:
	go build -o bin/gforge ./cmd/gforge

## tidy — sync go.mod and go.sum
tidy:
	go mod tidy

## test — run all unit and BDD tests
test:
	go test -race -count=1 ./...

## bdd — run only the @business godog BDD suite
bdd:
	go test -race -count=1 -run TestFeatures ./tests/...

## lint-go — Zero Trust Pillar 1: enforce hexagonal boundaries via depguard
lint-go:
	golangci-lint run ./...

## lint-features — Zero Trust Pillar 2: validate .feature files (tier tags + UI word ban)
lint-features:
	go run ./cmd/gforge lint features/

## lint — run both Go linter and feature linter
lint: lint-go lint-features

## integration — Zero Trust Pillar 3: run @integration suite with tx rollback (requires Docker)
integration:
	go test -tags=integration -race -count=1 -v ./tests/integration/...

## mutation — Zero Trust Pillar 4: mutation testing against the domain layer
## Requires: go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest
## CI fails if mutation score < 80%
mutation:
	go-mutesting ./pkg/context/order/... | tee /tmp/mutation-report.txt; \
	grep -E "^The mutation score" /tmp/mutation-report.txt

## scaffold-demo — generate a skeleton from the pilot feature into /tmp/demo
scaffold-demo:
	go run ./cmd/gforge scaffold \
		--feature features/business/create_order.feature \
		--out /tmp/gherkinforge-demo

## ci — full Zero Trust pipeline (Pillars 1–4 in sequence)
ci: lint-go lint-features test mutation
