SEMVER_BUMPS = major minor patch

export GO111MODULE := on
export GOPROXY := https://gocenter.io

./bin/goreleaser:
	curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh

./bin/svu:
	curl -sfL https://install.goreleaser.com/github.com/caarlos0/svu.sh | sh

./bin/golangci-lint:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh

lint: ./bin/golangci-lint
	./bin/golangci-lint run --tests=false --enable-all --disable=lll ./...
.PHONY: lint

$(SEMVER_BUMPS): ./bin/svu ./bin/goreleaser
	./release.sh $@
.PHONY: $(SEMVER_BUMPS)

.DEFAULT_GOAL := lint
