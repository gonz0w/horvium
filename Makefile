.PHONY: tools go-tools check lint lint-tools lint-go test test-tools test-go secrets hooks go-check go-deep-check build browser-preview review review-full test-focus lab-status lab-stop

GO := $(CURDIR)/.tools/go/bin/go
export GOTOOLCHAIN := local
export GOENV := off
export GOCACHE := $(CURDIR)/.tools/go-cache
export GOMODCACHE := $(CURDIR)/.tools/go-modules
export CGO_ENABLED := 0

tools: storage-preflight
	python3 tools/bootstrap-tools.py
	python3 tools/bootstrap-go.py

go-tools: storage-preflight
	python3 tools/bootstrap-go.py

check:
	sh tools/check-repo.sh

lint: lint-tools lint-go

lint-tools:
	.tools/bin/shellcheck -s sh packaging/ports/horvium/files/horvium.in packaging/ports/horvium/files/horvium_helper.in packaging/ports/horvium/files/horvium_rc.subr.in packaging/ports/horvium/files/pkg-pre-install.in packaging/ports/horvium/files/pkg-pre-deinstall.in
	.tools/bin/shellcheck tools/bsd-policy-startup-provenance-prebuilt.sh
	.tools/bin/shellcheck tools/research/m2-executor-recovery.sh
	.tools/bin/shellcheck tools/bsd-closed-startup-publication-prebuilt.sh tools/bsd-closed-startup-owner-prebuilt.sh tools/bsd-policy-startup-publication-prebuilt.sh tools/bsd-policy-startup-owner-prebuilt.sh
	.tools/bin/shellcheck tools/check-repo.sh tools/bsd-foundation-smoke.sh tools/bsd-native-discovery.sh tools/bsd-performance.sh tools/bsd-continuity.sh tools/bsd-properties.sh tools/bsd-devices.sh tools/bsd-state-prebuilt.sh tools/bsd-execution-system-prebuilt.sh tools/bsd-joint-leases-prebuilt.sh tools/bsd-original-backing-prebuilt.sh tools/bsd-persistent-backing-prebuilt.sh tools/bsd-pool-properties-prebuilt.sh tools/bsd-storage-job-status-prebuilt.sh tools/bsd-action-peer-prebuilt.sh tools/research/m1-native.sh .githooks/commit-msg .githooks/pre-commit

lint-go: storage-preflight
	@test -z "$$($(CURDIR)/.tools/go/bin/gofmt -l cmd internal tools/research/m1-geom.go)" || { echo 'Run gofmt on Go sources'; exit 1; }
	$(GO) vet ./...

test: test-tools test-go

test-tools:
	.tools/venv/bin/python3 -m unittest discover -s tools/tests -v

test-go: storage-preflight
	$(GO) test ./...

go-check: lint-go test-go go-deep-check

go-deep-check: storage-preflight
	.tools/venv/bin/python3 tools/deep-check.py --all

# Development-only narrow behavioral loop; PKG and TEST are validated by Python.
test-focus: storage-preflight
	.tools/venv/bin/python3 tools/owned-process.py run -- .tools/venv/bin/python3 tools/focused-tests.py

secrets:
	.tools/bin/gitleaks dir --redact --no-banner --config .gitleaks.toml .

hooks:
	git config core.hooksPath .githooks
	@echo "Git hooks enabled for this clone"

build: storage-preflight
	mkdir -p .tools/bin
	$(GO) build -trimpath -o .tools/bin/horvium ./cmd/horvium

# Public synthetic loopback fixture; expires after twenty minutes.
browser-preview: storage-preflight
	HORVIUM_UI_PREVIEW=1 .tools/venv/bin/python3 tools/owned-process.py run -- $(GO) test -v ./internal/console -run '^TestBrowserPreview$$' -count=1 -timeout=25m

# Proportional local review gate: committed source, title, DCO/signatures and evidence.
review: storage-preflight
	.tools/venv/bin/python3 tools/review-gate.py

# Fresh complete portable milestone/release gate; never reuses deep evidence.
review-full: storage-preflight
	.tools/venv/bin/python3 tools/review-gate.py --full

lab-status:
	.tools/venv/bin/python3 tools/owned-process.py status

lab-stop:
	.tools/venv/bin/python3 tools/owned-process.py stop

.PHONY: storage-preflight

storage-preflight:
	python3 tools/storage-preflight.py
