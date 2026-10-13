#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Owned integration for the existing dashboard performance suite.

Developer-controlled source/receipts require maintainer review. No runtime service,
operator storage, guest package installation or external attestation is provided.
"""
import hashlib
import errno
import json
import os
import platform
import re
import shutil
import socket
import subprocess
import sys
import tempfile
import time
from pathlib import Path

import importlib.util


def load(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


preflight = load('dashboard-host-preflight')
build, runner = preflight.build, preflight.runner
c, owned = build.contract, build.review.OWNED
ROOT = Path(__file__).resolve().parents[1]
REMOTE = ('cd /root/work/cmd/horvium && env HORVIUM_NATIVE_VM=1 HORVIUM_PERF_VM=1 '
          '../../.tools/dashboard-native.test -test.v -test.count=1 -test.timeout=30m '
          "-test.run='^TestNative(PerformanceBudgets|ReaderResourceBudgets)$'")
ORIGINAL_NAMES = {name: 'dashboard-original-' + name + '.json' for name in (*c.MODES, *c.PARENTS)}


def sha(payload):
    return hashlib.sha256(payload).hexdigest()


def transfer_operation(name):
    c.require(name in ('source.tar', *build.ARTIFACT_FILES.values()), 'fixed original transfer name required')
    if name == 'source.tar':
        remote = 'umask 077; mkdir /root/work && tar -xf - -C /root/work && mkdir /root/work/.tools'
    else:
        remote = 'umask 077; cat > /root/work/.tools/' + name + ' && chmod 700 /root/work/.tools/' + name
    return ['transfer', name], remote


def assemble(raw, provenance, originals, parents):
    """Combine independently retained originals; never learn expectations from rows."""
    c.require(type(raw) is bytes and 0 < len(raw) <= c.CEILING and raw.endswith(b'\n'), 'complete original resource transcript required')
    c.object_keys(originals, c.MODES, 'original fixture sidecars')
    c.object_keys(parents, c.PARENTS, 'original parent sidecars')
    fixtures = {}
    for mode, row in originals.items():
        c.object_keys(row, ('schema', 'mode', 'fixture'), 'original fixture')
        c.require(type(row['schema']) is int and row['schema'] == 1 and row['mode'] == mode, 'original fixture identity mismatch')
        c.object_keys(row['fixture'], ('datasets', 'owned_datasets', 'inventory_sha256', 'origin'), 'original fixture summary')
        c.digest(row['fixture']['inventory_sha256'])
        c.integer(row['fixture']['datasets'], 1, 100000)
        c.require(c.integer(row['fixture']['owned_datasets']) == (100 if mode == 'native-small' else 0), 'original owned scope mismatch')
        c.require(row['fixture']['origin'] == ('native-original' if mode == 'native-small' else 'generated'), 'original source mismatch')
        fixtures[mode] = dict(row['fixture'])
    for index, parent in enumerate(c.PARENTS):
        row = parents[parent]
        c.object_keys(row, ('schema', 'controls', 'cleanup', 'artifacts'), 'original parent')
        c.require(c.exact(row['artifacts'], {'artifacts': provenance['artifacts'], 'go_version': provenance['guest_go']}), 'original parent launch artifacts mismatch')
        c.require(type(row['schema']) is int and row['schema'] == 1, 'original parent schema required')
        modes = c.MODES[:2] if index == 0 else c.MODES[2:]
        controls = row['controls']
        c.object_keys(controls, ('parent', 'modes', 'predicate_assertions', 'boundary' if index == 0 else 'native'), 'original parent controls')
        c.require(controls['parent'] == parent and c.exact(controls['predicate_assertions'], 4000), 'parent identity/assertions mismatch')
        c.object_keys(controls['modes'], modes, 'original parent modes')
        for mode in modes:
            expected = {'fixture': fixtures[mode], 'predicate_assertions': 2000, 'replacements': 3,
                        'native_guid_assertions': 100 if mode == 'native-small' else 0}
            c.require(c.exact(controls['modes'][mode], expected), 'original declaration/control mismatch')
        expected_cleanup = {'descendants_removed': True, 'sockets_removed': True, 'temporary_directories_removed': True,
                            'storage_cleanups': index, 'pairs': 3 if index == 0 else 2}
        c.require(c.exact(row['cleanup'], expected_cleanup), 'original whole guest cleanup incomplete')
    boundary = parents[c.PARENTS[0]]['controls']['boundary']
    expected = {'accepted_model_records': 100000, 'record_exhaustions': 1, 'byte_exhaustions': 1,
                'recoveries': 1, 'outside_normal_budget': True}
    measures = ('sampled_rss_bytes', 'high_water_sum_bytes', 'samples', 'max_sample_gap_ns')
    c.object_keys(boundary, (*expected, *measures), 'original boundary')
    c.require(c.exact({key: boundary[key] for key in expected}, expected), 'original boundary controls incomplete')
    for key in measures:
        c.integer(boundary[key], 1)
    native = parents[c.PARENTS[1]]['controls']['native']
    c.require(c.exact(native, {'captured_commands': 1, 'subtree_exits': 1, 'stale_retentions': 1,
                            'recoveries': 1, 'boot_state': 'unchanged'}), 'original native controls incomplete')
    controls = {'predicate_assertions': sum(parents[p]['controls']['predicate_assertions'] for p in c.PARENTS),
                'native_owned_count': fixtures['native-small']['owned_datasets'],
                'native_guid_assertions': parents[c.PARENTS[1]]['controls']['modes']['native-small']['native_guid_assertions'],
                'native_boot_state': native['boot_state'], 'boundary_outside_normal_budget': boundary['outside_normal_budget'],
                'guest_cleanup': all(parents[p]['cleanup']['temporary_directories_removed'] for p in c.PARENTS),
                'pair_descendants_removed': all(parents[p]['cleanup']['descendants_removed'] for p in c.PARENTS),
                'sockets_removed': all(parents[p]['cleanup']['sockets_removed'] for p in c.PARENTS)}
    controls.update(zip(('boundary_sampled_rss_bytes', 'boundary_high_water_bytes', 'boundary_samples',
                         'boundary_max_sample_gap_ns'), [boundary[k] for k in measures]))
    try:
        lines = raw.decode('utf-8').splitlines()
    except UnicodeError as error:
        raise ValueError('original transcript encoding invalid') from error
    retained, partials = [], set()
    for line in lines:
        if 'NATIVE_DASHBOARD_CONTROLS ' in line:
            match = re.fullmatch(r'\s*\S+\.go:[1-9][0-9]*: NATIVE_DASHBOARD_CONTROLS (.+)', line)
            c.require(match is not None, 'malformed partial parent record')
            value = c.load_json(match[1])
            parent = value.get('parent') if type(value) is dict else None
            c.require(type(parent) is str and parent in parents and parent not in partials
                      and c.exact(value, parents[parent]['controls']), 'partial parent disagrees with independently retained original')
            partials.add(parent)
            continue
        c.require('DASHBOARD_PROVENANCE ' not in line and 'DASHBOARD_CONTROLS ' not in line,
                  'guest cannot supply host-assembled provenance or controls')
        retained.append(line)
    c.require(partials == set(c.PARENTS), 'original partial parent output incomplete')
    payload = ('DASHBOARD_PROVENANCE ' + json.dumps(provenance, separators=(',', ':')) + '\n'
               + '\n'.join(retained) + '\nDASHBOARD_CONTROLS ' + json.dumps(controls, separators=(',', ':')) + '\n').encode()
    # Unknown observations and failures remain in the retained transcript. Full
    # strict and legacy checks below reject them; the raw original is never edited.
    rows = c.records(payload.decode())
    c.require(len(rows['DASHBOARD_RESOURCE']) == 4, 'complete measured modes required')
    for row in rows['DASHBOARD_RESOURCE']:
        mode = row.get('mode')
        c.require(type(mode) is str and mode in fixtures and c.exact(fixtures[mode], {k: row.get(k) for k in fixtures[mode]}),
                  'resource transcript disagrees with independent original fixture')
    return payload, fixtures


def context(root):
    root = Path(root).resolve()
    title = os.environ.get('PR_TITLE', '')
    if not title:
        raise ValueError('independent dashboard bindings require PR_TITLE and a completed clean candidate review')
    base = os.environ.get('BASE_REF', 'origin/main')
    identity = build.review.identity(root, base, title)
    candidates = []
    for path in (root / '.tools/review-runs').glob('*/result.json'):
        record, _ = build.read_json(path)
        if record.get('qualified') is True and all(c.exact(record.get(k), v) for k, v in identity.items()):
            candidates.append(path.parent)
    c.require(bool(candidates), 'independent dashboard bindings require completed exact-source review')
    # Selection is deterministic; each selected report is independently verified.
    selected = sorted(candidates)[-1]
    build.verify_review(root, selected, source_commit=identity['source_commit'],
                        base_commit=identity['base_commit'], pr_title=title)
    return {'source_commit': identity['source_commit'], 'base_commit': identity['base_commit'], 'pr_title': title}, selected


def command(argv, log, *, timeout=180, cwd=ROOT):
    process = None
    with Path(log).open('xb') as output:
        try:
            process = owned.OwnedProcess(argv, cwd=cwd, stdout=output, stderr=subprocess.STDOUT,
                                         runs=owned.RUNS, timeout=timeout)
            status = process.wait()
        finally:
            if process is not None:
                process.stop()
    state = process.record()
    c.require(status == 0 and build.stopped(state) and state.get('reason') == 'exit'
              and state.get('escalated') is False and 'cleanup_failure' not in state,
              'original command failed or teardown uncertain')
    return process.directory, state


def transfer_worker(name, candidate_directory, lab, port, expected):
    """Open actual bytes in an owned worker; supervisor stdin remains its lease."""
    _, remote = transfer_operation(name)
    candidate_directory = preflight.owned(ROOT, candidate_directory, 'dashboard-candidates')
    lab = preflight.owned(ROOT, lab, 'lab-runs')
    c.integer(port, 1, 65535)
    c.digest(expected)
    digest, payload = build.files.read_verified_file(candidate_directory / name, 128 << 20, retain=True)
    c.require(digest == expected, 'original transfer bytes changed')
    ssh = runner.ssh_command(lab / 'fixture-key', lab / 'known_hosts', port)
    subprocess.run(ssh + [remote], input=payload, check=True, timeout=180)
    c.require(build.files.read_verified_file(candidate_directory / name, 128 << 20)[0] == digest,
              'original transfer changed during execution')


def listener_absent(port):
    with socket.socket() as probe:
        probe.settimeout(2)
        # Refusal is absence. A timeout or unexpected socket error is uncertainty.
        return probe.connect_ex(('127.0.0.1', port)) == errno.ECONNREFUSED


def read_originals(lab, ssh, *, completed):
    originals, parents, supervisors = {}, {}, []
    for name, filename in ORIGINAL_NAMES.items():
        path = lab / filename
        directory, state = command(ssh + ['cat /root/work/.tools/' + filename], path, timeout=30)
        preflight.supervisor_record(ROOT, directory, ssh + ['cat /root/work/.tools/' + filename], completed)
        row, digest = build.read_json(path)
        c.require(path.stat().st_size <= 65536, 'bounded original sidecar required')
        (originals if name in c.MODES else parents)[name] = row
        supervisors.append({'name': filename, 'sha256': digest, 'directory': str(directory), 'state': state})
    return originals, parents, supervisors


def release_worker(release, lab, identity, review_directory):
    """One snapshot guest owned by the outer lease; every failure stays unqualified."""
    lab = preflight.owned(ROOT, lab, 'lab-runs')
    candidate_directory = ROOT / '.tools/dashboard-candidates' / lab.name
    record = {'suite': 'performance', 'release': release, **identity, 'qualified': False,
              'host_teardown': False, 'native_acceptance': False, 'commands': []}
    guest, base, before, port = None, None, {}, None
    vm = load('prepare-performance-vm')
    image, key, variables = lab / 'image', lab / 'fixture-key', lab / 'vars.fd'
    try:
        record['build'] = build.build_candidate(ROOT, review_directory, candidate_directory,
                                                source_commit=identity['source_commit'], base_commit=identity['base_commit'], pr_title=identity['pr_title'])
        binding = preflight.verify_build(ROOT, review_directory, candidate_directory, source_commit=identity['source_commit'],
                                         base_commit=identity['base_commit'], pr_title=identity['pr_title'])
        candidate = binding['candidate']
        qemu = str(Path(shutil.which('qemu-system-x86_64')).resolve())
        code, template = runner.firmware(qemu)
        pins = json.loads((ROOT / 'tools/performance-vm-pins.json').read_text())
        keys = json.loads((ROOT / 'tools/local-vm-pins.json').read_text())
        c.require(keys['builder'] == pins['builder'], 'image/key builder mismatch')
        qemu_version = subprocess.check_output([qemu, '--version'], text=True).splitlines()[0]
        command([sys.executable, str(ROOT / 'tools/prepare-performance-vm.py'), release, str(image),
                 '--cache', str(ROOT / '.tools/lab-cache')], lab / 'prepare-image.log', timeout=1800)
        base = image / 'freebsd' / ('v' + pins['builder']) / ('freebsd-' + release + '.qcow2')
        paths = {'base': base, 'qemu': Path(qemu), 'firmware_code': code, 'firmware_vars': template}
        before = {name: vm.digest(path) for name, path in paths.items()}
        shutil.copyfile(template, variables)
        pin = keys['fixture_keys'][release]
        vm.download(pin['url'], pin['sha256'], pin['bytes'], key)
        key.chmod(0o600)
        with socket.socket() as listener:
            listener.bind(('127.0.0.1', 0))
            port = listener.getsockname()[1]
        ssh = runner.ssh_command(key, lab / 'known_hosts', port)
        owned.write_record(lab / 'host-original.json', {
            'schema': 1, 'candidate': candidate, 'pins': before, 'port': port,
            'runtime_paths': {name: str(path) for name, path in paths.items() if name != 'base'},
            'host_os': platform.system(), 'host_arch': platform.machine(), 'qemu_version': qemu_version,
            'candidate_build_sha256': binding['record_sha256']})
        with (lab / 'serial.log').open('xb') as serial:
            guest = subprocess.Popen(runner.qemu_command(qemu, base, code, variables, port), stdin=subprocess.DEVNULL,
                                     stdout=serial, stderr=serial)
            record['qemu_pid'] = guest.pid
            record['port'] = port
            owned.write_record(lab / 'result.json', record)
            deadline = time.monotonic() + 900
            while True:
                c.require(guest.poll() is None, 'owned QEMU exited during boot')
                probe = subprocess.run(ssh + ['true'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=15)
                if probe.returncode == 0:
                    break
                c.require(time.monotonic() < deadline, 'owned guest boot deadline expired')
                time.sleep(3)
            hashes = {'source.tar': candidate['archive_sha256'], **{build.ARTIFACT_FILES[k]: v for k, v in candidate['artifacts'][release].items()}}
            for name, expected in hashes.items():
                selection, _ = transfer_operation(name)
                argv = [sys.executable, str(Path(__file__).resolve()), *selection, str(candidate_directory), str(lab), str(port), expected]
                directory, state = command(argv, lab / (name + '-transfer.log'), timeout=200)
                preflight.supervisor_record(ROOT, directory, argv, binding['completed_at'])
                record['commands'].append({'operation': 'transfer-' + name, 'directory': str(directory), 'state': state})
            argv = preflight.preflight_command(ROOT, lab, port)
            directory, state = command(argv, lab / 'dashboard-preflight.log', timeout=60)
            checked = preflight.verify_preflight(ROOT, review_directory, candidate_directory, lab, directory, release, port,
                                                  source_commit=identity['source_commit'], base_commit=identity['base_commit'], pr_title=identity['pr_title'])
            record['preflight'] = checked
            record['preflight_supervisor_directory'] = str(directory)
            original = checked['observed']
            provenance = {'candidate': candidate, 'release': release, 'artifacts': candidate['artifacts'][release],
                          'kernel': original['kernel'], 'userland': original['userland'], 'zfs': original['zfs'],
                          'guest_go': candidate['go_version'], 'host_os': platform.system(), 'host_arch': platform.machine(),
                          'qemu_version': qemu_version, 'arch': 'amd64', 'cpu_model': 'qemu64', 'cpu_count': 2,
                          'memory_mib': 6144, 'accelerator': 'tcg', 'physical_bytes': original['physical_bytes']}
            argv = ssh + [REMOTE]
            resource_directory, resource_state = command(argv, lab / 'resource-raw.log', timeout=1900)
            preflight.supervisor_record(ROOT, resource_directory, argv, state['completed_at'])
            record['resource_supervisor'] = {'directory': str(resource_directory), 'state': resource_state}
            after_directory, after_state = command(preflight.preflight_command(ROOT, lab, port), lab / 'dashboard-preflight-after.log', timeout=60)
            preflight.supervisor_record(ROOT, after_directory, preflight.preflight_command(ROOT, lab, port), resource_state['completed_at'])
            _, after_payload = build.files.read_verified_file(lab / 'dashboard-preflight-after.log', 65536, retain=True)
            c.require(c.exact(preflight.observation(after_payload, candidate, release), original), 'guest artifacts/environment changed during resources')
            record['after_preflight_supervisor_directory'] = str(after_directory)
            record['after_preflight_sha256'] = sha(after_payload)
            originals, parents, observations = read_originals(lab, ssh, completed=resource_state['completed_at'])
            record['originals'] = observations
            raw_sha, raw = build.files.read_verified_file(lab / 'resource-raw.log', c.CEILING, retain=True)
            payload, fixtures = assemble(raw, provenance, originals, parents)
            (lab / 'qualification.log').write_bytes(payload)
            record.update(candidate=candidate, provenance=provenance, fixtures=fixtures, raw_transcript_sha256=raw_sha,
                          transcript_sha256=sha(payload), resource_commands_complete=True)
            c.require(c.exact(preflight.verify_build(ROOT, review_directory, candidate_directory,
                                                   source_commit=identity['source_commit'], base_commit=identity['base_commit'], pr_title=identity['pr_title']), binding),
                      'candidate changed during guest qualification')
    except BaseException as error:
        record['failure'] = type(error).__name__ + ': ' + str(error)
        raise
    finally:
        cleanup_errors = []
        try:
            runner.stop_guest(guest)
            record['qemu_exited'] = guest is not None and guest.poll() is not None
            record['listener_removed'] = port is not None and listener_absent(port)
            if before:
                after = {name: vm.digest(path) for name, path in paths.items()}
                record['preservation'] = {name: {'before': value, 'after': after[name]} for name, value in before.items()}
                c.require(before == after, 'base/runtime changed')
                command([sys.executable, str(ROOT / 'tools/prepare-performance-vm.py'), release, str(image), '--verify'], lab / 'verify-image.log')
        except BaseException as error:
            cleanup_errors.append(type(error).__name__ + ': ' + str(error))
        # Independent cleanup attempts are necessary on failure; never retry a
        # failed deletion or turn the eventual removal into a successful run.
        for path, name in ((key, 'fixture_key_removed'), (variables, 'variables_removed'), (image, 'image_removed')):
            try:
                if path.exists() or path.is_symlink():
                    if path == image and not path.is_symlink():
                        shutil.rmtree(path)
                    else:
                        path.unlink()
                record[name] = not path.exists() and not path.is_symlink()
            except BaseException as error:
                cleanup_errors.append(type(error).__name__ + ': ' + str(error))
                record[name] = False
        try:
            if candidate_directory.exists():
                build.remove_artifacts(ROOT, candidate_directory)
        except BaseException as error:
            cleanup_errors.append(type(error).__name__ + ': ' + str(error))
        record['host_teardown'] = not cleanup_errors and all(record.get(k) is True for k in
            ('qemu_exited', 'listener_removed', 'fixture_key_removed', 'variables_removed', 'image_removed'))
        if cleanup_errors:
            record['cleanup_failure'] = cleanup_errors
        owned.write_record(lab / 'result.json', record)
    c.require(record['host_teardown'], 'host teardown unverified')
    return record


def finalize(lab, identity, release, process_directory, worker_command):
    """Reopen actual worker, supervisor and transcript after whole owned group exits."""
    lab = preflight.owned(ROOT, lab, 'lab-runs')
    preflight.supervisor_record(ROOT, process_directory, worker_command, 0)
    record, record_sha = build.read_json(lab / 'result.json')
    c.require('failure' not in record and 'cleanup_failure' not in record and record.get('resource_commands_complete') is True,
              'original worker failed or incomplete')
    payload_sha, payload = build.files.read_verified_file(lab / 'qualification.log', c.CEILING, retain=True)
    c.require(payload_sha == record['transcript_sha256'], 'original assembled transcript changed')
    c.require(build.files.read_verified_file(lab / 'resource-raw.log', c.CEILING)[0] == record['raw_transcript_sha256'],
              'original resource transcript changed')
    c.require(all(c.exact(record.get(k), v) for k, v in identity.items()) and record['release'] == release,
              'original source/release changed')
    host, host_sha = build.read_json(lab / 'host-original.json')
    c.object_keys(host, ('schema', 'candidate', 'pins', 'port', 'runtime_paths', 'host_os', 'host_arch',
                        'qemu_version', 'candidate_build_sha256'), 'original host context')
    c.require(type(host['schema']) is int and host['schema'] == 1 and c.exact(host['candidate'], record['candidate'])
              and c.exact(host['pins'], {k: v['before'] for k, v in record['preservation'].items()})
              and c.exact(host['port'], record['port']), 'independent host context changed')
    c.object_keys(host['runtime_paths'], ('qemu', 'firmware_code', 'firmware_vars'), 'original runtime paths')
    vm = load('prepare-performance-vm')
    for name, path in host['runtime_paths'].items():
        c.require(vm.digest(Path(path)) == host['pins'][name], 'original host runtime changed after teardown')
    for key in ('host_os', 'host_arch', 'qemu_version'):
        c.require(c.exact(record['provenance'][key], host[key]), 'original host provenance changed')
    ssh = runner.ssh_command(lab / 'fixture-key', lab / 'known_hosts', host['port'])
    candidate_directory = ROOT / '.tools/dashboard-candidates' / lab.name
    built, _ = build.read_json(candidate_directory / 'result.json')
    c.require(c.exact(built, dict(record['build'], artifacts_removed=True)), 'original retained builder record changed')
    original_build_sha = sha((json.dumps(dict(built, artifacts_removed=False), indent=2) + '\n').encode())
    c.require(original_build_sha == host['candidate_build_sha256']
              and original_build_sha == record['preflight']['candidate_build_sha256'], 'original build binding changed')
    for name in ('source.tar', *build.ARTIFACT_FILES.values()):
        c.require(not (candidate_directory / name).exists() and not (candidate_directory / name).is_symlink(), 'original candidate artifact remains')
    completed = max(s['completed_at'] for s in record['build']['supervisors'])
    transfer_rows = record['commands']
    c.require(type(transfer_rows) is list and len(transfer_rows) == 4, 'original transfers incomplete')
    seen = set()
    for transfer in transfer_rows:
        name = transfer['operation'].removeprefix('transfer-')
        selection, _ = transfer_operation(name)
        c.require(name not in seen, 'duplicate original transfer')
        seen.add(name)
        expected = record['candidate']['archive_sha256'] if name == 'source.tar' else next(
            value for role, value in record['candidate']['artifacts'][release].items() if build.ARTIFACT_FILES[role] == name)
        argv = [sys.executable, str(Path(__file__).resolve()), *selection, str(candidate_directory), str(lab), str(host['port']), expected]
        preflight.supervisor_record(ROOT, Path(transfer['directory']), argv, completed)
    preflight.supervisor_record(ROOT, Path(record['preflight_supervisor_directory']), preflight.preflight_command(ROOT, lab, host['port']), completed)
    first_state, _ = build.read_json(Path(record['preflight_supervisor_directory']) / 'state.json')
    preflight.supervisor_record(ROOT, Path(record['resource_supervisor']['directory']), ssh + [REMOTE], first_state['completed_at'])
    resource_state, _ = build.read_json(Path(record['resource_supervisor']['directory']) / 'state.json')
    preflight.supervisor_record(ROOT, Path(record['after_preflight_supervisor_directory']), preflight.preflight_command(ROOT, lab, host['port']), resource_state['completed_at'])
    first_sha, first_payload = build.files.read_verified_file(lab / 'dashboard-preflight.log', 65536, retain=True)
    after_sha, after_payload = build.files.read_verified_file(lab / 'dashboard-preflight-after.log', 65536, retain=True)
    c.require(first_sha == record['preflight']['transcript_sha256'] and after_sha == record['after_preflight_sha256']
              and c.exact(preflight.observation(first_payload, record['candidate'], release), record['preflight']['observed'])
              and c.exact(preflight.observation(after_payload, record['candidate'], release), record['preflight']['observed']),
              'original before/after guest preflight changed')
    # Read originals again, after the worker is gone. A copied resource receipt
    # cannot replace the original sidecars or their actual transfer supervisors.
    originals, parents = {}, {}
    for observation in record['originals']:
        row, digest = build.read_json(lab / observation['name'])
        c.require(digest == observation['sha256'], 'original sidecar changed after worker')
        name = next(k for k, v in ORIGINAL_NAMES.items() if v == observation['name'])
        preflight.supervisor_record(ROOT, Path(observation['directory']), ssh + ['cat /root/work/.tools/' + observation['name']], resource_state['completed_at'])
        (originals if name in c.MODES else parents)[name] = row
    _, raw = build.files.read_verified_file(lab / 'resource-raw.log', c.CEILING, retain=True)
    rebuilt, fixtures = assemble(raw, record['provenance'], originals, parents)
    c.require(rebuilt == payload and c.exact(fixtures, record['fixtures']), 'original assembly changed')
    cleanup = ('host_teardown', 'fixture_key_removed', 'image_removed', 'variables_removed', 'listener_removed', 'qemu_exited')
    c.require(all(record.get(k) is True for k in cleanup), 'original host cleanup incomplete')
    for name in ('fixture-key', 'image', 'vars.fd'):
        c.require(not (lab / name).exists() and not (lab / name).is_symlink(), 'original host artifact remains')
    c.require(listener_absent(record['port']), 'original QEMU listener remains')
    strict_record = {k: record[k] for k in ('release', 'candidate', 'transcript_sha256', 'provenance', 'preservation', *cleanup)}
    strict_record.update(guest_assertions=True, qualified=True)
    supervisor = {'exit_code': 0, 'group_cleaned': True, 'stopped_verified': True,
                  'source_commit': identity['source_commit'], 'release': release, 'transcript_sha256': payload_sha}
    run = {'release': release, 'pins': host['pins'],
           'fixtures': fixtures, 'record': strict_record, 'transcript': payload, 'supervisor': supervisor}
    c.validate_run(record['candidate'], run)
    build.source_identity(ROOT, identity['source_commit'], identity['base_commit'], identity['pr_title'])
    c.require(build.read_json(lab / 'result.json')[1] == record_sha and build.read_json(lab / 'host-original.json')[1] == host_sha,
              'original worker/host record changed')
    record.update(qualified=True, guest_assertions=True, supervisor_directory=str(process_directory),
                  qualification_scope='one release resource receipt; both-release and broader acceptance remain separate')
    owned.write_record(lab / 'result.json', record)
    return run


def run_release(release, owned_directory=None):
    c.require(release in c.RELEASES, 'declared dashboard release required')
    identity, review_directory = context(ROOT)
    for name in ('qemu-system-x86_64', 'unzstd', 'ssh', 'git'):
        c.require(shutil.which(name), 'missing local prerequisite: ' + name)
    c.require(platform.system() in ('Darwin', 'Linux'), 'unsupported local host')
    if owned_directory is not None:
        return release_worker(release, owned_directory, identity, review_directory)
    runs = ROOT / '.tools/lab-runs'
    runs.mkdir(mode=0o700, parents=True, exist_ok=True)
    owned.private_directory(runs)
    lab = Path(tempfile.mkdtemp(prefix='performance-' + release + '-', dir=runs))
    worker = [sys.executable, str(ROOT / 'tools/local-qualification.py'), 'performance', release, '--run', '--owned-directory', str(lab)]
    process = owned.OwnedProcess(worker, native_fixture=lab)
    try:
        status = process.wait()
    finally:
        process.stop()
    c.require(status == 0, 'dashboard worker failed; inspect private owned records')
    try:
        result = finalize(lab, identity, release, process.directory, worker)
    except BaseException as error:
        record, _ = build.read_json(lab / 'result.json')
        record.update(qualified=False, validation_failure=type(error).__name__ + ': ' + str(error))
        owned.write_record(lab / 'result.json', record)
        raise
    print('PASS: one-release dashboard resource receipt and owned teardown; both-release/browser acceptance remains open', flush=True)
    print('Qualification record: ' + str(lab / 'result.json'), flush=True)
    return result


def verify_pair(labs):
    """Reopen two explicitly selected original owned runs; never select by success."""
    c.require(type(labs) is list and len(labs) == 2, 'two explicit owned dashboard runs required')
    runs = []
    for directory in labs:
        lab = preflight.owned(ROOT, directory, 'lab-runs')
        record, _ = build.read_json(lab / 'result.json')
        c.require(record.get('qualified') is True, 'failed/skipped/unqualified run cannot be promoted')
        identity = {k: record[k] for k in ('source_commit', 'base_commit', 'pr_title')}
        worker = [sys.executable, str(ROOT / 'tools/local-qualification.py'), 'performance', record['release'], '--run', '--owned-directory', str(lab)]
        runs.append(finalize(lab, identity, record['release'], Path(record['supervisor_directory']), worker))
    candidate = runs[0]['record']['candidate']
    c.require(all(c.exact(run['record']['candidate'], candidate) for run in runs), 'both release candidates must be identical')
    return c.validate(candidate, runs)


if __name__ == '__main__':
    # Private owned transfer worker only. The existing local runner owns user
    # entry, opt-in and release selection; no new public suite/flag is introduced.
    c.require(len(sys.argv) == 7 and sys.argv[1] == 'transfer', 'fixed transfer worker required')
    transfer_worker(sys.argv[2], Path(sys.argv[3]), Path(sys.argv[4]), int(sys.argv[5]), sys.argv[6])
