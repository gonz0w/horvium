#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Offline, read-only validation of the version-1 synthetic storage controls.

Expectations must come from independently verified host build/run records, not
the archive. This validates custody receipt consistency, not native attestation,
the complete inventory protocol, preflight, teardown or resource acceptance.
"""
import argparse
import base64
import hashlib
import importlib.util
import json
import os
import re
import stat
import sys
import tarfile
from pathlib import Path, PurePosixPath

SPEC = importlib.util.spec_from_file_location('dashboard_resource_contract', Path(__file__).with_name('dashboard-resource-evidence.py'))
C = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(C)
LABELS = ('normal', 'enospc-zero', 'file-limit-partial')
ARCHIVE_CAP = 272 << 20
FRAME_CAP = 65536
NAMESPACE = '0123456789abcdef0123456789abcdef'
FLAGS = ('PriorDenied', 'ForwardDenied', 'ResumeDenied', 'UnreadRejectedBody',
         'AllocationAfterFailure', 'ResourceRestored')


class StorageTarInfo(tarfile.TarInfo):
    @classmethod
    def frombuf(cls, buf, encoding, errors):
        member = super().frombuf(buf, encoding, errors)
        # Reject extension/sparse/link headers before tarfile can consume their
        # declared payloads. The supported original fixture uses plain entries.
        C.require(member.type in (tarfile.REGTYPE, tarfile.AREGTYPE, tarfile.DIRTYPE)
                  and 0 <= member.size <= (1 << 20), 'unsupported header or member size')
        return member


def sha(data):
    return hashlib.sha256(data).hexdigest()


def keys(value, required, optional=()):
    C.require(type(value) is dict and set(required) <= set(value) <= set(required) | set(optional),
              'missing or unknown receipt fields')


def exact(actual, expected):
    C.require(C.exact(actual, expected), 'receipt value/type mismatch')


def bounded_json(data):
    C.require(0 < len(data) <= FRAME_CAP, 'JSON byte bound')
    value = C.load_json(data)

    def walk(item, depth):
        C.require(depth <= 16 and type(item) in (dict, list, str, int, bool, type(None)), 'JSON depth/type bound')
        if type(item) is dict:
            for name, child in item.items():
                C.require(type(name) is str, 'JSON field name')
                walk(child, depth + 1)
        elif type(item) is list:
            for child in item:
                walk(child, depth + 1)
    walk(value, 0)
    return value


def framed(data):
    C.require(4 < len(data) <= FRAME_CAP + 4 and int.from_bytes(data[:4], 'big') == len(data) - 4,
              'bounded complete frame required')
    return bounded_json(data[4:])


def expectation(value):
    keys(value, ('schema', 'source_commit', 'base_commit', 'pr_title', 'release', 'source_archive_sha256',
                 'compiler_sha256', 'fixture_sha256', 'probe_sha256', 'archive_sha256'))
    exact(value['schema'], 1)
    for name in ('source_commit', 'base_commit'):
        C.digest(value[name], 40)
    C.require(value['source_commit'] != value['base_commit'] and value['release'] in C.RELEASES,
              'candidate range / pinned release required')
    C.require(re.fullmatch(r'(?:feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(?:\([^\n]+\))?!?: .+',
                           C.text(value['pr_title'])) is not None, 'actual PR title required')
    for name in ('source_archive_sha256', 'compiler_sha256', 'fixture_sha256', 'probe_sha256', 'archive_sha256'):
        C.digest(value[name])


def opened(path, ceiling):
    """Pin one regular descriptor; never follow its final symlink or block on FIFO."""
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    f = os.fdopen(fd, 'rb')
    try:
        info = os.fstat(f.fileno())
        C.require(stat.S_ISREG(info.st_mode) and 0 < info.st_size <= ceiling, 'bounded regular input required')
        return f, info
    except BaseException:
        f.close()
        raise


def stable(path, f, before):
    fields = ('st_dev', 'st_ino', 'st_size', 'st_mtime_ns', 'st_ctime_ns')
    for after in (os.fstat(f.fileno()), os.stat(path, follow_symlinks=False)):
        C.require(stat.S_ISREG(after.st_mode) and all(getattr(before, k) == getattr(after, k) for k in fields),
                  'input object changed during readback')


def chunks(f, limit):
    """Even a concurrently growing regular file cannot extend this read budget."""
    C.require(0 <= limit <= ARCHIVE_CAP + 1, 'input read budget')
    while limit:
        chunk = f.read(min(FRAME_CAP, limit))
        if not chunk:
            return
        limit -= len(chunk)
        yield chunk


def file_object(value, size):
    keys(value, ('device', 'inode', 'uid', 'gid', 'links', 'size', 'modified_ns'))
    for field in value:
        C.integer(value[field])
    C.require(value['inode'] > 0, 'original inode required')
    for name, want in (('uid', 0), ('gid', 0), ('links', 1), ('size', size)):
        exact(value[name], want)


def actor(value, fixture):
    keys(value, ('pid', 'uid', 'sec', 'usec', 'executable', 'sha256'))
    C.integer(value['pid'], 2)
    C.integer(value['sec'], 1)
    C.integer(value['usec'], 0, 999999)
    exact(value['uid'], 0)
    C.require(C.text(value['executable']).startswith('/'), 'original executable path required')
    exact(value['sha256'], fixture)


def golden_frames(data, request_id):
    """Framing/ID/count check only; the full independent Go decoder stays required."""
    rows, offset = [], 0
    while offset < len(data):
        C.require(offset + 4 <= len(data), 'truncated original header')
        size = int.from_bytes(data[offset:offset + 4], 'big')
        C.require(0 < size <= FRAME_CAP and offset + 4 + size <= len(data), 'bounded original frame required')
        row = bounded_json(data[offset + 4:offset + 4 + size])
        keys(row, ('version', 'request_id', 'type', 'data'))
        exact(row['version'], 1)
        exact(row['request_id'], request_id)
        rows.append(row)
        offset += size + 4
    C.require(2 <= len(rows) <= 100000 and rows[0]['type'] == 'begin' and rows[-1]['type'] == 'end',
              'original begin/end required')
    C.require(all(x['type'] in ('pool', 'dataset') for x in rows[1:-1]), 'original record order')
    exact(rows[-1]['data'], {'records': len(rows), 'complete': True})


def read_case(label, read, expected):
    prefix = 'witness-partial-storage/' + label + '/'
    payload = read(prefix + 'receipt.json')
    r = framed(payload)
    required = ('Label', 'DataClass', 'Controller', 'AttemptCap', 'RawCap', 'ArchiveCap', 'TmpfsCap',
                'ReleasedBytes', 'FillerBytesAfterRelease', 'SourceBytes', 'ExpectedPrefixBytes',
                'ExpectedPrefixSHA256', 'Boundary', 'StoreBytes', 'Attempts', 'PriorEligible',
                'Failed', 'Passed', 'Originals') + FLAGS
    optional = ('Fill',) if label == 'enospc-zero' else (('FileLimitBefore', 'FileLimitDuring', 'FileLimitAfter')
                                                       if label == 'file-limit-partial' else ())
    keys(r, required + optional)
    exact(r['Label'], label)
    exact(r['DataClass'], 'synthetic-custody-only')
    actor(r['Controller'], expected['fixture_sha256'])
    fault = label != 'normal'
    for key, value in (('AttemptCap', 32), ('RawCap', 67508864), ('ArchiveCap', 268435456),
                       ('TmpfsCap', 335544320), ('Attempts', 2 if fault else 1),
                       ('PriorEligible', True), ('Passed', True), ('Failed', fault)):
        exact(r[key], value)
    for name in FLAGS:
        exact(r[name], fault)
    for name in ('ReleasedBytes', 'FillerBytesAfterRelease', 'SourceBytes', 'ExpectedPrefixBytes', 'StoreBytes'):
        C.integer(r[name])
    C.require(type(r['Originals']) is list and len(r['Originals']) == (2 if fault else 1), 'original census')
    total = 0
    for index, item in enumerate(r['Originals'], 1):
        keys(item, ('Name', 'SHA256', 'Object', 'Mode', 'Ref'))
        name = f'original-{index:03d}.wire'
        exact(item['Name'], name)
        data = read(prefix + name)
        ref = item['Ref']
        complete = index == 1
        keys(ref, ('name', 'request_id', 'mode', 'management_incarnation', 'helper_incarnation', 'raw_sha256',
                   'raw_bytes', 'candidate_complete', 'delivered', 'forward_sha256'),
             ('forward',) if complete else ('capture_failure',))
        for key, value in (('name', name), ('request_id', NAMESPACE + f'-{index}'), ('mode', 'native-small'),
                           ('management_incarnation', 'capacity-management'), ('helper_incarnation', 'capacity-helper'),
                           ('raw_sha256', sha(data)), ('raw_bytes', len(data)), ('candidate_complete', complete),
                           ('delivered', complete), ('forward_sha256', sha(data) if complete else '')):
            exact(ref[key], value)
        exact(item['SHA256'], sha(data))
        exact(item['Mode'], 0o400)
        file_object(item['Object'], len(data))
        if complete:
            golden_frames(data, ref['request_id'])
            forward = ref.get('forward')
            keys(forward, ('stage', 'bytes', 'sha256', 'error', 'write_half_closed', 'elapsed_ns'))
            C.integer(forward['elapsed_ns'], 1)
            for k, v in (('stage', 'delivered'), ('bytes', len(data)), ('sha256', sha(data)),
                         ('error', {'class': ''}), ('write_half_closed', True)):
                exact(forward[k], v)
        else:
            exact(ref.get('capture_failure'), 'capture-io')
            size = 0 if label == 'enospc-zero' else 8193
            exact(data, (b'owned-partial-storage/' * 16384)[:size])
            exact(r['ExpectedPrefixBytes'], size)
            exact(r['ExpectedPrefixSHA256'], sha(data))
        total += len(data)
    exact(r['StoreBytes'], total)
    C.require(total < 268435456, 'archive accounting exceeded')
    exact(r['SourceBytes'], len(b'owned-partial-storage/') * 16384 if fault else 0)
    if not fault:
        exact(r['ExpectedPrefixBytes'], 0)
        exact(r['ExpectedPrefixSHA256'], '')
    error = {'class': 'syscall', 'errno': 28 if label == 'enospc-zero' else 27} if fault else {'class': ''}
    exact(r['Boundary'], {'ID': NAMESPACE + ('-2' if fault else '-1'), 'Management': 'capacity-management',
                         'Helper': 'capacity-helper', 'Stage': 'copy' if fault else 'sealed',
                         'AcknowledgedBytes': r['ExpectedPrefixBytes'] if fault else total, 'Error': error})
    if label == 'enospc-zero':
        fill = r['Fill']
        keys(fill, ('Bytes', 'ReportedWriteBytes', 'SHA256', 'Object', 'ENOSPC', 'SmallProbeENOSPC', 'SpaceReleased'))
        C.integer(fill['Bytes'], 32769, 335544320)
        C.integer(fill['ReportedWriteBytes'], fill['Bytes'])
        C.digest(fill['SHA256'])
        file_object(fill['Object'], fill['Bytes'])
        for name in ('ENOSPC', 'SmallProbeENOSPC'):
            exact(fill[name], True)
        exact(fill['SpaceReleased'], False)
        exact(r['ReleasedBytes'], 32768)
        exact(r['FillerBytesAfterRelease'], fill['Bytes'] - 32768)
    else:
        exact(r['ReleasedBytes'], 0)
        exact(r['FillerBytesAfterRelease'], 0)
    if label == 'file-limit-partial':
        for field in ('FileLimitBefore', 'FileLimitDuring', 'FileLimitAfter'):
            keys(r[field], ('Cur', 'Max'))
            for value in r[field].values():
                C.integer(value)
        C.require(r['FileLimitBefore']['Cur'] > 8193 and r['FileLimitBefore']['Max'] >= r['FileLimitBefore']['Cur'],
                  'original child limit')
        exact(r['FileLimitDuring'], {'Cur': 8193, 'Max': r['FileLimitBefore']['Max']})
        exact(r['FileLimitAfter'], r['FileLimitBefore'])
        wait = framed(read(prefix + 'child-wait.json'))
        keys(wait, ('PID', 'RawWait', 'ExitCode', 'ElapsedNS', 'Deadline', 'SendFailed', 'WaitFailed', 'Output'))
        for k, v in (('PID', r['Controller']['pid']), ('RawWait', 0), ('ExitCode', 0),
                     ('Deadline', False), ('SendFailed', False), ('WaitFailed', False)):
            exact(wait[k], v)
        C.integer(wait['ElapsedNS'], 1, 44999999999)
        C.require(type(wait['Output']) is str, 'child output encoding')
        output = base64.b64decode(wait['Output'], validate=True)
        C.require(b'--- PASS: TestNativeDashboardWitnessPartialFileLimitChild' in output, 'original child pass output')
    exact(framed(read(prefix + 'cleanup.json')), {'OriginalMount': True, 'UnderlyingDirectory': True, 'Removed': True})
    return {'case': label, 'receipt_sha256': sha(payload), 'originals': len(r['Originals']),
            'stored_bytes': total, 'capture_error': error, 'acknowledged_bytes': r['Boundary']['AcknowledgedBytes']}


def validate_archive(path, expected):
    """Return only scoped consistency evidence, after every original is checked."""
    try:
        expectation(expected)
        f, before = opened(path, ARCHIVE_CAP)
        with f:
            digest = hashlib.sha256()
            consumed = 0
            for chunk in chunks(f, before.st_size + 1):
                digest.update(chunk)
                consumed += len(chunk)
            C.require(consumed == before.st_size, 'archive size changed while hashing')
            exact(digest.hexdigest(), expected['archive_sha256'])
            f.seek(0)
            with tarfile.open(fileobj=f, mode='r:', tarinfo=StorageTarInfo) as archive:
                members = {}
                for member in archive:
                    C.require(len(members) < 128 and member.name not in members and
                              member.name == PurePosixPath(member.name).as_posix() and
                              not member.name.startswith('/') and '..' not in PurePosixPath(member.name).parts and
                              '\\' not in member.name and (member.isfile() or member.isdir()) and
                              member.sparse is None and member.uid == 0 and member.gid == 0,
                              'unsafe, duplicate or unsupported archive member')
                    C.require(member.size <= (1 << 20), 'member size bound')
                    members[member.name] = member
                used = set()

                def read(name):
                    C.require(name in members and members[name].isfile(), 'missing original file')
                    m = members[name]
                    mode = 0o600 if name.endswith('credential-probe.c') else (0o755 if name.endswith('/credential-probe') else 0o400)
                    C.require(m.mode == mode, 'original archive mode')
                    if name.endswith('.json'):
                        C.require(m.size <= FRAME_CAP + 4, 'receipt byte bound')
                    elif name.endswith('.wire'):
                        C.require(m.size <= FRAME_CAP, 'synthetic original bound')
                    used.add(name)
                    return archive.extractfile(m).read(m.size + 1)

                pb = read('witness-probe-original.json')
                exact(sha(pb), expected['probe_sha256'])
                probe = framed(pb)
                keys(probe, ('abi', 'source_sha256', 'binary_sha256', 'compiler_sha256', 'compiler_version', 'header_sha256'))
                exact(sha(read('witness-probe/credential-probe.c')), probe['source_sha256'])
                exact(sha(read('witness-probe/credential-probe')), probe['binary_sha256'])
                cases = [read_case(label, read, expected) for label in LABELS]
                exact(framed(read('witness-partial-storage-cleanup.json')), {'Verified': True, 'Mounts': dict.fromkeys(LABELS, True)})
                allowed_dirs = {'witness-partial-storage', 'witness-probe'} | {'witness-partial-storage/' + label for label in LABELS}
                C.require(all((m.name in allowed_dirs and m.mode == 0o700 and m.size == 0) if m.isdir()
                              else m.name in used for m in members.values()), 'unexpected original namespace entry')
                # tar readers otherwise ignore unparsed bytes after their end marker.
                f.seek(archive.offset)
                padding = 0
                for chunk in chunks(f, before.st_size - archive.offset):
                    C.require(not any(chunk), 'nonzero trailing archive bytes')
                    padding += len(chunk)
                C.require(padding >= 1024, 'complete archive end marker required')
            stable(path, f, before)
        return {'schema': 1, 'scope': 'synthetic-storage-custody-readback', **expected, 'cases': cases,
                'qualified': False, 'native_acceptance': False, 'resource_acceptance': False, 'resource_activation': False}
    except (OSError, tarfile.TarError, KeyError, TypeError, ValueError, OverflowError) as error:
        raise ValueError('storage archive or independently supplied expectations rejected') from error


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('archive', type=Path)
    parser.add_argument('expectation', type=Path)
    args = parser.parse_args()
    try:
        f, before = opened(args.expectation, FRAME_CAP)
        with f:
            expected = bounded_json(f.read(FRAME_CAP + 1))
            stable(args.expectation, f, before)
        result = validate_archive(args.archive, expected)
    except (OSError, ValueError):
        print('Storage evidence rejected; no acceptance result produced.', file=sys.stderr)
        return 1
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
