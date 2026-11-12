#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Bound ordinary development output before allocation; never deletes existing data."""
import importlib.util
from pathlib import Path
import shutil
import sys

_spec = importlib.util.spec_from_file_location('storage_guard', Path(__file__).with_name('storage_guard.py'))
guard = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(guard)


def reserve_bytes(total):
    # Workstation floor without imposing a 200 GB requirement on a small guest.
    return min(200_000_000_000, total // 10)


def preflight(root):
    output = root / '.tools'
    try:
        output.mkdir(exist_ok=True)
        roots = [output.resolve()]
        if (root / 'bin').exists():
            roots.append((root / 'bin').resolve())
        if len({p.stat().st_dev for p in roots}) != 1:
            raise ValueError('output roots on different filesystems require separate reviewed budgets')
        disk = shutil.disk_usage(roots[0])
        used = guard.allocated_bytes(roots)
        reserve = reserve_bytes(disk.total)
        problems = guard.check_capacity(used, disk.free, 2_000_000_000, 20_000_000_000, reserve)
        print(f'output_allocated_bytes={used} free_bytes={disk.free} peak_bytes=2000000000 '
              f'budget_bytes=20000000000 reserve_bytes={reserve}')
        for problem in problems:
            print(problem, file=sys.stderr)
        return 1 if problems else 0
    except (OSError, ValueError) as error:
        print(f'storage preflight unavailable: {error}', file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(preflight(Path(__file__).resolve().parents[1]))
