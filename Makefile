.PHONY: tools check lint test secrets hooks

tools:
	python3 tools/bootstrap-tools.py

check:
	sh tools/check-repo.sh

lint:
	.tools/bin/shellcheck tools/check-repo.sh tools/bsd-foundation-smoke.sh .githooks/commit-msg .githooks/pre-commit
	.tools/bin/actionlint

test:
	.tools/venv/bin/python3 -m unittest discover -s tools/tests -v

secrets:
	.tools/bin/gitleaks dir --redact --no-banner --config .gitleaks.toml .

hooks:
	git config core.hooksPath .githooks
	@echo "Git hooks enabled for this clone"
