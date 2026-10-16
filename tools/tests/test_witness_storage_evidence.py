# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic archive controls; these fixtures are not FreeBSD evidence."""
import base64
import copy
import hashlib
import importlib.util
import io
import json
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest import mock
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('storage_evidence', ROOT / 'tools/witness-storage-evidence.py')
E = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(E)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def frame(value):
    payload = json.dumps(value, separators=(',', ':')).encode()
    return len(payload).to_bytes(4, 'big') + payload


def fixture():
    actor = dict(pid=101, uid=0, sec=1000, usec=10, executable='/owned/fixture.test', sha256='a' * 64)
    identity = '0123456789abcdef0123456789abcdef'
    golden = b''.join(frame(dict(version=1, request_id=identity + '-1', type=kind, data=data))
                      for kind, data in [('begin', {}), ('end', dict(records=2, complete=True))])
    files = {}
    for label in ('normal', 'enospc-zero', 'file-limit-partial'):
        fault = label != 'normal'
        r = dict(Label=label, DataClass='synthetic-custody-only', Controller=copy.deepcopy(actor),
                 AttemptCap=32, RawCap=67508864, ArchiveCap=268435456, TmpfsCap=335544320,
                 Attempts=2 if fault else 1, PriorEligible=True, Passed=True, Failed=fault,
                 PriorDenied=fault, ForwardDenied=fault, ResumeDenied=fault, UnreadRejectedBody=fault,
                 AllocationAfterFailure=fault, ResourceRestored=fault, ReleasedBytes=0,
                 FillerBytesAfterRelease=0, SourceBytes=360448 if fault else 0,
                 ExpectedPrefixBytes=0, ExpectedPrefixSHA256='', Originals=[])
        if label == 'file-limit-partial':
            r['Controller']['pid'] = 102
            r.update(FileLimitBefore=dict(Cur=1000000, Max=1000000),
                     FileLimitDuring=dict(Cur=8193, Max=1000000),
                     FileLimitAfter=dict(Cur=1000000, Max=1000000))
        for index, raw in enumerate([golden] + ([b'' if label == 'enospc-zero' else
                                               (b'owned-partial-storage/' * 16384)[:8193]] if fault else []), 1):
            name = f'original-{index:03d}.wire'
            complete = index == 1
            ref = dict(name=name, request_id=identity + f'-{index}', mode='native-small',
                       management_incarnation='capacity-management', helper_incarnation='capacity-helper',
                       raw_sha256=sha(raw), raw_bytes=len(raw), candidate_complete=complete,
                       delivered=complete, forward_sha256=sha(raw) if complete else '')
            if complete:
                ref['forward'] = dict(stage='delivered', bytes=len(raw), sha256=sha(raw), error=dict(class_=''),
                                      write_half_closed=True, elapsed_ns=100)
                ref['forward']['error'] = {'class': ''}
            else:
                ref['capture_failure'] = 'capture-io'
            obj = dict(device=1, inode=index, uid=0, gid=0, links=1, size=len(raw), modified_ns=123)
            r['Originals'].append(dict(Name=name, SHA256=sha(raw), Object=obj, Mode=0o400, Ref=ref))
            files[f'witness-partial-storage/{label}/{name}'] = (raw, 0o400)
        r['StoreBytes'] = sum(x['Ref']['raw_bytes'] for x in r['Originals'])
        if fault:
            r['ExpectedPrefixBytes'] = r['Originals'][1]['Ref']['raw_bytes']
            r['ExpectedPrefixSHA256'] = r['Originals'][1]['SHA256']
        r['Boundary'] = dict(ID=identity + ('-2' if fault else '-1'), Management='capacity-management',
                             Helper='capacity-helper', Stage='copy' if fault else 'sealed',
                             AcknowledgedBytes=r['ExpectedPrefixBytes'] if fault else len(golden),
                             Error={'class': 'syscall', 'errno': 28 if label == 'enospc-zero' else 27} if fault else {'class': ''})
        if label == 'enospc-zero':
            r.update(Fill=dict(Bytes=335540224, ReportedWriteBytes=335540224, SHA256='b' * 64,
                               Object=dict(device=1, inode=4, uid=0, gid=0, links=1, size=335540224, modified_ns=123),
                               ENOSPC=True, SmallProbeENOSPC=True, SpaceReleased=False),
                     ReleasedBytes=32768, FillerBytesAfterRelease=335540224 - 32768)
        if label == 'file-limit-partial':
            wait = dict(PID=102, RawWait=0, ExitCode=0, ElapsedNS=1000, Deadline=False,
                        SendFailed=False, WaitFailed=False,
                        Output=base64.b64encode(b'--- PASS: TestNativeDashboardWitnessPartialFileLimitChild (0.1s)\nPASS\n').decode())
            files[f'witness-partial-storage/{label}/child-wait.json'] = (frame(wait), 0o400)
        files[f'witness-partial-storage/{label}/receipt.json'] = (frame(r), 0o400)
        files[f'witness-partial-storage/{label}/cleanup.json'] = (frame(dict(OriginalMount=True, UnderlyingDirectory=True, Removed=True)), 0o400)
    files['witness-partial-storage-cleanup.json'] = (frame(dict(Verified=True, Mounts={x: True for x in ('normal', 'enospc-zero', 'file-limit-partial')})), 0o400)
    source, binary = b'synthetic C probe', b'synthetic executable'
    probe = dict(abi={}, source_sha256=sha(source), binary_sha256=sha(binary), compiler_sha256='c' * 64,
                 compiler_version='synthetic', header_sha256={})
    files['witness-probe-original.json'] = (frame(probe), 0o400)
    files['witness-probe/credential-probe.c'] = (source, 0o600)
    files['witness-probe/credential-probe'] = (binary, 0o755)
    expected = dict(schema=1, source_commit='1' * 40, base_commit='2' * 40, pr_title='test(dashboards): synthetic evidence fixture',
                    release='15.1', source_archive_sha256='d' * 64, compiler_sha256='e' * 64,
                    fixture_sha256=actor['sha256'], probe_sha256=sha(files['witness-probe-original.json'][0]),
                    archive_sha256='')
    return files, expected


