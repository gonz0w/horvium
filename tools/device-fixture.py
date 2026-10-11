#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Own fixed disposable QEMU disk profiles; never accepts host disk inputs."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import stat

SIZE = 64 << 20
DISKS = [('HVM-UNIQUE-512', 512, 4096), ('HVM-UNIQUE-4K', 4096, 4096),
         ('HVM-DUPLICATE', 512, 512), ('HVM-DUPLICATE', 512, 512), ('', 512, 512)]

PROFILES = {'baseline': DISKS, 'policy-facts': DISKS + [
    ('HVM-FACTS-512', 512, 4096), ('HVM-FACTS-4K', 4096, 4096)]}


def profile_disks(profile):
    if profile not in PROFILES:
        raise ValueError('unknown disposable disk profile')
    return PROFILES[profile]


def directory(root):
    root = Path(root).absolute()
    if any(c not in '/abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-.' for c in str(root)):
        raise ValueError('unsafe QEMU option path')
    if root.resolve() != root:
        raise ValueError('symlink fixture path')
    return root


def digest(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def prepare(root, size=SIZE, profile='baseline'):
    disks = profile_disks(profile)
    root = directory(root)
    root.mkdir(mode=0o700)
    records = []
    for i, (serial, logical, physical) in enumerate(disks):
        path = root / f'disk-{i}.raw'
        with path.open('xb') as target:
            os.chmod(path, 0o600)
            target.truncate(size)
        records.append({'file': path.name, 'serial': serial, 'logical': logical,
                        'physical': physical, 'bytes': size, 'sha256': digest(path)})
    (root / 'disks.json').write_text(json.dumps(records, sort_keys=True) + '\n')


def verify(root, size=SIZE, profile='baseline'):
    disks = profile_disks(profile)
    root = directory(root)
    info = root.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        raise ValueError('fixture directory must remain owned and private')
    expected = json.loads((root / 'disks.json').read_text())
    if len(expected) != len(disks):
        raise ValueError('fixture manifest incomplete')
    for i, (serial, logical, physical) in enumerate(disks):
        record = expected[i]; path = root / f'disk-{i}.raw'
        info = path.lstat()
        if not stat.S_ISREG(info.st_mode) or info.st_size != size or stat.S_IMODE(info.st_mode) != 0o600:
            raise ValueError('fixture backing shape changed')
        if record != {'file': path.name, 'serial': serial, 'logical': logical,
                      'physical': physical, 'bytes': size, 'sha256': digest(path)}:
            raise ValueError('fixture backing changed')
    return expected


def command(args, root, size=SIZE, profile='baseline'):
    if args in [['--version'], ['-version'], ['-accel', 'help'], ['-cpu', 'help']]:
        return list(args)
    drives = [args[i+1] for i, a in enumerate(args[:-1]) if a == '-drive']
    boot_drives = [d for d in drives if 'if=pflash' not in d.split(',')]
    if args.count('-snapshot') != 1 or len(boot_drives) != 1 or len(drives) > 3:
        raise ValueError('requires one disposable snapshot boot drive')
    if any(flag in args for flag in ('-pidfile', '-blockdev', '-daemonize')):
        raise ValueError('unexpected disk/process options')
    disks = profile_disks(profile)
    root = directory(root)
    verify(root, size=size, profile=profile)
    extra = []
    for i, (serial, logical, physical) in enumerate(disks):
        drive = f'horvium_device_{i}'
        extra += ['-drive', f'file={root}/disk-{i}.raw,format=raw,if=none,id={drive}',
                  '-device', f'virtio-blk-pci,drive={drive},logical_block_size={logical},physical_block_size={physical}' + (f',serial={serial}' if serial else '')]
    return list(args) + extra


def remove(root, profile='baseline'):
    """Caller must stop/reap the owned guest/group before deleting its files."""
    disks = profile_disks(profile)
    root = directory(root)
    info = root.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        raise ValueError('untrusted fixture directory')
    for i in range(len(disks)):
        (root / f'disk-{i}.raw').unlink(missing_ok=True)
    (root / 'disks.json').unlink(missing_ok=True)
    # Unknown entries are preserved and make cleanup fail; never rmtree a
    # device directory or follow a backing symlink into another resource.
    root.rmdir()


def cleanup(root, size=SIZE, profile='baseline'):
    try:
        verify(root, size=size, profile=profile)
    finally:
        remove(root, profile=profile)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=['prepare', 'cleanup'])
    parser.add_argument('directory', type=Path)
    args = parser.parse_args()
    if args.mode == 'prepare':
        prepare(args.directory)
        print(json.dumps(verify(args.directory), sort_keys=True))
    else:
        cleanup(args.directory)


if __name__ == '__main__':
    main()
