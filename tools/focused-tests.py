#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Development-only selected behavioral tests; not PR or native qualification."""
import importlib.util
import os
import re
import subprocess
from pathlib import Path
ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('deep', ROOT / 'tools/deep-check.py')
DEEP = importlib.util.module_from_spec(spec)
spec.loader.exec_module(DEEP)


def main():
    package = DEEP.valid_package(os.environ.get('PKG', ''))
    pattern = os.environ.get('TEST', '')
    if not pattern or len(pattern) > 512 or any((ord(c) < 32 for c in pattern)):
        raise ValueError('set TEST to a bounded Go test-name expression')
    go = str(ROOT / '.tools/go/bin/go')
    env = DEEP.PLAN.go_environment(ROOT)
    listed = subprocess.check_output([go, 'test', package, '-list', pattern], cwd=ROOT, env=env, text=True)
    if not any((re.fullmatch('(?:Test|Example)[A-Za-z0-9_]+', line) for line in listed.splitlines())):
        raise ValueError('no matching behavioral tests; refused empty success')
    subprocess.run([go, 'test', package, '-run', pattern, '-count=1'], cwd=ROOT, env=env, check=True)
if __name__ == '__main__':
    main()
