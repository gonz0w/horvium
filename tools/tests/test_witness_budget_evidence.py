# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic byte-budget archives are portable controls, never native evidence."""
import base64
import copy
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('budget_evidence', ROOT / 'tools/witness-budget-evidence.py')
E = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(E)
RAW = 67508864
CAP = 268435456
ID = '0123456789abcdef0123456789abcdef-1'


def sha(b):
    return hashlib.sha256(b).hexdigest()


def frame(v):
    b = json.dumps(v, separators=(',', ':')).encode()
    return len(b).to_bytes(4, 'big') + b


def zeros_sha(n):
    h = hashlib.sha256()
    while n:
        size = min(n, 65536)
        h.update(bytes(size))
        n -= size
    return h.hexdigest()


def actor(pid, uid=0, digest='a' * 64):
    return dict(pid=pid, uid=uid, sec=1000, usec=10, executable='/owned/program', sha256=digest)


def incarnation(a):
    return f"{a['pid']}:{a['sec']}:{a['usec']}:{a['sha256']}"


def obj(n, inode=1, gid=0):
    return dict(device=1, inode=inode, uid=0, gid=gid, links=1, size=n, modified_ns=123)


def snapshot():
    return dict(ok=True, failed=False, failure='', store_failed=False, denied=0, originals=None,
                observer_high_rss=100, observer_cpu_ns=100, connector_started=0, connector_active=0,
                connector_max_rss=0, connector_cpu_ns=0, connectors=0, front_removed=False, active=False)


