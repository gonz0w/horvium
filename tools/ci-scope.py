#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Keep required checks running; omit Go only for an allowlisted prose change."""
import os
import re
import subprocess
from pathlib import Path, PurePosixPath


def needs_go(paths):
    if not paths:
        return True
    prose = {'README.md', 'CONTRIBUTING.md', 'SECURITY.md', 'GOVERNANCE.md',
             'AGENTS.md', 'CODE_OF_CONDUCT.md', 'NOTICE.md', 'LICENSE'}
    for name in paths:
        path = PurePosixPath(name)
        if any(ord(c) < 32 for c in name) or '..' in path.parts:
            return True
        if name not in prose and not (name.startswith('docs/') and path.suffix == '.md'):
            return True
    return False


def main():
    base, head = os.environ.get('BASE_SHA', ''), os.environ.get('HEAD_SHA', '')
    required = True
    if base and head and set(base) != {'0'}:
        if not all(re.fullmatch(r'[0-9a-f]{40}', rev) for rev in (base, head)):
            raise ValueError('expected full Git object IDs')
        result = subprocess.run(['git', 'diff', '--no-renames', '--name-only', '-z',
                                 base, head, '--'], check=True, capture_output=True)
        paths = result.stdout.decode('utf-8').split('\0')[:-1]
        required = needs_go(paths)
    output = 'go=' + str(required).lower() + '\n'
    print(output, end='')
    if os.environ.get('GITHUB_OUTPUT'):
        with Path(os.environ['GITHUB_OUTPUT']).open('a') as target:
            target.write(output)


if __name__ == '__main__':
    main()
