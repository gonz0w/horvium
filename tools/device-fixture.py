#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Own five disposable QEMU test disks; never accepts host disk inputs."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import signal
import stat
import time

SIZE = 512 << 20
DISKS = [('HVM-UNIQUE-512', 512, 4096), ('HVM-UNIQUE-4K', 4096, 4096),
         ('HVM-DUPLICATE', 512, 512), ('HVM-DUPLICATE', 512, 512), ('', 512, 512)]


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


def prepare(root, size=SIZE):
    root = directory(root)
    root.mkdir(mode=0o700)
    records = []
    for i, (serial, logical, physical) in enumerate(DISKS):
        path = root / f'disk-{i}.raw'
        with path.open('xb') as target:
            os.chmod(path, 0o600)
            target.truncate(size)
        records.append({'file': path.name, 'serial': serial, 'logical': logical,
                        'physical': physical, 'bytes': size, 'sha256': digest(path)})
    (root / 'disks.json').write_text(json.dumps(records, sort_keys=True) + '\n')
    # PATH selects this wrapper only for this disposable launch. AnyVM and the
    # actual /usr/bin QEMU remain unmodified and separately checksum-recorded.
    bindir = root / 'bin'; bindir.mkdir(mode=0o700)
    wrapper = bindir / 'qemu-system-x86_64'
    source = str(Path(__file__).resolve())
    wrapper.write_text('#!/usr/bin/python3\nimport runpy,sys\n'
                       f"runpy.run_path({source!r})['launch'](sys.argv[1:], {str(root)!r})\n")
    wrapper.chmod(0o700)


def verify(root, size=SIZE):
    root = directory(root)
    if root.is_symlink() or not root.is_dir():
        raise ValueError('fixture directory unavailable')
    expected = json.loads((root / 'disks.json').read_text())
    if len(expected) != len(DISKS):
        raise ValueError('fixture manifest incomplete')
    for i, (serial, logical, physical) in enumerate(DISKS):
        record = expected[i]; path = root / f'disk-{i}.raw'
        info = path.lstat()
        if not stat.S_ISREG(info.st_mode) or info.st_size != size or stat.S_IMODE(info.st_mode) != 0o600:
            raise ValueError('fixture backing shape changed')
        if record != {'file': path.name, 'serial': serial, 'logical': logical,
                      'physical': physical, 'bytes': size, 'sha256': digest(path)}:
            raise ValueError('fixture backing changed')
    return expected


def command(args, root, size=SIZE):
    if args in [['--version'], ['-version'], ['-accel', 'help'], ['-cpu', 'help']]:
        return list(args)
    drives = [args[i+1] for i, a in enumerate(args[:-1]) if a == '-drive']
    boot_drives = [d for d in drives if 'if=pflash' not in d.split(',')]
    if args.count('-snapshot') != 1 or len(boot_drives) != 1 or len(drives) > 3:
        raise ValueError('requires one disposable snapshot boot drive')
    if '-pidfile' in args or '-blockdev' in args:
        raise ValueError('unexpected disk/process options')
    root = directory(root)
    verify(root, size=size)
    extra = []
    for i, (serial, logical, physical) in enumerate(DISKS):
        drive = f'horvium_device_{i}'
        extra += ['-drive', f'file={root}/disk-{i}.raw,format=raw,if=none,id={drive}',
                  '-device', f'virtio-blk-pci,drive={drive},logical_block_size={logical},physical_block_size={physical}' + (f',serial={serial}' if serial else '')]
    return list(args) + extra


def start_ticks(pid):
    return Path(f'/proc/{pid}/stat').read_text().rsplit(')', 1)[1].split()[19]


def launch(args, root):
    qualified = command(args, root)
    if qualified != args:
        record = {'pid': os.getpid(), 'start_ticks': start_ticks(os.getpid()),
                  'argv_sha256': hashlib.sha256(b'\0'.join(x.encode() for x in qualified)).hexdigest()}
        with (Path(root) / 'process.json').open('x') as out:
            json.dump(record, out)
    os.execv('/usr/bin/qemu-system-x86_64', ['/usr/bin/qemu-system-x86_64'] + qualified)


def verify_process(record, argv, ticks):
    pid = record['pid']
    if (type(pid) is not int or pid < 2 or ticks != record['start_ticks']
            or not argv or argv[0] != b'/usr/bin/qemu-system-x86_64'
            or hashlib.sha256(b'\0'.join(argv[1:])).hexdigest() != record['argv_sha256']):
        raise ValueError('fixture process identity changed; discard runner')
    return pid


def active_process(proc):
    try:
        return proc.joinpath('stat').read_text().rsplit(')', 1)[1].split()[0] != 'Z'
    except FileNotFoundError:
        return False


def cleanup(root):
    root = directory(root)
    record = json.loads((root / 'process.json').read_text())
    pid = record['pid']
    if type(pid) is not int or pid < 2:
        raise ValueError('unsafe fixture process')
    proc = Path(f'/proc/{pid}')
    if proc.exists():
        argv = (proc / 'cmdline').read_bytes().rstrip(b'\0').split(b'\0')
        verify_process(record, argv, start_ticks(pid))
        os.kill(pid, signal.SIGTERM)
        deadline = time.monotonic() + 10
        while active_process(proc) and time.monotonic() < deadline:
            # A terminated child can remain as a zombie awaiting AnyVM/init;
            # its file descriptors and backing access have already ended.
            time.sleep(0.05)
        if active_process(proc):
            raise ValueError('fixture process still active; discard runner')
    verify(root)
    for i in range(len(DISKS)):
        (root / f'disk-{i}.raw').unlink()
    (root / 'disks.json').unlink(); (root / 'process.json').unlink()
    (root / 'bin/qemu-system-x86_64').unlink(); (root / 'bin').rmdir(); root.rmdir()
    if root.exists():
        raise ValueError('fixture remains')
    print('DEVICE_FIXTURE_CLEANUP pass')


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