def fixture(profile):
    prefix = 'witness-' + ('raw' if profile == 'raw-limit' else 'archive')
    golden = b''.join(frame(dict(version=1, request_id=ID, type=k, data=v)) for k, v in
                      [('begin', {'synthetic': 'x' * 160}), ('dataset', {}), ('dataset', {}),
                       ('end', dict(records=4, complete=True))])
    files = {}
    for label in ('normal', profile):
        fault = label != 'normal'
        identity = 'f' * 32 + '-1' if fault else ID
        actors = dict(Owner=actor(101), Observer=actor(202 if fault else 102), Management=actor(203 if fault else 103, 65534, 'b' * 64),
                      Helper=actor(204 if fault else 104, digest='a' * 64 if fault and profile == 'raw-limit' else 'b' * 64))
        r = dict(Label=label, Mode='native-small', Publication='' if fault else identity, **actors,
                 Before=snapshot(), Boundary=snapshot(), Followup=snapshot(), Final=snapshot(),
                 RefreshStatus=202 if fault else 0, OriginalDenied=fault, LastCollectionFailed=fault,
                 Reaped=True, HelperUnchanged=True, ConfigurationUnchanged=True, Originals=[], Indexes=[],
                 PriorBytes=len(golden) if fault else 0,
                 ConfiguredPriorBytes=(len(golden) if profile == 'raw-limit' else CAP - 80) if fault else 0,
                 FinalAggregateBytes=(len(golden) + RAW if profile == 'raw-limit' else CAP) if fault else len(golden),
                 FailureStatus=base64.b64encode(b'Last collection: failed. Cached data is stale.').decode() if fault else None,
                 UnavailableBody=base64.b64encode(b'{"code":"unavailable","message":"Inventory unavailable"}').decode() if fault else None,
                 UnavailableStatus=503 if fault else 0)
        raw = RAW if fault and profile == 'raw-limit' else (golden[:80] if fault else golden)
        size = raw if type(raw) is int else len(raw)
        digest = zeros_sha(raw) if type(raw) is int else sha(raw)
        ref = dict(name='original-001.wire', request_id=identity, mode='native-small',
                   management_incarnation=incarnation(actors['Management']), helper_incarnation=incarnation(actors['Helper']),
                   raw_sha256=digest, raw_bytes=size, candidate_complete=not fault, delivered=not fault,
                   forward_sha256='' if fault else digest)
        if fault:
            ref['capture_failure'] = profile
        else:
            ref['forward'] = dict(stage='delivered', bytes=size, sha256=digest, error={'class': ''},
                                  write_half_closed=True, elapsed_ns=100)
        original = dict(ref=ref, object=obj(size))
        r['Originals'] = [dict(Name=ref['name'], SHA256=digest, Object=original['object'], Mode=0o400)]
        r['Final'].update(originals=[original], connector_started=1, connectors=1, connector_eof_checks=1,
                          failed=fault, store_failed=fault, failure='producer-capture' if fault else '', front_removed=True)
        boundary = dict(ID=identity, Management=ref['management_incarnation'], Helper=ref['helper_incarnation'],
                        Stage='copy' if fault else 'sealed', AcknowledgedBytes=size, Error={'class': ''})
        r['Final']['capture_boundary'] = boundary
        if not fault:
            r['Final']['forward_boundary'] = dict(ID=identity, Management=ref['management_incarnation'], Helper=ref['helper_incarnation'],
                StartedNS=1000, DeadlineNS=45000001000, CapturedElapsedNS=100, ForwardStartedElapsedNS=200,
                FinishedElapsedNS=300, DeadlineReached=False, ContextDeadlineExceeded=False, ContextCancelled=False,
                Forward=ref['forward'])
        r['Boundary'] = copy.deepcopy(r['Final'])
        r['Boundary']['front_removed'] = not fault
        r['Followup'] = copy.deepcopy(r['Boundary'])
        if fault:
            r['Followup']['denied'] = r['Final']['denied'] = 2
        out = prefix + '-cases/' + label + '/'
        files[out + ref['name']] = (raw, 0o400)
        if not fault:
            item = dict(Name=ref['name'] + '.json', SHA256=sha(frame(original)), Object=obj(len(frame(original)), 2), Mode=0o400)
            r['Indexes'] = [item]
            files[out + item['Name']] = (frame(original), 0o400)
        if profile == 'raw-limit':
            r['Source'] = None
            if fault:
                connector = dict(Actor=actor(205, 65534), Stage='connector-wait', Action='raw-limit',
                                 Present=True, Exited=True, ExitCode=0, Signal=0, Status=0, Accounted=True,
                                 ChannelClosed=False, WaitFailed=False, ContextEnded=False)
                for k in ('Boundary', 'Followup', 'Final'):
                    r[k]['connector_boundary'] = copy.deepcopy(connector)
                source = dict(DataClass='synthetic-oversized-producer-control', Actor=actors['Helper'], Socket=obj(0, 5, 65534),
                              PeerUID=65534, PeerPID=205,
                              Request=base64.b64encode(frame(dict(version=1, request_id=identity, operation='inventory'))).decode(),
                              RequestID=identity, StartedNS=1000, DeadlineNS=45000001000, FinishedNS=10000,
                              SentBytes=RAW + 1, SentSHA256=zeros_sha(RAW + 1), WriteError={'class': ''}, CloseWriteError={'class': ''})
                r['Source'] = source
                files['witness-raw-source.json'] = (frame(source), 0o400)
        files[out + 'receipt.json'] = (frame(r), 0o400)
        files['witness-observer-' + label + '.log'] = (b'PASS\n', 0o600)
    if profile == 'archive-limit':
        total, originals = 0, []
        while total < CAP - len(golden) - 80:
            n = min(RAW, CAP - len(golden) - 80 - total)
            name = f'original-{len(originals)+1:03d}.wire'
            digest = zeros_sha(n)
            ref = dict(name=name, request_id=ID.rsplit('-', 1)[0] + '-' + str(len(originals)+1), mode='native-small',
                       management_incarnation='archive-prefill-owner', helper_incarnation='synthetic-zero-stream',
                       raw_sha256=digest, raw_bytes=n, candidate_complete=False, delivered=False, forward_sha256='')
            originals.append(dict(Copy=dict(Name=name, SHA256=digest, Object=obj(n, len(originals)+10), Mode=0o400), Ref=ref))
            files['witness-archive-prefill/' + name] = (n, 0o400)
            total += n
        pressure = dict(DataClass='synthetic-zero-stream-pressure', Actor=actor(101), PriorBytes=len(golden),
                        CapturedBytes=total, AggregateBytes=CAP-80, RemainingBytes=80,
                        Originals=originals, Exported=True, MountRemoved=True)
        files['witness-archive-prefill/receipt.json'] = (frame(pressure), 0o400)
    probe = dict(abi={}, source_sha256=sha(b'C'), binary_sha256=sha(b'ELF'), compiler_sha256='c'*64,
                 compiler_version='synthetic\ncompiler', header_sha256={})
    files.update({'witness-probe-original.json': (frame(probe), 0o400),
                  'witness-probe/credential-probe.c': (b'C', 0o600), 'witness-probe/credential-probe': (b'ELF', 0o755),
                  prefix + '-cleanup.json': (frame(dict(Verified=True)), 0o400)})
    if profile == 'raw-limit':
        files.update({'witness-raw-abi.json': (frame(probe['abi']), 0o400), 'witness-raw-producer.log': (b'PASS\n', 0o600)})
    expected = dict(schema=1, profile=profile, source_commit='1'*40, base_commit='2'*40, pr_title='test: synthetic evidence',
                    release='15.1', source_archive_sha256='d'*64, compiler_sha256='e'*64,
                    fixture_sha256='a'*64, management_sha256='b'*64, probe_sha256=sha(frame(probe)), archive_sha256='')
    return files, expected


