#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Prepare an offline private distfile; builds/installs/runs no product code."""
import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile

ROOT = Path(__file__).resolve().parents[1]


def digest(data):
    return hashlib.sha256(data).hexdigest()


def safe_members(archive):
    members = archive.getmembers()
    for member in members:
        name = Path(member.name)
        if name.is_absolute() or '..' in name.parts or not (member.isdir() or member.isfile()):
            raise ValueError('source archive contains a path/link/type outside this profile')
    return members


def run(argv, *, cwd=ROOT, env=None):
    return subprocess.check_output(argv, cwd=cwd, env=env, timeout=90)


def go_environment(compiler, cache, out):
    # Do not inherit GOFLAGS/modfile/overlay, GONOPROXY/GOPRIVATE direct-fetch
    # exceptions, an alternate GOROOT, or credential/proxy/VCS configuration.
    return dict(PATH='/usr/bin:/bin:/usr/sbin:/sbin', HOME='/nonexistent',
                LANG='C', LC_ALL='C', TZ='UTC', GOTOOLCHAIN='local',
                GOROOT=str(compiler.resolve().parents[1]), GOENV='off', GOWORK='off',
                GOFLAGS='', GOPROXY='off', GOSUMDB='off', GOPRIVATE='',
                GONOPROXY='', GONOSUMDB='', GOINSECURE='', GOVCS='*:off',
                GOTELEMETRY='off', GOMAXPROCS='2', CGO_ENABLED='0',
                GOMODCACHE=str(cache), GOCACHE=str(out / 'go-cache'),
                TMPDIR=str(out / 'go-tmp'))


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--ref', required=True, help='full signed source commit')
    p.add_argument('--version', required=True, help='explicit private lab package version, not a release tag')
    p.add_argument('--go', type=Path, required=True)
    p.add_argument('--go-sha256', required=True)
    p.add_argument('--module-cache', type=Path, required=True)
    p.add_argument('--allowed-signers', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    args = p.parse_args()
    if not re.fullmatch('[0-9a-f]{40}', args.ref) or not re.fullmatch('[0-9a-f]{64}', args.go_sha256):
        p.error('full source commit and compiler SHA256 required')
    if not re.fullmatch('[0-9]+(?:[.][0-9]+)+', args.version):
        p.error('lab version must contain numeric dot-separated components')
    out = args.out.absolute()
    if out.resolve() != out or out.exists() or out == ROOT or ROOT in out.parents:
        p.error('use a fresh non-symlink output directory outside the checkout')
    compiler = args.go.absolute()
    if not compiler.is_file() or digest(compiler.read_bytes()) != args.go_sha256:
        p.error('pinned compiler digest mismatch')
    cache = args.module_cache.resolve(strict=True)
    if not cache.is_dir():
        p.error('existing module cache required; downloads are disabled')
    signer = args.allowed_signers.resolve(strict=True)
    run(['git', '-c', 'gpg.format=ssh', '-c', 'gpg.ssh.allowedSignersFile=' + str(signer),
         'verify-commit', args.ref])
    tree = run(['git', 'rev-parse', args.ref + '^{tree}']).decode().strip()
    stamp = int(run(['git', 'show', '-s', '--format=%ct', args.ref]))
    original = run(['git', 'archive', '--format=tar', args.ref])
    os.umask(0o077)
    out.mkdir(mode=0o700)
    name = 'horvium-' + args.version + '-' + args.ref
    staged = out / 'tree' / name
    staged.mkdir(parents=True)
    with tarfile.open(fileobj=io.BytesIO(original), mode='r:') as archive:
        archive.extractall(staged, members=safe_members(archive), filter='data')
    pins = json.loads((staged / 'tools/go-pins.json').read_text())
    (out / 'go-tmp').mkdir(mode=0o700)
    env = go_environment(compiler, cache, out)
    version = run([str(compiler), 'version'], env=env).decode().strip()
    if version.split()[2] != pins['version']:
        raise ValueError('compiler version does not match signed source pins')
    modules = {n: digest((staged / n).read_bytes()) for n in ['go.mod', 'go.sum']}
    run([str(compiler), 'mod', 'verify'], cwd=staged, env=env)
    run([str(compiler), 'mod', 'vendor'], cwd=staged, env=env)
    if any(digest((staged / n).read_bytes()) != sha for n, sha in modules.items()):
        raise ValueError('module inputs changed; retain failed preparation, do not use it')
    vendor = {}
    for f in sorted((staged / 'vendor').rglob('*')):
        if f.is_symlink():
            raise ValueError('vendored symlink refused')
        if f.is_file():
            vendor[f.relative_to(staged).as_posix()] = digest(f.read_bytes())
    if 'vendor/modules.txt' not in vendor:
        raise ValueError('complete vendored module closure required')
    provenance = {'schema': 1, 'source_commit': args.ref, 'source_tree': tree,
                  'original_git_archive_sha256': digest(original), 'lab_version': args.version,
                  'go_version': version, 'go_binary_sha256': args.go_sha256,
                  'preparer_sha256': digest(Path(__file__).read_bytes()),
                  'go_environment_policy': 'fixed_allowlist_no_ambient_go_proxy_or_vcs_overrides',
                  'module_inputs': modules, 'vendor_files': len(vendor),
                  'vendor_manifest_sha256': digest(json.dumps(vendor, sort_keys=True).encode()),
                  'native_package_built': False, 'runtime_qualified': False}
    (staged / 'PORT-SOURCE.json').write_text(json.dumps(provenance, sort_keys=True, indent=2) + '\n')
    distfile = out / (name + '.tar.gz')
    with distfile.open('xb') as raw, gzip.GzipFile(filename='', mode='wb', fileobj=raw, mtime=0) as zipped:
        with tarfile.open(fileobj=zipped, mode='w|', format=tarfile.PAX_FORMAT) as archive:
            for f in [staged, *sorted(staged.rglob('*'))]:
                if f.is_symlink() or not (f.is_file() or f.is_dir()):
                    raise ValueError('unsupported staged source object')
                info = archive.gettarinfo(str(f), arcname=name if f == staged else name + '/' + f.relative_to(staged).as_posix())
                info.uid = info.gid = 0
                info.uname = info.gname = 'root'
                info.mtime = 0
                info.pax_headers = {}
                info.mode = 0o755 if f.is_dir() or f.stat().st_mode & 0o111 else 0o644
                if f.is_file():
                    with f.open('rb') as content:
                        archive.addfile(info, content)
                else:
                    archive.addfile(info)
    checksum = digest(distfile.read_bytes())
    key = 'horvium/' + distfile.name
    (out / 'distinfo').write_text(f'TIMESTAMP = {stamp}\nSHA256 ({key}) = {checksum}\nSIZE ({key}) = {distfile.stat().st_size}\n')
    (out / 'source.mk').write_text(f'HORVIUM_SOURCE_REV=\t{args.ref}\nDISTVERSION=\t{args.version}\n')
    record = dict(provenance, distfile=distfile.name, sha256=checksum, bytes=distfile.stat().st_size,
                  module_downloads='disabled_with_GOPROXY_off', network_operations_requested=False, retained_stage=True)
    (out / 'bundle.json').write_text(json.dumps(record, indent=2) + '\n')
    print(json.dumps(record, indent=2))


if __name__ == '__main__':
    main()
