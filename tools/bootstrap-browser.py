#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Install checksum-pinned Linux browser test tools under ignored .tools only."""
import hashlib
import json
import platform
import shutil
import tarfile
import tempfile
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def verify(data, expected):
    if hashlib.sha256(data).hexdigest() != expected:
        raise ValueError('browser artifact checksum mismatch')


def install(pin, target):
    if target.exists():
        raise ValueError('browser installation already exists; refusing replacement')
    with urllib.request.urlopen(pin['url'], timeout=60) as response:
        data = response.read(200 * 1024 * 1024 + 1)
    if len(data) > 200 * 1024 * 1024:
        raise ValueError('browser artifact exceeds download bound')
    verify(data, pin['sha256'])
    target.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(dir=target.parent) as temporary:
        stage = Path(temporary)
        archive = stage / 'download'
        archive.write_bytes(data)
        unpacked = stage / 'unpacked'
        unpacked.mkdir()
        with tarfile.open(archive, 'r:*') as contents:
            contents.extractall(unpacked, filter='data')
        unpacked.rename(target)


def main():
    if platform.system() != 'Linux' or platform.machine() not in ('x86_64', 'amd64'):
        raise SystemExit('Pinned Firefox fixture requires Linux amd64; Safari uses installed macOS driver')
    pins = json.loads((ROOT / 'tools/browser-pins.json').read_text())
    target = ROOT / '.tools/browser'
    if target.exists():
        raise SystemExit('Existing browser fixture retained; remove it explicitly before reinstalling')
    target.mkdir(parents=True)
    try:
        for name, pin in pins.items():
            install(pin, target / name)
    except BaseException:
        shutil.rmtree(target)
        raise
    print('Verified Firefox and geckodriver development fixtures installed')


if __name__ == '__main__':
    main()
