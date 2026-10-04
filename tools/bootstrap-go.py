#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Install the reviewed Go toolchain under ignored .tools; no system changes."""
import hashlib
import io
import json
import platform
import tempfile
import tarfile
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
pin = json.loads((ROOT / "tools/go-pins.json").read_text())
machine = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}.get(platform.machine())
target = f"{platform.system().lower()}_{machine}"
artifact = pin["artifacts"].get(target)
if artifact is None:
    raise SystemExit(f"No reviewed Go archive for {target}")
directory = ROOT / ".tools"
directory.mkdir(exist_ok=True)
binary = directory / "go/bin/go"
marker = directory / "go.sha256.json"
if binary.is_file() and marker.is_file():
    cached = json.loads(marker.read_text())
    if cached == {"artifact": artifact["sha256"], "binary": hashlib.sha256(binary.read_bytes()).hexdigest()}:
        print(f"Verified cached {pin['version']} for {target}")
        raise SystemExit(0)
request = urllib.request.Request(artifact["url"], headers={"User-Agent": "horvium-go-bootstrap"})
with urllib.request.urlopen(request, timeout=60) as response:
    data = response.read(artifact["size"] + 1)
if len(data) != artifact["size"] or hashlib.sha256(data).hexdigest() != artifact["sha256"]:
    raise SystemExit("Go archive size/checksum mismatch")
with tempfile.TemporaryDirectory(prefix="go-install-", dir=directory) as temporary:
    staged = Path(temporary)
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as archive:
        # Authenticate first, stage a clean tree, and reject escaping paths/links.
        if any(not (member.name.startswith("go/") or
                    member.name == "go" and member.isdir()) for member in archive.getmembers()):
            raise SystemExit("Unexpected Go archive layout")
        archive.extractall(staged, filter="data")
    current, previous = directory / "go", staged / "previous"
    if current.exists():
        current.rename(previous)
    try:
        (staged / "go").rename(current)
    except OSError:
        if previous.exists():
            previous.rename(current)
        raise
marker.write_text(json.dumps({"artifact": artifact["sha256"], "binary": hashlib.sha256(binary.read_bytes()).hexdigest()}) + "\n")
print(f"Installed checksum-verified {pin['version']} for {target}")
