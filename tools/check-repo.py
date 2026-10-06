#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Check complete working-tree content, including committed files before review."""
import json
import re
import subprocess
import sys
import tomllib
from pathlib import Path
from urllib.parse import unquote

try:
    import yaml
except ImportError:
    sys.exit("Run make tools first (PyYAML is a pinned development dependency).")

ROOT = Path(__file__).resolve().parent.parent
REQUIRED = (
    "README.md", "CONTRIBUTING.md", "SECURITY.md", "GOVERNANCE.md", "AGENTS.md",
    "LICENSE", "CODE_OF_CONDUCT.md", ".github/CODEOWNERS",
    ".github/PULL_REQUEST_TEMPLATE.md", ".github/dependabot.yml",
    ".github/main-ruleset.json", "tools/review-gate.py", "tools/review-plan.py", "tools/deep-check.py",
    "tools/focused-tests.py", "tools/bsd-state-prebuilt.sh", "tools/owned-process.py", "docs/templates/TDD_TEMPLATE.md",
    "NOTICE.md", "docs/project/charter.md", "docs/architecture/baseline.md",
    "docs/engineering/bootstrap-status.md", "docs/testing/strategy.md",
    "docs/compatibility/matrix.md", "packaging/ports/README.md",
)
EXCLUDED = {".git", ".tools", "__pycache__"}
TEXT_SUFFIXES = {".md", ".py", ".sh", ".go", ".mod", ".sum", ".c", ".yml", ".yaml", ".json", ".txt", ".toml", ".html", ".css", ".js"}
errors = []


class StrictLoader(yaml.SafeLoader):
    """Reject duplicate keys instead of silently overriding settings."""


def mapping(loader, node, deep=False):
    result = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if key in result:
            raise ValueError(f"duplicate YAML key: {key}")
        result[key] = loader.construct_object(value_node, deep=deep)
    return result


StrictLoader.add_constructor(yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, mapping)


def error(path, message):
    errors.append(f"{path}: {message}")


for name in REQUIRED:
    if not (ROOT / name).is_file() or not (ROOT / name).stat().st_size:
        error(name, "required file missing or empty")

for path in sorted(ROOT.rglob("*")):
    rel = path.relative_to(ROOT)
    if any(part in EXCLUDED for part in rel.parts) or not path.is_file():
        continue
    if path.is_symlink():
        error(rel, "symlinks require an explicit policy before adoption")
        continue
    if path.suffix in {".pem", ".key"} or path.name in {"id_rsa", "id_ed25519", ".env"}:
        error(rel, "private-key or secret-file pattern")
    is_text = path.suffix in TEXT_SUFFIXES or path.name in {
        "Makefile", "LICENSE", "CODEOWNERS", "commit-msg", "pre-commit",
        ".gitignore", ".gitattributes", ".editorconfig", ".python-version",
    }
    if not is_text:
        continue
    try:
        text = path.read_bytes().decode("utf-8")
    except UnicodeDecodeError:
        error(rel, "expected UTF-8 text")
        continue
    if "\r" in text:
        error(rel, "CRLF/CR line endings")
    if text and not text.endswith("\n"):
        error(rel, "missing final newline")
    for number, line in enumerate(text.splitlines(), 1):
        if line.rstrip(" \t") != line:
            trailing = line[len(line.rstrip(" \t")):]
            if not (path.suffix == ".md" and trailing == "  " and line.strip()):
                error(rel, f"line {number}: trailing whitespace")
        if re.search(r"^<{7} |^={7}$|^>{7} ", line):
            error(rel, f"line {number}: unresolved merge conflict")
    if re.search(r"-----BEGIN (?:[A-Z]+ )?PRIVATE KEY-----", text):
        error(rel, "private key content")
    if re.search(r"\b(?:gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{50,}|AKIA[A-Z0-9]{16})\b", text):
        error(rel, "credential-like content")
    if path.name in {"CODEOWNERS", "SECURITY.md", "CODE_OF_CONDUCT.md", "config.yml"}:
        if "@PROJECT-MAINTAINERS" in text or "github.com/OWNER/REPO" in text:
            error(rel, "unresolved bootstrap placeholder")
    if path.suffix == ".md":
        for target in re.findall(r"\[[^\]]*\]\(([^\s)]+)\)", text):
            if re.match(r"^[a-zA-Z][a-zA-Z0-9+.-]*:", target) or target.startswith("#"):
                continue
            target = unquote(target.split("#", 1)[0].strip("<>"))
            if target and not (path.parent / target).exists():
                error(rel, f"missing local link target: {target}")
    if path.suffix in {".yml", ".yaml", ".json", ".toml"}:
        try:
            data = (json.loads(text) if path.suffix == ".json" else
                    tomllib.loads(text) if path.suffix == ".toml" else
                    yaml.load(text, Loader=StrictLoader))
        except (ValueError, yaml.YAMLError) as exc:
            error(rel, f"invalid configuration: {exc}")
            continue
        if rel.parts[:2] == (".github", "workflows") and path.suffix in {".yml", ".yaml"}:
            error(rel, "GitHub Actions are disabled by repository policy")
        if str(rel) == '.github/main-ruleset.json':
            types = {rule['type'] for rule in data.get('rules', [])}
            for required in ('deletion', 'non_fast_forward', 'required_linear_history', 'required_signatures', 'pull_request'):
                if required not in types:
                    error(rel, 'main protection missing: ' + required)
            if 'required_status_checks' in types:
                error(rel, 'local review policy forbids required hosted status checks')
            if data.get('bypass_actors') or data.get('enforcement') != 'active':
                error(rel, 'main protections must remain active without bypasses')
            pr = next((rule.get('parameters', {}) for rule in data.get('rules', []) if rule['type'] == 'pull_request'), {})
            if not pr.get('required_review_thread_resolution'):
                error(rel, 'main PR conversations must be resolved')

if subprocess.run(["git", "rev-parse", "--is-inside-work-tree"], cwd=ROOT, capture_output=True).returncode == 0:
    for args in (["git", "diff", "--check", "--", ".", ":(exclude)*.md"], ["git", "diff", "--cached", "--check", "--", ".", ":(exclude)*.md"]):
        result = subprocess.run(args, cwd=ROOT, capture_output=True, text=True)
        if result.returncode:
            error("git", result.stdout.strip() or result.stderr.strip())

if errors:
    print("\n".join(f"ERROR: {message}" for message in errors), file=sys.stderr)
    sys.exit(1)
print("Repository content, configuration, license, links, and local review policy passed")
