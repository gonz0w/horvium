#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Race/fuzz only selected packages, or discover every portable target for --all."""
import argparse
import importlib.util
import json
import re
import subprocess
from pathlib import Path
ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('plan', ROOT / 'tools/review-plan.py')
PLAN = importlib.util.module_from_spec(spec)
spec.loader.exec_module(PLAN)


def valid_package(package):
    if not re.fullmatch('\\./(?:internal|cmd)/[A-Za-z0-9_/-]+', package) or any((part in {'.', '..', ''} for part in package[2:].split('/'))):
        raise ValueError('invalid focused package')
    return package


def fuzz_names(text):
    if re.search('(?m)^FAIL', text):
        raise ValueError('fuzz discovery failed')
    names = []
    for line in text.splitlines():
        if line.startswith('Fuzz'):
            if not re.fullmatch('Fuzz[A-Z][A-Za-z0-9_]*', line) or line in names:
                raise ValueError('invalid/duplicate compiler-listed fuzz target')
            names.append(line)
    return sorted(names)


def fuzz_command(go, package, name):
    valid_package(package)
    if not re.fullmatch('Fuzz[A-Z][A-Za-z0-9_]*', name):
        raise ValueError('invalid fuzz name')
    return [go, 'test', package, '-run', '^$', '-fuzz', '^' + name + '$', '-fuzztime=3s', '-parallel=2']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    select = parser.add_mutually_exclusive_group(required=True)
    select.add_argument('--all', action='store_true')
    select.add_argument('--packages', nargs='+')
    args = parser.parse_args()
    go = str(ROOT / '.tools/go/bin/go')
    env = PLAN.go_environment(ROOT)
    if args.all:
        values = PLAN.json_stream(subprocess.check_output([go, 'list', '-json', './...'], cwd=ROOT, env=env, text=True))
        packages = sorted(('./' + str(Path(p['Dir']).relative_to(ROOT)) for p in values))
    else:
        packages = sorted(set(args.packages))
    for package in packages:
        valid_package(package)
    if not packages:
        raise ValueError('no packages selected')
    targets = []
    for package in packages:
        listed = subprocess.check_output([go, 'test', package, '-list', '^Fuzz'], cwd=ROOT, env=env, text=True)
        targets.extend(({'package': package, 'target': name} for name in fuzz_names(listed)))
    race_env = dict(env, CGO_ENABLED='1')
    subprocess.run([go, 'test', '-race', *packages], cwd=ROOT, env=race_env, check=True)
    for target in targets:
        command = fuzz_command(go, target['package'], target['target'])
        print('Running ' + ' '.join(command), flush=True)
        subprocess.run(command, cwd=ROOT, env=env, check=True)
    print('DEEP_RESULT ' + json.dumps({'race_packages': packages, 'fuzz_targets': targets}, sort_keys=True), flush=True)
if __name__ == '__main__':
    main()
