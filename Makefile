.PHONY: tools go-tools check lint lint-tools lint-go test test-tools test-go secrets hooks go-check go-deep-check build browser-preview review lab-status lab-stop

GO := $(CURDIR)/.tools/go/bin/go
export GOTOOLCHAIN := local
export GOENV := off
export GOCACHE := $(CURDIR)/.tools/go-cache
export GOMODCACHE := $(CURDIR)/.tools/go-modules
export CGO_ENABLED := 0

tools:
	python3 tools/bootstrap-tools.py
	python3 tools/bootstrap-go.py

go-tools:
	python3 tools/bootstrap-go.py

check:
	sh tools/check-repo.sh

lint: lint-tools lint-go

lint-tools:
	.tools/bin/shellcheck tools/check-repo.sh tools/bsd-foundation-smoke.sh tools/bsd-native-discovery.sh tools/bsd-performance.sh tools/bsd-continuity.sh tools/bsd-properties.sh tools/bsd-devices.sh tools/research/m1-native.sh .githooks/commit-msg .githooks/pre-commit

lint-go:
	@test -z "$$($(CURDIR)/.tools/go/bin/gofmt -l cmd internal tools/research/m1-geom.go)" || { echo 'Run gofmt on Go sources'; exit 1; }
	$(GO) vet ./...

test: test-tools test-go

test-tools:
	.tools/venv/bin/python3 -m unittest discover -s tools/tests -v

test-go:
	$(GO) test ./...

go-check: lint-go test-go go-deep-check

go-deep-check:
	$(GO) test ./internal/storage -run '^$$' -fuzz '^FuzzDeviceUseReview$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/storage -run '^$$' -fuzz '^FuzzFilesystemSignature$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/storage -run '^$$' -fuzz '^FuzzImportedPoolUse$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/storage -run '^$$' -fuzz '^FuzzSystemDeviceUse$$' -fuzztime=3s -parallel=2
	CGO_ENABLED=1 $(GO) test -race ./...
	$(GO) test ./internal/inventory -run '^$$' -fuzz '^FuzzPools$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/inventory -run '^$$' -fuzz '^FuzzGEOM$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/inventory -run '^$$' -fuzz '^FuzzDiskObservation$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/inventory -run '^$$' -fuzz '^FuzzDiskAttachments$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/helper -run '^$$' -fuzz '^FuzzRequest$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/helper -run '^$$' -fuzz '^FuzzSnapshot$$' -fuzztime=3s -parallel=2

	$(GO) test ./internal/admin -run '^$$' -fuzz '^FuzzConfiguration$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/storage -run '^$$' -fuzz '^FuzzPropertyPlan$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/storage -run '^$$' -fuzz '^FuzzProperties$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/storage -run '^$$' -fuzz '^FuzzIdentities$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/storage -run '^$$' -fuzz '^FuzzFilesystemStates$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/storage -run '^$$' -fuzz '^FuzzPoolState$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/storage -run '^$$' -fuzz '^FuzzPoolFeatures$$' -fuzztime=3s -parallel=2
	$(GO) test ./internal/storage -run '^$$' -fuzz '^FuzzPoolTopology$$' -fuzztime=3s -parallel=2

secrets:
	.tools/bin/gitleaks dir --redact --no-banner --config .gitleaks.toml .

hooks:
	git config core.hooksPath .githooks
	@echo "Git hooks enabled for this clone"

build:
	mkdir -p .tools/bin
	$(GO) build -trimpath -o .tools/bin/horvium ./cmd/horvium

# Public synthetic loopback fixture; expires after twenty minutes.
browser-preview:
	HORVIUM_UI_PREVIEW=1 .tools/venv/bin/python3 tools/owned-process.py run -- $(GO) test -v ./internal/console -run '^TestBrowserPreview$$' -count=1 -timeout=25m

# Complete local review gate: committed source, title, DCO/signatures and evidence.
review:
	.tools/venv/bin/python3 tools/review-gate.py

lab-status:
	.tools/venv/bin/python3 tools/owned-process.py status

lab-stop:
	.tools/venv/bin/python3 tools/owned-process.py stop