def archive(files, extras=(), compressed=False):
    out = io.BytesIO()
    with tarfile.open(fileobj=out, mode='w:gz' if compressed else 'w:') as a:
        for name, (data, mode) in files.items():
            member = tarfile.TarInfo(name)
            member.size, member.mode, member.uid, member.gid = len(data), mode, 0, 0
            a.addfile(member, io.BytesIO(data))
        for member, data in extras:
            a.addfile(member, io.BytesIO(data))
    return out.getvalue()


def mutate_record(files, name, mutate):
    value = json.loads(files[name][0][4:])
    mutate(value)
    files[name] = (frame(value), files[name][1])


class StorageEvidenceTests(unittest.TestCase):
    def validate(self, files=None, expected=None, extras=(), compressed=False):
        original, context = fixture()
        payload = archive(files if files is not None else original, extras, compressed)
        context['archive_sha256'] = sha(payload)
        if expected:
            context.update(expected)
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / 'originals.tar'
            path.write_bytes(payload)
            return E.validate_archive(path, context)

    def test_complete_scoped_readback_never_grants_acceptance(self):
        r = self.validate()
        self.assertEqual(r['scope'], 'synthetic-storage-custody-readback')
        self.assertEqual(len(r['cases']), 3)
        self.assertEqual(sum(x['originals'] for x in r['cases']), 5)
        for name in ('native_acceptance', 'resource_acceptance', 'qualified', 'resource_activation'):
            self.assertIs(r[name], False)
        self.assertEqual(self.validate(expected={'release': '14.5'})['release'], '14.5')

    def test_external_archive_and_probe_and_fixture_pins_required(self):
        for key in ('archive_sha256', 'probe_sha256', 'fixture_sha256'):
            with self.subTest(key=key), self.assertRaises(ValueError):
                self.validate(expected={key: '0' * 64})

    def test_expectations_reject_unknown_fields_bool_schema_and_invalid_identity(self):
        for edit in ({'extra': True}, {'schema': True}, {'release': '16.0'}, {'source_commit': 'bad'},
                     {'source_commit': '2' * 40}, {'compiler_sha256': 'bad'}):
            with self.subTest(edit=edit), self.assertRaises(ValueError):
                self.validate(expected=edit)

    def test_retained_failure_is_not_a_successful_matrix(self):
        files, _ = fixture()
        name = 'witness-partial-storage/file-limit-partial/receipt.json'
        mutate_record(files, name, lambda r: r.update(Passed=False))
        with self.assertRaises(ValueError):
            self.validate(files)

    def test_receipt_and_original_mutations_fail_even_with_new_archive_hash(self):
        name = 'witness-partial-storage/file-limit-partial/receipt.json'
        changes = [lambda r: r.update(StoreBytes=True), lambda r: r.update(AttemptCap=33),
                   lambda r: r.update(PriorDenied=False), lambda r: r.update(Attempts=1),
                   lambda r: r.update(ExpectedPrefixSHA256='0' * 64),
                   lambda r: r['Controller'].update(uid=1), lambda r: r['Controller'].update(pid=True),
                   lambda r: r['Originals'][1]['Ref'].update(candidate_complete=True),
                   lambda r: r['Originals'][1]['Ref'].update(delivered=True),
                   lambda r: r['Originals'][1]['Ref'].update(raw_bytes=0),
                   lambda r: r['Originals'][1]['Object'].update(links=2),
                   lambda r: r['Boundary'].update(ID='f' * 32 + '-2'),
                   lambda r: r['Boundary']['Error'].update(errno=28),
                   lambda r: r['FileLimitAfter'].update(Cur=8193),
                   lambda r: r['FileLimitDuring'].update(Max=8193), lambda r: r.update(unknown=True)]
        for index, change in enumerate(changes):
            files, _ = fixture()
            mutate_record(files, name, change)
            with self.subTest(index=index), self.assertRaises(ValueError):
                self.validate(files)
        files, _ = fixture()
        files['witness-partial-storage/file-limit-partial/original-002.wire'] = (b'X' * 8193, 0o400)
        with self.assertRaises(ValueError):
            self.validate(files)

    def test_wait_and_cleanup_are_not_inferred_from_other_receipts(self):
        changes = [('child-wait.json', lambda r: r.update(PID=999)),
                   ('child-wait.json', lambda r: r.update(Deadline=True)),
                   ('child-wait.json', lambda r: r.update(ExitCode=1)),
                   ('child-wait.json', lambda r: r.update(ElapsedNS=45000000001)),
                   ('cleanup.json', lambda r: r.update(Removed=False))]
        for name, change in changes:
            files, _ = fixture()
            mutate_record(files, 'witness-partial-storage/file-limit-partial/' + name, change)
            with self.subTest(name=name, change=change), self.assertRaises(ValueError):
                self.validate(files)

    def test_original_framing_is_checked_after_consistent_hash_rebinding(self):
        for field, value in (('request_id', 'f' * 32 + '-1'), ('type', 'dataset'), ('version', True)):
            files, _ = fixture()
            prefix = 'witness-partial-storage/normal/'
            raw = files[prefix + 'original-001.wire'][0]
            size = int.from_bytes(raw[:4], 'big')
            first = json.loads(raw[4:4 + size])
            first[field] = value
            changed = frame(first) + raw[4 + size:]
            files[prefix + 'original-001.wire'] = (changed, 0o400)

            def rebind(r):
                item = r['Originals'][0]
                item['SHA256'] = sha(changed)
                item['Object']['size'] = len(changed)
                item['Ref'].update(raw_sha256=sha(changed), raw_bytes=len(changed), forward_sha256=sha(changed))
                item['Ref']['forward'].update(sha256=sha(changed), bytes=len(changed))
                r['StoreBytes'] = len(changed)
                r['Boundary']['AcknowledgedBytes'] = len(changed)

            mutate_record(files, prefix + 'receipt.json', rebind)
            with self.subTest(field=field), self.assertRaises(ValueError):
                self.validate(files)

    def test_malformed_frames_duplicate_json_and_depth_fail(self):
        for data in (b'\x00\x01\x00\x01', b'\x00\x00\x00\x00', frame({}) + b'x',
                     b'\x00\x00\x00\x0d' + b'{"a":1,"a":2}', frame({'x': float('nan')}),
                     (len(b'[' * 40 + b'0' + b']' * 40)).to_bytes(4, 'big') + b'[' * 40 + b'0' + b']' * 40):
            files, _ = fixture()
            files['witness-partial-storage/normal/receipt.json'] = (data, 0o400)
            with self.subTest(data=data[:20]), self.assertRaises(ValueError):
                self.validate(files)

    def test_missing_extra_duplicate_unsafe_and_link_members_fail(self):
        files, _ = fixture()
        del files['witness-partial-storage/normal/cleanup.json']
        with self.assertRaises(ValueError):
            self.validate(files)
        for name in ('../outside', '/absolute', './alias', 'a//b', 'a\\b',
                     'witness-partial-storage/normal/receipt.json', 'unknown'):
            member = tarfile.TarInfo(name)
            member.size = 1
            with self.subTest(name=name), self.assertRaises(ValueError):
                self.validate(extras=[(member, b'x')])
        for kind in (tarfile.SYMTYPE, tarfile.LNKTYPE, tarfile.FIFOTYPE):
            member = tarfile.TarInfo('link')
            member.type, member.linkname = kind, '../outside'
            with self.subTest(kind=kind), self.assertRaises(ValueError):
                self.validate(extras=[(member, b'')])

    def test_modes_and_compression_cannot_hide_original_custody(self):
        files, _ = fixture()
        name = 'witness-partial-storage/normal/original-001.wire'
        files[name] = (files[name][0], 0o600)
        with self.assertRaises(ValueError):
            self.validate(files)
        with self.assertRaises(ValueError):
            self.validate(compressed=True)

    def test_archive_and_expectation_symlinks_are_refused(self):
        files, context = fixture()
        payload = archive(files)
        context['archive_sha256'] = sha(payload)
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / 'original'
            path.write_bytes(payload)
            link = Path(td) / 'link'
            link.symlink_to(path)
            with self.assertRaises(ValueError):
                E.validate_archive(link, context)
            expected = Path(td) / 'expected.json'
            expected.write_text(json.dumps(context))
            expected_link = Path(td) / 'expected-link.json'
            expected_link.symlink_to(expected)
            result = subprocess.run([sys.executable, str(ROOT / 'tools/witness-storage-evidence.py'),
                                     str(path), str(expected_link)], capture_output=True, check=False)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, b'')

    def test_headers_are_bounded_before_extended_payload_reads(self):
        for kind, size in ((tarfile.REGTYPE, 1 << 40), (tarfile.XHDTYPE, 100),
                           (tarfile.GNUTYPE_LONGNAME, 100), (tarfile.GNUTYPE_SPARSE, 0)):
            m = tarfile.TarInfo('unsupported')
            m.type, m.size = kind, size
            with self.subTest(kind=kind, size=size), self.assertRaises(ValueError):
                E.StorageTarInfo.frombuf(m.tobuf(format=tarfile.GNU_FORMAT)[:512], 'utf-8', 'strict')

    def test_no_extraction_and_archive_size_and_member_count_bounds(self):
        with mock.patch.object(tarfile.TarFile, 'extractall', side_effect=AssertionError('extraction forbidden')), \
                mock.patch.object(tarfile.TarFile, 'extract', side_effect=AssertionError('extraction forbidden')):
            self.validate()
        files, context = fixture()
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            large = root / 'large.tar'
            with large.open('wb') as f:
                f.truncate(E.ARCHIVE_CAP + 1)
            context['archive_sha256'] = sha(b'')
            with self.assertRaises(ValueError):
                E.validate_archive(large, context)
        extras = []
        for i in range(130):
            m = tarfile.TarInfo(f'extra-{i}')
            extras.append((m, b''))
        with self.assertRaises(ValueError):
            self.validate(extras=extras)

    def test_replaced_archive_name_fails_even_when_replacement_bytes_match(self):
        files, context = fixture()
        payload = archive(files)
        context['archive_sha256'] = sha(payload)
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / 'originals.tar'
            path.write_bytes(payload)
            original = E.read_case
            replaced = False

            def replace_then_read(*args):
                nonlocal replaced
                if not replaced:
                    replaced = True
                    path.rename(path.with_name('held-original.tar'))
                    path.write_bytes(payload)
                return original(*args)

            with mock.patch.object(E, 'read_case', side_effect=replace_then_read), self.assertRaises(ValueError):
                E.validate_archive(path, context)

    def test_cli_reports_scoped_success_and_no_output_on_failure(self):
        files, context = fixture()
        payload = archive(files)
        context['archive_sha256'] = sha(payload)
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / 'originals.tar').write_bytes(payload)
            (root / 'expected.json').write_text(json.dumps(context))
            command = [sys.executable, str(ROOT / 'tools/witness-storage-evidence.py'),
                       str(root / 'originals.tar'), str(root / 'expected.json')]
            result = subprocess.run(command, capture_output=True, check=False)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertFalse(json.loads(result.stdout)['native_acceptance'])
            context['archive_sha256'] = '0' * 64
            (root / 'expected.json').write_text(json.dumps(context))
            result = subprocess.run(command, capture_output=True, check=False)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, b'')


if __name__ == '__main__':
    unittest.main()
