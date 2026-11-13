#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Admit reviewed Temper/policy bytes, then validate their inert contracts."""
import hashlib
import json
import os
import re
import stat
import sys
import tomllib
from pathlib import Path, PurePosixPath

ROOT = Path(__file__).resolve().parents[1]
# Reviewed #444 Temper template refresh over canonical consumer fe3afe9.
# This trusted verifier anchor is deliberately separate from candidate checksums.
# Any covered byte change needs explicit source review and readmission here.
REVIEWED_ADMISSION_SHA256 = '97870bb7434429533dd58f306593c20e8247860ae6c89153177a0bc70b5f2e15'


def directory(path):
    mode = path.lstat().st_mode
    if stat.S_ISLNK(mode):
        raise ValueError("Inventory contains a symlink: " + str(path))
    if not stat.S_ISDIR(mode):
        raise ValueError("Unsupported inventory directory type: " + str(path))


def inventory(base):
    """Inspect complete inventories before reading any bundle or role content."""
    directory(base)
    files, directories = set(), {base}

    def visit(folder):
        entries = list(folder.iterdir())
        if not entries:
            raise ValueError("Unexpected empty inventory directory: " + str(folder))
        for path in entries:
            mode = path.lstat().st_mode
            if stat.S_ISLNK(mode):
                raise ValueError("Inventory contains a symlink: " + str(path))
            if stat.S_ISDIR(mode):
                directories.add(path)
                visit(path)
            elif stat.S_ISREG(mode) and not mode & 0o111:
                files.add(path)
            else:
                raise ValueError("Unexpected inventory entry or type: " + str(path))
    visit(base)
    return files, directories


def contained_file(root, name):
    """Reject traversal, links and special files before opening selected paths."""
    if (type(name) is not str or not name or '\\' in name or
            PurePosixPath(name).is_absolute() or
            any(part in {'', '.', '..'} for part in name.split('/'))):
        raise ValueError("Invalid selected module path")
    path = root
    for part in name.split('/')[:-1]:
        path = path / part
        directory(path)
    path = root / name
    mode = path.lstat().st_mode
    if stat.S_ISLNK(mode):
        raise ValueError("Selected path contains a symlink: " + name)
    if not stat.S_ISREG(mode) or mode & 0o111:
        raise ValueError("Unsupported selected file type: " + name)
    return path


def selection(raw):
    """Only structural policy parsing precedes admission; no declared code runs."""
    policy = tomllib.loads(raw.decode('utf-8'))
    if (set(policy) != {'schema_version', 'modules'} or
            type(policy['schema_version']) is not int or policy['schema_version'] != 1 or
            type(policy['modules']) is not list or not policy['modules']):
        raise ValueError('Invalid root policy declaration')
    modules = policy['modules']
    ids, paths = set(), set()
    for module in modules:
        if type(module) is not dict or set(module) != {'id', 'path', 'owner', 'requires'}:
            raise ValueError('Invalid selected module declaration')
        if (type(module['id']) is not str or not re.fullmatch('[a-z][a-z0-9-]*', module['id']) or
                module['id'] in ids or type(module['path']) is not str or module['path'] in paths or
                type(module['owner']) is not str or not module['owner'].strip() or
                type(module['requires']) is not list or
                any(type(item) is not str for item in module['requires']) or
                len(set(module['requires'])) != len(module['requires'])):
            raise ValueError('Invalid selected module identity or prerequisites')
        ids.add(module['id'])
        paths.add(module['path'])
    visited, active = set(), set()
    by_id = {module['id']: module for module in modules}
    def visit(name):
        if name not in ids or name in active:
            raise ValueError('Missing or cyclic selected module prerequisite')
        if name in visited:
            return
        active.add(name)
        for prerequisite in by_id[name]['requires']:
            visit(prerequisite)
        active.remove(name)
        visited.add(name)
    for name in ids:
        visit(name)
    return modules


