#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Offline consistency checks for retained broker raw/archive byte-budget controls.

Independent host expectations are mandatory. No extraction, launch, network,
qualification authority, full wire/numeric oracle or newer-source native claim.
"""
import argparse
import base64
import hashlib
import importlib.util
import json
from pathlib import Path, PurePosixPath
import re
import sys
import tarfile

SPEC = importlib.util.spec_from_file_location('witness_storage_contract', Path(__file__).with_name('witness-storage-evidence.py'))
E = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(E)
C = E.C
RAW_CAP = 67508864
ARCHIVE_CAP = 268435456
PROFILES = ('raw-limit', 'archive-limit')
CASE_FIELDS = ('FailureStatus', 'UnavailableBody', 'UnavailableStatus', 'Label', 'Mode', 'Publication',
               'Owner', 'Observer', 'Management', 'Helper', 'Before', 'Boundary', 'Followup', 'Final',
               'RefreshStatus', 'OriginalDenied', 'LastCollectionFailed', 'Reaped', 'HelperUnchanged',
               'ConfigurationUnchanged', 'Originals', 'Indexes', 'PriorBytes', 'ConfiguredPriorBytes', 'FinalAggregateBytes')
SNAPSHOT_FIELDS = ('ok', 'failed', 'failure', 'store_failed', 'denied', 'originals', 'observer_high_rss',
                   'observer_cpu_ns', 'connector_started', 'connector_active', 'connector_max_rss',
                   'connector_cpu_ns', 'connectors', 'front_removed', 'active')
SNAPSHOT_OPTIONAL = ('capture_boundary', 'forward_boundary', 'connector_eof_checks', 'connector_eof_rejected', 'connector_boundary')


class BudgetTarInfo(tarfile.TarInfo):
    @classmethod
    def frombuf(cls, buf, encoding, errors):
        m = super().frombuf(buf, encoding, errors)
        wire = re.fullmatch(r'witness-(?:raw|archive)-cases/(?:normal|raw-limit|archive-limit)/original-[0-9]{3}\.wire', m.name)
        pressure = re.fullmatch(r'witness-archive-prefill/original-[0-9]{3}\.wire', m.name)
        cap = RAW_CAP if wire or pressure else (E.FRAME_CAP + 4 if m.name.endswith('.json') else 1 << 20)
        C.require(m.type in (tarfile.REGTYPE, tarfile.AREGTYPE, tarfile.DIRTYPE) and 0 <= m.size <= cap,
                  'unsupported header or member bound')
        return m


def expectation(value):
    E.keys(value, ('profile', 'management_sha256', 'schema', 'source_commit', 'base_commit', 'pr_title',
                   'release', 'source_archive_sha256', 'compiler_sha256', 'fixture_sha256', 'probe_sha256', 'archive_sha256'))
    C.require(value['profile'] in PROFILES, 'unsupported budget profile')
    C.digest(value['management_sha256'])
    E.expectation({k: v for k, v in value.items() if k not in ('profile', 'management_sha256')})


def actor(value, digest, uid=0):
    E.actor({**value, 'uid': 0}, digest)
    E.exact(value['uid'], uid)


def incarnation(a):
    return f"{a['pid']}:{a['sec']}:{a['usec']}:{a['sha256']}"


def request_id(value):
    C.require(type(value) is str and re.fullmatch(r'[0-9a-f]{32}-[1-9][0-9]*', value) is not None, 'original request ID')
    C.integer(int(value.rsplit('-', 1)[1]), 1, (1 << 64) - 1)


def frames(data, identity):
    offset, rows = 0, []
    while offset < len(data):
        C.require(offset + 4 <= len(data), 'original frame header')
        size = int.from_bytes(data[offset:offset+4], 'big')
        C.require(0 < size <= E.FRAME_CAP and offset + 4 + size <= len(data), 'bounded complete original frame')
        row = E.bounded_json(data[offset+4:offset+4+size])
        E.keys(row, ('version', 'request_id', 'type', 'data'))
        E.exact(row['version'], 1)
        E.exact(row['request_id'], identity)
        rows.append(row)
        C.require(len(rows) <= 100000, 'record bound')
        offset += size + 4
    C.require(len(rows) >= 4 and rows[0]['type'] == 'begin' and rows[-1]['type'] == 'end'
              and all(x['type'] in ('class', 'geom', 'provider', 'consumer', 'pool', 'dataset') for x in rows[1:-1])
              and sum(x['type'] == 'dataset' for x in rows) >= 2, 'normal native-small frame order')
    E.exact(rows[-1]['data'], {'records': len(rows), 'complete': True})


def snapshot(r, fault):
    E.keys(r, SNAPSHOT_FIELDS, SNAPSHOT_OPTIONAL)
    for k, v in (('ok', True), ('failed', fault), ('store_failed', fault), ('active', False),
                 ('failure', 'producer-capture' if fault else ''), ('connector_active', 0)):
        E.exact(r[k], v)
    for k in ('denied', 'observer_high_rss', 'observer_cpu_ns', 'connector_started', 'connector_max_rss',
              'connector_cpu_ns', 'connectors', 'connector_eof_checks', 'connector_eof_rejected'):
        C.integer(r.get(k, 0))
    C.integer(r['connector_started'], 0, 32)
    E.exact(r['connector_started'], r['connectors'])
    E.exact(r.get('connector_eof_rejected', 0), 0)
    C.require(type(r['front_removed']) is bool, 'socket removal type')


def boundary(r, ref, fault):
    E.exact(r, {'ID': ref['request_id'], 'Management': ref['management_incarnation'],
                'Helper': ref['helper_incarnation'], 'Stage': 'copy' if fault else 'sealed',
                'AcknowledgedBytes': ref['raw_bytes'], 'Error': {'class': ''}})


def copied(item, read, prefix, size):
    E.keys(item, ('Name', 'SHA256', 'Object', 'Mode'))
    C.require(type(item['Name']) is str and re.fullmatch(r'original-[0-9]{3}\.wire(?:\.json)?', item['Name']) is not None,
              'original slot name')
    E.exact(item['Mode'], 0o400)
    data = read(prefix + item['Name'])
    E.file_object(item['Object'], len(data))
    E.exact(len(data), size)
    E.exact(E.sha(data), item['SHA256'])
    return data


def read_case(label, read, expected):
    fault = label != 'normal'
    raw_profile = expected['profile'] == 'raw-limit'
    prefix = 'witness-' + ('raw' if raw_profile else 'archive') + '-cases/' + label + '/'
    receipt = read(prefix + 'receipt.json')
    r = E.framed(receipt)
    E.keys(r, CASE_FIELDS + (('Source',) if raw_profile else ()))
    for k, v in (('Label', label), ('Mode', 'native-small'), ('Reaped', True), ('HelperUnchanged', True), ('ConfigurationUnchanged', True)):
        E.exact(r[k], v)
    for k in ('PriorBytes', 'ConfiguredPriorBytes', 'FinalAggregateBytes'):
        C.integer(r[k], 0, ARCHIVE_CAP)
    uid = r['Management']['uid']
    C.integer(uid, 1, (1 << 32) - 2)
    for k in ('Owner', 'Observer'):
        actor(r[k], expected['fixture_sha256'])
    actor(r['Management'], expected['management_sha256'], uid)
    actor(r['Helper'], expected['fixture_sha256'] if raw_profile and fault else expected['management_sha256'])
    C.require(len({r[k]['pid'] for k in ('Owner', 'Observer', 'Management', 'Helper')}) == 4, 'distinct original actors')
    for k in ('Before', 'Boundary', 'Followup', 'Final'):
        snapshot(r[k], fault and k != 'Before')
    before, hit, follow, final = (r[k] for k in ('Before', 'Boundary', 'Followup', 'Final'))
    for k in ('connector_started', 'connectors', 'connector_active'):
        E.exact(before[k], 0)
    E.exact(before['originals'], None)
    E.exact(before['front_removed'], False)
    E.exact(final['front_removed'], True)
    originals = final['originals']
    C.require(type(originals) is list and 1 <= len(originals) <= 32 and type(r['Originals']) is list,
              'complete bounded original list')
    E.exact(len(r['Originals']), len(originals))
    E.exact(final['connector_started'], len(originals))
    E.exact(final.get('connector_eof_checks', 0), len(originals))
    total, ids = 0, set()
    for i, (original, item) in enumerate(zip(originals, r['Originals']), 1):
        E.keys(original, ('ref', 'object'))
        ref = original['ref']
        E.keys(ref, ('name', 'request_id', 'mode', 'management_incarnation', 'helper_incarnation', 'raw_sha256',
                     'raw_bytes', 'candidate_complete', 'delivered', 'forward_sha256'), ('capture_failure', 'forward'))
        request_id(ref['request_id'])
        C.require(ref['request_id'] not in ids, 'duplicate original request')
        ids.add(ref['request_id'])
        for k, v in (('name', f'original-{i:03d}.wire'), ('mode', 'native-small'),
                     ('management_incarnation', incarnation(r['Management'])), ('helper_incarnation', incarnation(r['Helper'])),
                     ('candidate_complete', not fault), ('delivered', not fault)):
            E.exact(ref[k], v)
        C.integer(ref['raw_bytes'], 1, RAW_CAP)
        E.exact(item['Name'], ref['name'])
        E.exact(item['Object'], original['object'])
        data = copied(item, read, prefix, ref['raw_bytes'])
        E.exact(item['SHA256'], ref['raw_sha256'])
        if fault:
            E.exact(ref.get('capture_failure'), expected['profile'])
            E.exact(ref['forward_sha256'], '')
            C.require('forward' not in ref, 'failed capture was forwarded')
            E.exact(ref['raw_bytes'], RAW_CAP if raw_profile else 80)
            if raw_profile:
                C.require(not any(data), 'malformed zero-stream control changed')
            else:
                C.require(len(data) - 4 < int.from_bytes(data[:4], 'big') <= E.FRAME_CAP,
                          'archive prefix is not an incomplete bounded frame')
        else:
            C.require('capture_failure' not in ref, 'normal capture failed')
            E.exact(ref['forward_sha256'], ref['raw_sha256'])
            forward = ref['forward']
            E.keys(forward, ('stage', 'bytes', 'sha256', 'error', 'write_half_closed', 'elapsed_ns'))
            for k, v in (('stage', 'delivered'), ('bytes', len(data)), ('sha256', ref['raw_sha256']),
                         ('error', {'class': ''}), ('write_half_closed', True)):
                E.exact(forward[k], v)
            C.integer(forward['elapsed_ns'], 0, 44999999999)
            frames(data, ref['request_id'])
        total += len(data)
    indexes = r['Indexes']
    C.require(indexes is None or type(indexes) is list, 'index list type')
    E.exact(len(indexes or []), 0 if fault else len(originals))
    for original, item in zip(originals, indexes or []):
        E.exact(item['Name'], original['ref']['name'] + '.json')
        data = copied(item, read, prefix, item['Object']['size'])
        E.exact(E.framed(data), original)
    boundary(final['capture_boundary'], originals[-1]['ref'], fault)
    if fault:
        E.exact(len(originals), 1)
        for snap in (hit, follow, final):
            E.exact(snap['originals'], originals)
            E.exact(snap['capture_boundary'], final['capture_boundary'])
            E.exact(snap['connector_started'], 1)
            C.require('forward_boundary' not in snap, 'failed capture has forward boundary')
            if not raw_profile:
                C.require('connector_boundary' not in snap, 'unexpected fault connector diagnostic')
        C.require(follow['denied'] > hit['denied'] and final['denied'] >= follow['denied'], 'permanent denial missing')
        for k in ('OriginalDenied', 'LastCollectionFailed'):
            E.exact(r[k], True)
        C.require(type(r['RefreshStatus']) is int and r['RefreshStatus'] in (202, 409), 'actual refresh or busy join')
        E.exact(r['Publication'], '')
        body = base64.b64decode(r['FailureStatus'], validate=True)
        unavailable = base64.b64decode(r['UnavailableBody'], validate=True)
        C.require(0 < len(body) <= E.FRAME_CAP and b'Last collection:' in body and b'Cached data is stale.' in body
                  and b'Collecting an observation' not in body, 'failure response missing')
        E.exact(r['UnavailableStatus'], 503)
        E.exact(E.bounded_json(unavailable), {'code': 'unavailable', 'message': 'Inventory unavailable'})
    else:
        C.require(r['Publication'] in ids, 'recorded normal publication missing')
        for k in ('OriginalDenied', 'LastCollectionFailed'):
            E.exact(r[k], False)
        for k in ('RefreshStatus', 'UnavailableStatus', 'PriorBytes', 'ConfiguredPriorBytes'):
            E.exact(r[k], 0)
        for k in ('FailureStatus', 'UnavailableBody'):
            E.exact(r[k], None)
        E.exact(r['FinalAggregateBytes'], total)
        E.exact(hit, final)
        E.exact(follow, final)
        fw = final['forward_boundary']
        E.keys(fw, ('ID', 'Management', 'Helper', 'StartedNS', 'DeadlineNS', 'CapturedElapsedNS', 'ForwardStartedElapsedNS',
                    'FinishedElapsedNS', 'DeadlineReached', 'ContextDeadlineExceeded', 'ContextCancelled', 'Forward'))
        last = originals[-1]['ref']
        for k, v in (('ID', last['request_id']), ('Management', last['management_incarnation']),
                     ('Helper', last['helper_incarnation']), ('Forward', last['forward']), ('DeadlineReached', False),
                     ('ContextDeadlineExceeded', False), ('ContextCancelled', False)):
            E.exact(fw[k], v)
        for k in ('StartedNS', 'DeadlineNS', 'CapturedElapsedNS', 'ForwardStartedElapsedNS', 'FinishedElapsedNS'):
            C.integer(fw[k])
        E.exact(fw['DeadlineNS'] - fw['StartedNS'], 45000000000)
        C.require(fw['CapturedElapsedNS'] <= fw['ForwardStartedElapsedNS'] <= fw['FinishedElapsedNS'] < 45000000000,
                  'normal forward deadline')
    if raw_profile and not fault:
        E.exact(r['Source'], None)
    return r, {'case': label, 'receipt_sha256': E.sha(receipt), 'originals': len(originals), 'stored_bytes': total,
               'original_indexes': len(indexes or []), 'publication_recorded': not fault, 'permanent_denial_recorded': fault,
               'data_class': 'synthetic-oversized-producer-control' if fault and raw_profile else 'native-product-helper'}


def raw_source(r, read, expected, abi):
    data = read('witness-raw-source.json')
    source = E.framed(data)
    E.exact(source, r['Source'])
    E.keys(source, ('DataClass', 'Actor', 'Socket', 'PeerUID', 'PeerPID', 'Request', 'RequestID', 'StartedNS',
                    'DeadlineNS', 'FinishedNS', 'SentBytes', 'SentSHA256', 'WriteError', 'CloseWriteError'))
    E.exact(source['DataClass'], 'synthetic-oversized-producer-control')
    E.exact(source['Actor'], r['Helper'])
    E.keys(source['Socket'], ('device', 'inode', 'uid', 'gid', 'links', 'size', 'modified_ns'))
    for value in source['Socket'].values():
        C.integer(value)
    E.exact(source['Socket']['uid'], 0)
    E.exact(source['Socket']['links'], 1)
    C.integer(source['Socket']['inode'], 1)
    connector = r['Boundary']['connector_boundary']
    E.keys(connector, ('Actor', 'Stage', 'Action', 'Present', 'Exited', 'ExitCode', 'Signal', 'Status', 'Accounted',
                       'ChannelClosed', 'WaitFailed', 'ContextEnded'))
    actor(connector['Actor'], expected['fixture_sha256'], r['Management']['uid'])
    C.require(connector['Actor']['pid'] not in {r[k]['pid'] for k in ('Owner', 'Observer', 'Management', 'Helper')},
              'connector is not an original independent actor')
    for k, v in (('Stage', 'connector-wait'), ('Action', 'raw-limit'), ('Present', True), ('Exited', True),
                 ('ExitCode', 0), ('Signal', 0), ('Status', 0), ('Accounted', True), ('ChannelClosed', False),
                 ('WaitFailed', False), ('ContextEnded', False)):
        E.exact(connector[k], v)
    for k in ('Followup', 'Final'):
        E.exact(r[k]['connector_boundary'], connector)
    E.exact(source['PeerUID'], r['Management']['uid'])
    E.exact(source['PeerPID'], connector['Actor']['pid'])
    E.exact(source['RequestID'], r['Boundary']['capture_boundary']['ID'])
    request = base64.b64decode(source['Request'], validate=True)
    E.exact(E.framed(request), {'version': 1, 'request_id': source['RequestID'], 'operation': 'inventory'})
    for k in ('StartedNS', 'DeadlineNS', 'FinishedNS'):
        C.integer(source[k], 1)
    E.exact(source['DeadlineNS'] - source['StartedNS'], 45000000000)
    C.require(source['StartedNS'] < source['FinishedNS'] < source['DeadlineNS'], 'source deadline')
    E.exact(source['SentBytes'], RAW_CAP + 1)
    h = hashlib.sha256()
    remaining = RAW_CAP + 1
    while remaining:
        n = min(remaining, E.FRAME_CAP)
        h.update(bytes(n))
        remaining -= n
    E.exact(source['SentSHA256'], h.hexdigest())
    E.exact(source['WriteError'], {'class': ''})
    # These retained successful profiles recorded successful half-close too.
    E.exact(source['CloseWriteError'], {'class': ''})
    E.exact(E.framed(read('witness-raw-abi.json')), abi)
    E.exact(read('witness-raw-producer.log'), b'PASS\n')
    return {'receipt_sha256': E.sha(data), 'sent_bytes': RAW_CAP + 1, 'sent_sha256': h.hexdigest(),
            'request_sha256': E.sha(request), 'peer_matches_connector': True}


def pressure(read, expected, normal_bytes):
    payload = read('witness-archive-prefill/receipt.json')
    r = E.framed(payload)
    E.keys(r, ('DataClass', 'Actor', 'PriorBytes', 'CapturedBytes', 'AggregateBytes', 'RemainingBytes',
               'Originals', 'Exported', 'MountRemoved'))
    actor(r['Actor'], expected['fixture_sha256'])
    for k, v in (('DataClass', 'synthetic-zero-stream-pressure'), ('PriorBytes', normal_bytes),
                 ('CapturedBytes', ARCHIVE_CAP - normal_bytes - 80), ('AggregateBytes', ARCHIVE_CAP - 80),
                 ('RemainingBytes', 80), ('Exported', True), ('MountRemoved', True)):
        E.exact(r[k], v)
    C.require(type(r['Originals']) is list and len(r['Originals']) == 4, 'four actual pressure captures required')
    total, ids = 0, set()
    for i, item in enumerate(r['Originals'], 1):
        E.keys(item, ('Copy', 'Ref'))
        ref = item['Ref']
        E.keys(ref, ('name', 'request_id', 'mode', 'management_incarnation', 'helper_incarnation', 'raw_sha256', 'raw_bytes',
                     'candidate_complete', 'delivered', 'forward_sha256'))
        request_id(ref['request_id'])
        C.require(ref['request_id'] not in ids, 'duplicate pressure capture ID')
        ids.add(ref['request_id'])
        size = min(RAW_CAP, ARCHIVE_CAP - normal_bytes - 80 - total)
        for k, v in (('name', f'original-{i:03d}.wire'), ('mode', 'native-small'), ('management_incarnation', 'archive-prefill-owner'),
                     ('helper_incarnation', 'synthetic-zero-stream'), ('raw_bytes', size), ('candidate_complete', False),
                     ('delivered', False), ('forward_sha256', '')):
            E.exact(ref[k], v)
        E.exact(item['Copy']['Name'], ref['name'])
        data = copied(item['Copy'], read, 'witness-archive-prefill/', size)
        E.exact(E.sha(data), ref['raw_sha256'])
        C.require(not any(data), 'pressure bytes changed')
        total += size
    E.exact(total, r['CapturedBytes'])
    return {'receipt_sha256': E.sha(payload), 'originals': 4, 'stored_bytes': total,
            'data_class': 'synthetic-zero-stream-pressure'}


def validate_archive(path, expected):
    try:
        expectation(expected)
        f, before = E.opened(path, E.ARCHIVE_CAP)
        with f:
            digest, consumed = hashlib.sha256(), 0
            for b in E.chunks(f, before.st_size + 1):
                digest.update(b)
                consumed += len(b)
            E.exact(consumed, before.st_size)
            E.exact(digest.hexdigest(), expected['archive_sha256'])
            f.seek(0)
            with tarfile.open(fileobj=f, mode='r:', tarinfo=BudgetTarInfo) as archive:
                members, used = {}, set()
                for m in archive:
                    C.require(len(members) < 128 and m.name not in members and m.name == PurePosixPath(m.name).as_posix()
                              and not m.name.startswith('/') and '..' not in PurePosixPath(m.name).parts and '\\' not in m.name
                              and (m.isfile() or m.isdir()) and m.sparse is None and m.uid == m.gid == 0,
                              'unsafe archive member')
                    members[m.name] = m
                C.require(sum(m.size for m in members.values()) <= E.ARCHIVE_CAP, 'aggregate archive member bound')

                def read(name):
                    C.require(name in members and members[name].isfile() and name not in used, 'missing/repeated original')
                    m = members[name]
                    mode = 0o600 if name.endswith(('.log', 'credential-probe.c')) else (0o755 if name.endswith('/credential-probe') else 0o400)
                    E.exact(m.mode, mode)
                    used.add(name)
                    data = archive.extractfile(m).read(m.size + 1)
                    E.exact(len(data), m.size)
                    return data

                pb = read('witness-probe-original.json')
                E.exact(E.sha(pb), expected['probe_sha256'])
                probe = E.framed(pb)
                E.keys(probe, ('abi', 'source_sha256', 'binary_sha256', 'compiler_sha256', 'compiler_version', 'header_sha256'))
                E.exact(E.sha(read('witness-probe/credential-probe.c')), probe['source_sha256'])
                E.exact(E.sha(read('witness-probe/credential-probe')), probe['binary_sha256'])
                profile = expected['profile']
                prefix = 'witness-' + ('raw' if profile == 'raw-limit' else 'archive')
                normal, nr = read_case('normal', read, expected)
                negative, fr = read_case(profile, read, expected)
                E.exact(normal['Owner'], negative['Owner'])
                C.require(all(incarnation(normal[k]) != incarnation(negative[k])
                              for k in ('Management', 'Helper', 'Observer')), 'cross-pair actor replay')
                normal_ids = {v['ref']['request_id'] for v in normal['Final']['originals']}
                C.require(not normal_ids.intersection(v['ref']['request_id'] for v in negative['Final']['originals']),
                          'cross-pair original ID replay')
                E.exact(negative['PriorBytes'], nr['stored_bytes'])
                if profile == 'raw-limit':
                    E.exact(negative['ConfiguredPriorBytes'], nr['stored_bytes'])
                    E.exact(negative['FinalAggregateBytes'], nr['stored_bytes'] + RAW_CAP)
                    C.require(negative['FinalAggregateBytes'] < ARCHIVE_CAP, 'archive cap supplied raw failure')
                    source = raw_source(negative, read, expected, probe['abi'])
                    prefill = None
                else:
                    E.exact(negative['ConfiguredPriorBytes'], ARCHIVE_CAP - 80)
                    E.exact(negative['FinalAggregateBytes'], ARCHIVE_CAP)
                    prefill = pressure(read, expected, nr['stored_bytes'])
                    source = None
                E.exact(E.framed(read(prefix + '-cleanup.json')), {'Verified': True})
                for label in ('normal', profile):
                    read('witness-observer-' + label + '.log')
                allowed_dirs = {'witness-probe', prefix + '-cases', prefix + '-cases/normal', prefix + '-cases/' + profile}
                if prefill:
                    allowed_dirs.add('witness-archive-prefill')
                C.require(all((m.name in allowed_dirs and m.mode == 0o700 and m.size == 0) if m.isdir()
                              else m.name in used for m in members.values()), 'unrecognized or mixed evidence namespace')
                f.seek(archive.offset)
                padding = 0
                for b in E.chunks(f, before.st_size - archive.offset):
                    C.require(not any(b), 'nonzero trailing archive bytes')
                    padding += len(b)
                C.require(padding >= 1024, 'incomplete archive end marker')
            E.stable(path, f, before)
        return {**expected, 'scope': 'broker-byte-budget-consistency-readback', 'cases': [nr, fr],
                'source_control': source, 'synthetic_pressure': prefill, 'pressure_originals': 4 if prefill else 0,
                'actual_total_original_bytes': nr['stored_bytes'] + fr['stored_bytes'] + (prefill['stored_bytes'] if prefill else 0),
                'qualified': False, 'native_acceptance': False, 'resource_acceptance': False, 'resource_activation': False}
    except (OSError, tarfile.TarError, KeyError, TypeError, ValueError, OverflowError, RecursionError) as error:
        raise ValueError('broker budget archive or independent expectations rejected') from error


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('archive', type=Path)
    parser.add_argument('expectation', type=Path)
    args = parser.parse_args()
    try:
        f, before = E.opened(args.expectation, E.FRAME_CAP)
        with f:
            expected = E.bounded_json(f.read(E.FRAME_CAP + 1))
            E.stable(args.expectation, f, before)
        result = validate_archive(args.archive, expected)
    except (OSError, ValueError):
        print('Broker budget evidence rejected; no acceptance result produced.', file=sys.stderr)
        return 1
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
