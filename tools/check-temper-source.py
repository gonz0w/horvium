#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Verify retained Temper bytes and selected role configuration, without Go."""
import hashlib
import os
import platform
import re
import stat
import sys
import tomllib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REVISION = "c07af93dc99e770bf4f3ec6f30a0536cb2ae14c5"
# Exact imported Git blobs, independently compared during adoption source review.
IMPORTED = {
    "LICENSE": "1bc9ce76b659a58c7b8edb88304880f36fd2a005f7b7f3349d1efc860d8deb87",
    "METHOD.md": "9ef31c00af31dcd6460d75153c2476b4ccfe4c899b584604deffff4d0baa794a",
    "docs/reference/documentation.md": "a3757a32af57cfb78b452b1c62dff685acc0ccdb5a23bff2ec2d17009e5ee39c",
    "docs/reference/extensions.md": "bdc1ef0504aa2c34e94616dc20b9f252de0c84284635a41e58a57744b9448195",
    "docs/reference/verification.md": "9cbc60b862f15e493d97f6f60985931112e1bac028ef18a4a27f2e470520ac9c",
    "docs/reference/workflow.md": "d17cc3f973ae42e9652b6eb02ccc201568d8434ea0400d0dd70785aae2734f4a",
    "profiles/codex.md": "9fa51cf35b66a4d08761bdb7b00312dc0bad0590f92bfc70910465d9ddcdd511",
    "profiles/github.md": "596cdfe85c232cb8d313fa467326d264ce625ed03a4c2fcc3d328b64d0f7fb7e",
    "profiles/strict-test-first.md": "cad47afe8be377119a3d5557d72cfb8f0a04fb878f9aa943b2d107c09b9d66fb",
}
ROLES = {"temper_workflow": ("gpt-6-luna", "medium"),
         "temper_implementer": ("gpt-6.1-sol", "high"),
         "temper_analyst": ("gpt-6-astra", "high"),
         "temper_reviewer": ("gpt-6.1-sol", "high")}

INSTRUCTIONS = {'temper_reviewer': '4e47ec22745174dc646080287899e0329b137164fc2d672b3fc12bd49fde9e69', 'temper_implementer': '32f79ee8e510281608f93854e65a0a5e76192e9a0d2d477faad8b4adb6c7d582', 'temper_analyst': '2cf9b5b6c2d61691bf35932aa1eb6b43b43aedb7c91f643d9616d08da7d5adb0', 'temper_workflow': 'b8872c3e6e6e4c6477b8c9714229d30f8cbe19cd45fb9c7f4b9e52b89311bb80'}
SOURCE_SHA256 = "bfd320a0d3fed521b3cfd9cd68c7e6fb25854bf1c49c99c7e1c493fb9b5ba80e"

def directory(path):
    mode = path.lstat().st_mode
    if stat.S_ISLNK(mode):
        raise ValueError("Inventory contains a symlink: " + str(path))
    if not stat.S_ISDIR(mode):
        raise ValueError("Unsupported inventory directory type: " + str(path))


def exact_inventory(base, files):
    """Admit exact parent directories and regular files, using lstat.

    Do not follow links or open a file until BOTH bundle and role inventories
    pass. Unexpected empty directories, FIFOs, sockets and devices all fail.
    """
    directory(base)
    directories = {parent.as_posix() for name in files for parent in Path(name).parents
                   if parent != Path(".")}
    seen_files, seen_directories = set(), set()

    def visit(folder):
        for path in folder.iterdir():
            name = path.relative_to(base).as_posix()
            mode = path.lstat().st_mode
            if stat.S_ISLNK(mode):
                raise ValueError("Inventory contains a symlink: " + name)
            if stat.S_ISDIR(mode) and name in directories:
                seen_directories.add(name)
                visit(path)
            elif stat.S_ISREG(mode) and not mode & 0o111 and name in files:
                seen_files.add(name)
            else:
                raise ValueError("Unexpected inventory entry or type: " + name)

    visit(base)
    if seen_files != set(files) or seen_directories != directories:
        raise ValueError("Incomplete inventory: " + str(base))


def validate(root):
    bundle = root / ".agent-temper"
    expected = set(IMPORTED) | {"SOURCE.md", "SHA256SUMS"}
    exact_inventory(bundle, expected)
    exact_inventory(root / ".codex", {"agents/" + name + ".toml" for name in ROLES})
    role_dir = root / ".codex/agents"
    exact_inventory(role_dir, {name + ".toml" for name in ROLES})
    manifest = {}
    for line in (bundle / "SHA256SUMS").read_text().splitlines():
        match = re.fullmatch(r"([0-9a-f]{64})  (.+)", line)
        if not match or match[2] not in expected - {"SHA256SUMS"} or match[2] in manifest:
            raise ValueError("Malformed or duplicate manifest entry")
        manifest[match[2]] = match[1]
    if set(manifest) != expected - {"SHA256SUMS"}:
        raise ValueError("Incomplete manifest")
    for name, digest in manifest.items():
        if hashlib.sha256((bundle / name).read_bytes()).hexdigest() != digest:
            raise ValueError("Manifest hash mismatch: " + name)
        if name in IMPORTED and digest != IMPORTED[name]:
            raise ValueError("Immutable upstream byte mismatch: " + name)
    source = (bundle / "SOURCE.md").read_text()
    if (hashlib.sha256(source.encode()).hexdigest() != SOURCE_SHA256 or
            f"Revision: `{REVISION}`" not in source or
            f"Source: https://github.com/gonz0w/agent-temper/tree/{REVISION}" not in source):
        raise ValueError("Upstream provenance mismatch")
    for name, settings in ROLES.items():
        role = tomllib.loads((role_dir / (name + ".toml")).read_text())
        if (role.get("name"), role.get("model"), role.get("model_reasoning_effort")) != (name, *settings):
            raise ValueError("Selected routing configuration mismatch: " + name)
        if set(role) != {"name", "description", "model", "model_reasoning_effort", "developer_instructions"}:
            raise ValueError("Unexpected role routing options: " + name)
        instructions = role.get("developer_instructions")
        if (type(instructions) is not str or
                hashlib.sha256(instructions.encode()).hexdigest() != INSTRUCTIONS[name]):
            raise ValueError("Accepted role instructions mismatch: " + name)
    project = (root / "PROJECT.md").read_text()
    agents = (root / "AGENTS.md").read_text()
    if ('Only the [Codex profile](.agent-temper/profiles/codex.md) is selected.' not in project or
            'and strict-test-first profiles are inactive.' not in project or
            'Only the Codex profile is selected in local project policy.' not in agents):
        raise ValueError("Codex-only profile selection contract mismatch")
    # Human policy review remains required; syntax and requested settings do not
    # establish runtime discovery/model metadata or native product acceptance.


def main():
    if platform.system() not in ("OpenBSD", "FreeBSD") or os.getuid() == 0 or sys.version_info[:3] != (3, 14, 7):
        raise RuntimeError("Pinned Python 3.14.7 on admitted unprivileged BSD is required")
    validate(ROOT)
    print("Temper inventory, immutable bytes, license, manifest, provenance and requested role configuration passed")
    print("Actual client role discovery and model routing remain unverified")


if __name__ == "__main__":
    main()