class Zeroes:
    def __init__(self, n):
        self.n = n

    def read(self, n):
        size = min(n, self.n)
        self.n -= size
        return bytes(size)


def write_archive(path, files, extras=()):
    with tarfile.open(path, 'w:') as a:
        for name, (data, mode) in files.items():
            m = tarfile.TarInfo(name)
            m.size, m.mode = data if type(data) is int else len(data), mode
            a.addfile(m, Zeroes(data) if type(data) is int else io.BytesIO(data))
        for m, data in extras:
            a.addfile(m, io.BytesIO(data))
    with path.open('rb') as f:
        return hashlib.file_digest(f, 'sha256').hexdigest()


def change(files, name, fn):
    value = json.loads(files[name][0][4:])
    fn(value)
    files[name] = (frame(value), files[name][1])


class BudgetEvidenceTests(unittest.TestCase):
    def validate(self, profile='raw-limit', edit=None, context=None, extras=()):
        files, expected = fixture(profile)
        if edit:
            edit(files)
        with tempfile.TemporaryDirectory() as td:
            path = Path(td)/'originals.tar'
            expected['archive_sha256'] = write_archive(path, files, extras)
            expected.update(context or {})
            return E.validate_archive(path, expected)

    def test_both_profiles_reopen_real_bounds_without_acceptance(self):
        for profile in ('raw-limit', 'archive-limit'):
            with self.subTest(profile=profile):
                r = self.validate(profile)
                self.assertEqual(r['profile'], profile)
                self.assertEqual(len(r['cases']), 2)
                self.assertEqual(r['cases'][1]['stored_bytes'], RAW if profile == 'raw-limit' else 80)
                self.assertEqual(r['pressure_originals'], 0 if profile == 'raw-limit' else 4)
                for field in ('qualified', 'native_acceptance', 'resource_acceptance', 'resource_activation'):
                    self.assertIs(r[field], False)

    def test_external_pins_and_profile_cannot_be_rebound(self):
        for key, value in [('archive_sha256','0'*64), ('fixture_sha256','0'*64), ('management_sha256','0'*64),
                           ('probe_sha256','0'*64), ('profile','other'), ('schema',True), ('extra',True)]:
            with self.subTest(key=key), self.assertRaises(ValueError):
                self.validate(context={key:value})

    def test_permanent_failure_and_wait_mutations_are_rejected(self):
        name='witness-raw-cases/raw-limit/receipt.json'
        mutations=[lambda r:r.update(OriginalDenied=False), lambda r:r.update(FinalAggregateBytes=CAP),
                   lambda r:r['Final'].update(failed=False), lambda r:r['Followup'].update(connector_started=2),
                   lambda r:r['Boundary']['capture_boundary'].update(AcknowledgedBytes=RAW-1),
                   lambda r:r['Final']['originals'][0]['ref'].update(delivered=True),
                   lambda r:r['Final']['connector_boundary'].update(WaitFailed=True),
                   lambda r:r['Final']['connector_boundary']['Actor'].update(pid=103),
                   lambda r:r.update(unknown=True)]
        for i,fn in enumerate(mutations):
            with self.subTest(i=i), self.assertRaises(ValueError):
                self.validate(edit=lambda f:change(f,name,fn))

    def test_source_copies_peer_and_bytes_must_agree(self):
        for fn in (lambda s:s.update(SentBytes=RAW), lambda s:s.update(SentSHA256='0'*64),
                   lambda s:s.update(PeerPID=103), lambda s:s['Actor'].update(sha256='b'*64),
                   lambda s:s.update(FinishedNS=s['DeadlineNS']), lambda s:s.update(DataClass='native-product-helper')):
            def edit(f):
                change(f,'witness-raw-source.json',fn)
                change(f,'witness-raw-cases/raw-limit/receipt.json',lambda r:fn(r['Source']))
            with self.subTest(fn=fn), self.assertRaises(ValueError):
                self.validate(edit=edit)

    def test_original_changed_even_when_archive_digest_is_rebound(self):
        def edit(f):
            name='witness-raw-cases/normal/original-001.wire'
            f[name]=(b'bad',0o400)
        with self.assertRaises(ValueError):
            self.validate(edit=edit)

    def test_archive_pressure_counts_and_ownership_are_required(self):
        for fn in (lambda r:r.update(CapturedBytes=0), lambda r:r.update(RemainingBytes=81),
                   lambda r:r.update(MountRemoved=False), lambda r:r['Originals'][0]['Copy']['Object'].update(links=2)):
            with self.subTest(fn=fn), self.assertRaises(ValueError):
                self.validate('archive-limit',edit=lambda f:change(f,'witness-archive-prefill/receipt.json',fn))

    def test_missing_wrong_mode_extra_and_namespace_mixing_fail(self):
        for edit in (lambda f:f.pop('witness-raw-source.json'),
                     lambda f:f.update({'witness-raw-source.json':(f['witness-raw-source.json'][0],0o600)}),
                     lambda f:f.update({'witness-archive-cleanup.json':(frame({'Verified':True}),0o400)}),
                     lambda f:f.update({'witness-raw-abi.json':(frame({'other':1}),0o400)})):
            with self.subTest(edit=edit), self.assertRaises(ValueError):
                self.validate(edit=edit)

    def test_unsafe_headers_fail_before_payload_reads(self):
        for kind,size in ((tarfile.REGTYPE,RAW+1),(tarfile.XHDTYPE,100),(tarfile.GNUTYPE_SPARSE,0),
                          (tarfile.GNUTYPE_LONGNAME,100),(tarfile.SYMTYPE,0)):
            m=tarfile.TarInfo('witness-raw-cases/raw-limit/original-001.wire');m.type,m.size=kind,size
            with self.subTest(kind=kind), self.assertRaises(ValueError):
                E.BudgetTarInfo.frombuf(m.tobuf(format=tarfile.GNU_FORMAT)[:512],'utf-8','strict')

    def test_no_extraction_and_input_object_replacement(self):
        with mock.patch.object(tarfile.TarFile,'extractall',side_effect=AssertionError('no extraction')), \
                mock.patch.object(tarfile.TarFile,'extract',side_effect=AssertionError('no extraction')):
            self.validate()
        files,expected=fixture('raw-limit')
        with tempfile.TemporaryDirectory() as td:
            path=Path(td)/'originals.tar';expected['archive_sha256']=write_archive(path,files)
            original=E.read_case
            def replace(*args):
                path.rename(path.with_name('held.tar'));write_archive(path,files)
                return original(*args)
            with mock.patch.object(E,'read_case',side_effect=replace),self.assertRaises(ValueError):
                E.validate_archive(path,expected)

    def test_rebound_incomplete_archive_prefix_still_needs_a_real_header(self):
        for header in (0, 76, 65537):
            def edit(f):
                name = 'witness-archive-cases/archive-limit/original-001.wire'
                data = header.to_bytes(4, 'big') + bytes(76)
                f[name] = (data, 0o400)
                def rebind(r):
                    r['Originals'][0]['SHA256'] = sha(data)
                    for k in ('Boundary', 'Followup', 'Final'):
                        r[k]['originals'][0]['ref']['raw_sha256'] = sha(data)
                change(f, 'witness-archive-cases/archive-limit/receipt.json', rebind)
            with self.subTest(header=header), self.assertRaises(ValueError):
                self.validate('archive-limit', edit=edit)

    def test_jointly_rebound_cross_manager_replay_is_rejected(self):
        def edit(f):
            normal = json.loads(f['witness-raw-cases/normal/receipt.json'][0][4:])
            management = normal['Management']
            def rebind(r):
                r['Management'] = management
                for k in ('Boundary', 'Followup', 'Final'):
                    r[k]['capture_boundary']['Management'] = incarnation(management)
                    for original in r[k]['originals']:
                        original['ref']['management_incarnation'] = incarnation(management)
            change(f, 'witness-raw-cases/raw-limit/receipt.json', rebind)
        with self.assertRaises(ValueError):
            self.validate(edit=edit)

    def test_trailing_data_compression_and_archive_symlinks_fail(self):
        files, expected = fixture('raw-limit')
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / 'originals.tar'
            expected['archive_sha256'] = write_archive(path, files)
            alias = path.with_name('alias')
            alias.symlink_to(path)
            with self.assertRaises(ValueError):
                E.validate_archive(alias, expected)
            with path.open('ab') as f:
                f.write(b'not padding')
            with path.open('rb') as f:
                expected['archive_sha256'] = hashlib.file_digest(f, 'sha256').hexdigest()
            with self.assertRaises(ValueError):
                E.validate_archive(path, expected)
            with tarfile.open(path, 'w:gz') as f:
                pass
            expected['archive_sha256'] = sha(path.read_bytes())
            with self.assertRaises(ValueError):
                E.validate_archive(path, expected)

    def test_cli_success_and_failure_have_scoped_output(self):
        files,expected=fixture('raw-limit')
        with tempfile.TemporaryDirectory() as td:
            path=Path(td)/'originals.tar';expected['archive_sha256']=write_archive(path,files)
            context=Path(td)/'expectation.json';context.write_text(json.dumps(expected))
            argv=[sys.executable,str(ROOT/'tools/witness-budget-evidence.py'),str(path),str(context)]
            result=subprocess.run(argv,capture_output=True,check=False)
            self.assertEqual(result.returncode,0,result.stderr)
            self.assertFalse(json.loads(result.stdout)['native_acceptance'])
            expected['archive_sha256']='0'*64;context.write_text(json.dumps(expected))
            result=subprocess.run(argv,capture_output=True,check=False)
            self.assertNotEqual(result.returncode,0);self.assertEqual(result.stdout,b'')
            context.unlink();context.symlink_to(path)
            result=subprocess.run(argv,capture_output=True,check=False)
            self.assertNotEqual(result.returncode,0);self.assertEqual(result.stdout,b'')


if __name__=='__main__':
    unittest.main()