def admission_input(root):
    """Read complete, safely typed data and frame sorted path/type/hash records.

    The result is still untrusted until checked against the reviewed anchor.
    No candidate-supplied checksum, environment setting or Git object admits it.
    """
    directory(root)
    bundle_files, bundle_dirs = inventory(root / '.agent-temper')
    role_files, role_dirs = inventory(root / '.codex')
    policy_path = contained_file(root, 'policy.toml')
    agents_path = contained_file(root, 'AGENTS.md')
    policy_raw = policy_path.read_bytes()
    modules = selection(policy_raw)
    selected = {contained_file(root, module['path']) for module in modules}
    files = bundle_files | role_files | selected | {policy_path, agents_path}
    contents = {path.relative_to(root).as_posix(): path.read_bytes() for path in sorted(files)}
    contents['policy.toml'] = policy_raw
    records = [{'path': path.relative_to(root).as_posix(), 'type': 'directory'}
               for path in bundle_dirs | role_dirs]
    records += [{'path': name, 'type': 'file', 'sha256': hashlib.sha256(raw).hexdigest()}
                for name, raw in contents.items()]
    framed = json.dumps({'domain': 'horvium-temper-reviewed-admission-v1',
                         'entries': sorted(records, key=lambda item: item['path'])},
                        sort_keys=True, separators=(',', ':'), ensure_ascii=True).encode('utf-8')
    return hashlib.sha256(framed).hexdigest(), contents, modules


def admitted_inputs(root):
    digest, contents, modules = admission_input(root)
    if digest != REVIEWED_ADMISSION_SHA256:
        raise ValueError('Reviewed admission mismatch: inventory, immutable upstream, provenance, role/routing or profile selection bytes changed')
    return contents, modules


def validate(root):
    contents, modules = admitted_inputs(root)
    bundle = {name.removeprefix('.agent-temper/'): raw for name, raw in contents.items()
              if name.startswith('.agent-temper/')}
    if 'LICENSE' not in bundle or not bundle['LICENSE'].strip():
        raise ValueError('Missing complete retained license')
    manifest = {}
    for line in bundle['SHA256SUMS'].decode().splitlines():
        match = re.fullmatch(r'([0-9a-f]{64})  (.+)', line)
        if not match or match[2] not in bundle or match[2] == 'SHA256SUMS' or match[2] in manifest:
            raise ValueError('Malformed or duplicate manifest entry')
        manifest[match[2]] = match[1]
    if set(manifest) != set(bundle) - {'SHA256SUMS'}:
        raise ValueError('Incomplete manifest')
    for name, digest in manifest.items():
        if hashlib.sha256(bundle[name]).hexdigest() != digest:
            raise ValueError('Manifest hash mismatch: ' + name)
    source = bundle['SOURCE.md'].decode()
    revisions = re.findall(r'(?m)^Revision: `([0-9a-f]{40})`$', source)
    urls = re.findall(r'(?m)^Source: (\S+)$', source)
    if (len(revisions) != 1 or urls != ['https://github.com/gonz0w/agent-temper/tree/' + revisions[0]] or
            'Repository: https://github.com/gonz0w/agent-temper' not in source):
        raise ValueError('Upstream provenance mismatch')
    roles = {}
    for name, raw in contents.items():
        if not name.startswith('.codex/'):
            continue
        if not re.fullmatch(r'.codex/agents/[a-z][a-z0-9_]*\.toml', name):
            raise ValueError('Unexpected role inventory')
        role = tomllib.loads(raw.decode())
        if (set(role) != {'name', 'description', 'model', 'model_reasoning_effort', 'developer_instructions'} or
                any(type(value) is not str or not value.strip() for value in role.values()) or
                role['name'] != Path(name).stem or role['name'] in roles or
                role['model_reasoning_effort'] not in {'minimal', 'low', 'medium', 'high', 'xhigh', 'max', 'ultra'}):
            raise ValueError('Invalid selected role routing configuration: ' + name)
        roles[role['name']] = role
    if not roles:
        raise ValueError('Missing selected roles')
    # Selection was structurally checked before opening paths; admission binds
    # its exact IDs, prerequisites and prose to the reviewed source.
    return frozenset(name for name in contents if name.startswith(('.agent-temper/', '.codex/')))


def main():
    if os.name != 'posix' or os.getuid() == 0 or sys.version_info[:3] != (3, 14, 7):
        raise RuntimeError('Pinned Python 3.14.7 on a permitted unprivileged POSIX host is required')
    validate(ROOT)
    print('Reviewed Temper inventory, immutable bytes, license, manifest, provenance, selected policy and requested roles passed')
    print('Actual client role discovery and model routing remain unverified')


if __name__ == '__main__':
    main()
