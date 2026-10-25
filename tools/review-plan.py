#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Conservative impact selection and exact-input portable evidence reuse."""
import hashlib
import json
import os
import platform
import shlex
import subprocess
from pathlib import Path
POLICY = {'AGENTS.md', 'CONTRIBUTING.md', 'docs/engineering/local-review.md', 'docs/testing/strategy.md'}
TEMPER_PATHS = frozenset({
    '.agent-temper/LICENSE', '.agent-temper/METHOD.md',
    '.agent-temper/SOURCE.md', '.agent-temper/SHA256SUMS',
    '.agent-temper/docs/reference/documentation.md',
    '.agent-temper/docs/reference/extensions.md',
    '.agent-temper/docs/reference/verification.md',
    '.agent-temper/docs/reference/workflow.md',
    '.agent-temper/profiles/codex.md', '.agent-temper/profiles/github.md',
    '.agent-temper/profiles/strict-test-first.md',
    '.codex/agents/temper_workflow.toml',
    '.codex/agents/temper_implementer.toml',
    '.codex/agents/temper_analyst.toml',
    '.codex/agents/temper_reviewer.toml',
})


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def documentation_path(path):
    if path in POLICY:
        return False
    return path in {'README.md', 'NOTICE.md', 'CHANGELOG.md'} or (path.startswith('docs/') and (path.endswith('.md') or path.endswith('-manifest.json')))


def go_environment(root, native=False):
    env = dict(os.environ)
    for key in ['GOOS', 'GOARCH']:
        env.pop(key, None)
    env.update(GOTOOLCHAIN='local', GOENV='off', GOWORK='off', GOFLAGS='', CGO_ENABLED='0', GOCACHE=str(root / '.tools/go-cache'), GOMODCACHE=str(root / '.tools/go-modules'))
    if native:
        env.update(GOOS='freebsd', GOARCH='amd64', GOAMD64='v1')
    return env


def json_stream(text):
    decoder = json.JSONDecoder()
    values = []
    while text.strip():
        text = text.lstrip()
        value, end = decoder.raw_decode(text)
        values.append(value)
        text = text[end:]
    return values


def merge_graphs(root, datasets):
    graph = {}
    names = {}
    for dataset in datasets:
        for p in dataset:
            try:
                directory = Path(p['Dir']).relative_to(root).as_posix()
            except (KeyError, ValueError):
                continue
            if not directory.startswith(('internal/', 'cmd/')):
                continue
            name = './' + directory
            names[p['ImportPath']] = name
            node = graph.setdefault(name, {'directory': directory, 'imports': set()})
            node['imports'].update(p.get('Imports', []) + p.get('TestImports', []) + p.get('XTestImports', []))
    for node in graph.values():
        node['imports'] = sorted({names[name] for name in node['imports'] if name in names})
    return graph


def package_graph(root):
    datasets = []
    for native in [False, True]:
        text = subprocess.check_output([str(root / '.tools/go/bin/go'), 'list', '-json', './...'], cwd=root, env=go_environment(root, native), text=True)
        datasets.append(json_stream(text))
    return merge_graphs(root, datasets)


def select_plan(paths, graph, force_full=False):

    def full(reason):
        return {'mode': 'full', 'reason': reason, 'packages': [], 'native_review_required': True}
    if force_full:
        return full('explicit full milestone/release review')
    if not paths:
        return full('empty/unknown change selection')
    if all((documentation_path(p) for p in paths)):
        return {'mode': 'documentation', 'reason': 'documentation/evidence only; baseline still required', 'packages': [], 'native_review_required': False}
    if all((p in TEMPER_PATHS or documentation_path(p) for p in paths)):
        return {'mode': 'temper', 'reason': 'pinned Temper sources and selected local roles; metadata checks required', 'packages': [], 'native_review_required': False}
    changed = set()
    for p in paths:
        if documentation_path(p):
            continue
        if p in POLICY or p in {'go.mod', 'go.sum', 'Makefile'} or p.startswith(('tools/', '.githooks/', '.github/', 'packaging/')):
            return full('shared tooling, policy, module or packaging change')
        owners = [name for name, node in graph.items() if p.startswith(node['directory'] + '/')]
        if not owners:
            return full('unknown or unmapped input: ' + p)
        changed.add(max(owners, key=lambda name: len(graph[name]['directory'])))
    affected = set(changed)
    while True:
        more = {name for name, node in graph.items() if set(node['imports']) & affected}
        if more <= affected:
            break
        affected.update(more)
    sensitive = any((name.startswith(('./internal/storage', './internal/inventory', './internal/host/', './internal/helper', './internal/admin', './cmd/')) for name in changed))
    return {'mode': 'focused', 'reason': 'changed packages plus transitive host/FreeBSD/test consumers', 'packages': sorted(affected), 'native_review_required': sensitive}


def changed_paths(root, base, source):
    data = subprocess.check_output(['git', 'diff', '--name-only', '--no-renames', '-z', base, source], cwd=root)
    return sorted(set((p.decode('utf-8') for p in data.split(b'\x00') if p)))


def input_fingerprint(root, source):
    data = subprocess.check_output(['git', 'ls-tree', '-r', '-z', source], cwd=root)
    inputs = {}
    for entry in data.split(b'\x00'):
        if not entry:
            continue
        metadata, path = entry.split(b'\t', 1)
        name = path.decode('utf-8')
        if not documentation_path(name):
            inputs[name] = metadata.decode('ascii')
    encoded = json.dumps(inputs, sort_keys=True, separators=(',', ':')).encode()
    return (hashlib.sha256(encoded).hexdigest(), inputs)


def environment_fingerprint(root):
    go = str(root / '.tools/go/bin/go')
    env = go_environment(root)
    settings = json.loads(subprocess.check_output([go, 'env', '-json'], cwd=root, env=env, text=True))
    # Go synthesizes GOGCCFLAGS with a fresh compiler scratch directory on every
    # query. Its inputs (CC/CGO flags/target/SDK) remain covered by this digest.
    settings.pop('GOGCCFLAGS', None)
    compiler = subprocess.check_output(shlex.split(settings['CC']) + ['--version'], env=env)
    # Hash compiler settings/output; never retain arbitrary host environment dumps.
    fields = {'host': platform.system(), 'arch': platform.machine(), 'release': platform.release(), 'go_binary_sha256': digest(Path(go)), 'go_settings_sha256': hashlib.sha256(json.dumps(settings, sort_keys=True).encode()).hexdigest(), 'compiler_version_sha256': hashlib.sha256(compiler).hexdigest(),
              'behavioral_environment_sha256': hashlib.sha256(json.dumps({name: env.get(name, '') for name in ['GORACE', 'GODEBUG', 'GOMAXPROCS', 'GOTRACEBACK']}, sort_keys=True).encode()).hexdigest()}
    return hashlib.sha256(json.dumps(fields, sort_keys=True).encode()).hexdigest()


def reusable(report, key, base, log):
    try:
        evidence = report['deep_evidence']
        supervisor = evidence['supervisor']
        return report.get('schema') == 2 and report.get('qualified') is True and (report.get('base_commit') == base) and (report.get('deep_key') == key) and (evidence.get('executed') is True) and (supervisor.get('completed') is True) and (supervisor.get('group_cleaned') is True) and (supervisor.get('worker_returncode') == 0) and log.is_file() and (log.stat().st_size <= 4 << 20) and (digest(log) == evidence.get('log_sha256'))
    except (KeyError, TypeError, OSError):
        return False
