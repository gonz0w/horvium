#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Stage checksum-pinned test images; this tool installs no host/NAS service."""
import argparse
import hashlib
import json
import subprocess
import urllib.request
from pathlib import Path

PINS = Path(__file__).with_name('performance-vm-pins.json')


def digest(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def download(url, expected, maximum, target):
    # Exclusive creation protects any existing fixture, including on failure.
    with target.open('xb') as output:
        try:
            total = 0
            hash_value = hashlib.sha256()
            with urllib.request.urlopen(url, timeout=120) as source:
                while chunk := source.read(1024 * 1024):
                    total += len(chunk)
                    if total > maximum:
                        raise ValueError('download exceeds pinned size')
                    hash_value.update(chunk)
                    output.write(chunk)
            if hash_value.hexdigest() != expected:
                raise ValueError('artifact checksum mismatch')
        except BaseException:
            target.unlink(missing_ok=True)
            raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('release', choices=['15.1', '14.5'])
    parser.add_argument('directory', type=Path)
    parser.add_argument('--verify', action='store_true')
    args = parser.parse_args()
    pins = json.loads(PINS.read_text())
    image = pins['images'][args.release]
    root = args.directory.resolve()
    qcow = root / 'freebsd' / ('v' + pins['builder']) / ('freebsd-' + args.release + '.qcow2')
    runtime = root / 'anyvm.py'
    metadata = root / 'image-hashes.json'
    if args.verify:
        recorded = json.loads(metadata.read_text())
        if recorded['compressed_sha256'] != image['sha256'] or digest(qcow) != recorded['qcow2_sha256']:
            raise ValueError('base image changed; performance evidence invalid')
        if digest(runtime) != pins['anyvm']['sha256']:
            raise ValueError('VM runtime changed; performance evidence invalid')
        print(json.dumps(recorded, sort_keys=True))
        return
    root.mkdir(parents=True, exist_ok=False)
    qcow.parent.mkdir(parents=True)
    archive = root / 'image.qcow2.zst'
    download(image['url'], image['sha256'], image['bytes'], archive)
    subprocess.run(['unzstd', '--quiet', str(archive), '-o', str(qcow)], check=True)
    archive.unlink()
    download(pins['anyvm']['url'], pins['anyvm']['sha256'], 2 * 1024 * 1024, runtime)
    recorded = {'release': args.release, 'builder': pins['builder'],
                'compressed_sha256': image['sha256'], 'qcow2_sha256': digest(qcow),
                'anyvm_sha256': digest(runtime), 'cpu_model': pins['cpu_model'],
                'cpu_count': pins['cpu_count'], 'memory_mib': pins['memory_mib']}
    metadata.write_text(json.dumps(recorded, indent=2) + '\n')
    print(json.dumps(recorded, sort_keys=True))


if __name__ == '__main__':
    main()
