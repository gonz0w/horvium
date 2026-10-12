#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Adopted separate diagnostic profile; no lifecycle acceptance."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]
BINARY = 'policy-startup-provenance-native.test'
NORMAL = 'policy-startup-provenance-normal.test'
BEGIN = 'POLICY_STARTUP_PROVENANCE_TRANSCRIPT_BEGIN\n'
END = 'POLICY_STARTUP_PROVENANCE_TRANSCRIPT_END\n'
ARGUMENTS = ['go', 'test', '-c', '-trimpath', '-tags=horvium_native_fixture', '-o', '<owned-artifact>', './internal/host/freebsd']


def load(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


candidate = load('check-policy-provenance-candidate')
consumer = candidate.consumer


def check_selections(normal, tagged):
    for selection in (normal, tagged):
        consumer.shape(selection, {'storage', 'host'})
        for package in selection.values():
            consumer.shape(package, {'go_files', 'test_files'})
            for files in package.values():
                if type(files) is not list or len(files) > 1024 or any(type(f) is not str or re.fullmatch('[A-Za-z0-9_]+\\.go', f) is None for f in files) or len(set(files)) != len(files):
                    raise ValueError('bounded exact compiler file selection required')
    storage_tagged = {'deadline_return_fixture.go', 'deadline_semantics_fixture.go'}
    host_tagged = {'execution_policy_startup_provenance_fixture_test.go', 'execution_policy_provenance_support_test.go', 'execution_policy_provenance_support_fixture_test.go'}
    ordinary_storage = set(normal['storage']['go_files'])
    tagged_storage = set(tagged['storage']['go_files'])
    ordinary_host = set(normal['host']['test_files'])
    tagged_host = set(tagged['host']['test_files'])
    if 'deadline_return_plain.go' not in ordinary_storage or storage_tagged & ordinary_storage or 'deadline_return_plain.go' in tagged_storage or not storage_tagged <= tagged_storage or host_tagged & ordinary_host or not host_tagged <= tagged_host:
        raise ValueError('ordinary/tagged diagnostic source isolation required')


def check_build(build):
    if type(build) is not dict:
        raise ValueError('external reviewed build receipt required')
    for name, size in [('source_commit', 40), ('compiler_sha256', 64), ('sha256', 64)]:
        consumer.digest(build.get(name), size)
    if build.get('target') != 'freebsd/amd64' or build.get('goamd64') != 'v1' or build.get('cgo_enabled') is not False or build.get('build_arguments') != ARGUMENTS:
        raise ValueError('original pinned FreeBSD tagged build required')
    proof = build.get('source_isolation')
    consumer.shape(proof, {'schema', 'normal_binary_sha256', 'normal', 'tagged'})
    if type(proof['schema']) is not int or proof['schema'] != 1:
        raise ValueError('known source isolation receipt required')
    consumer.digest(proof['normal_binary_sha256'], 64)
    if proof['normal_binary_sha256'] == build['sha256']:
        raise ValueError('separate ordinary and tagged executables required')
    check_selections(proof['normal'], proof['tagged'])

def command(build):
    check_build(build)
    return ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_POLICY_FACTS_VM=1 '
            'HORVIUM_POLICY_STARTUP_PROVENANCE_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1 '
            'HORVIUM_POLICY_PROVENANCE_SOURCE_ISOLATED=1 '
            'HORVIUM_TEST_BINARY_SHA256=' + build['sha256'] + ' '
            'HORVIUM_SOURCE_COMMIT=' + build['source_commit'] + ' '
            'HORVIUM_COMPILER_SHA256=' + build['compiler_sha256'] + ' '
            'sh tools/bsd-policy-startup-provenance-prebuilt.sh')


def transcript(log, complete=False):
    if type(log) is not str or len(log.encode('utf-8')) > 1 << 20 or log.count(BEGIN) != 1 or log.count(END) != 1:
        raise ValueError('one bounded full diagnostic frame required')
    prefix, remainder = log.split(BEGIN)
    text, suffix = remainder.split(END)
    outer = prefix + suffix
    if any(marker in outer for marker in ('POLICY_STARTUP_PROVENANCE_', 'POLICY_FACTS_CHILD_RESULT', 'POLICY_FACTS_PARENT_RESULT')) or re.search(r'(?m)^(?:=== RUN |--- (?:PASS|FAIL|SKIP):|PASS$)', outer):
        raise ValueError('diagnostic/test evidence outside canonical frame')
    if complete and (re.findall(r'(?m)^DEVICE_FIXTURE_CLEANUP (.*)$', log) != ['pass'] or re.findall(r'(?m)^DEVICE_FIXTURE_CLEANUP (.*)$', suffix) != ['pass']):
        raise ValueError('independent post-parent device teardown required')
    return text


def validate(log, build, complete=False):
    check_build(build)
    text = transcript(log, complete)
    result = consumer.validate(text, source_commit=build['source_commit'],
                               compiler_sha256=build['compiler_sha256'], artifact_sha256=build['sha256'])
    result['transcript_sha256'] = hashlib.sha256(text.encode()).hexdigest()
    if build.get('validated_transcript_sha256', result['transcript_sha256']) != result['transcript_sha256']:
        raise ValueError('canonical evidence changed after actual-file verification')
    if not result['diagnostic_complete']:
        raise candidate.IncompleteDiagnostic(result)
    return result


def verify_run(directory, build, log):
    check_build(build)
    text = transcript(log)
    path = directory / 'provenance-transcript.txt'
    try:
        fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    except OSError as error:
        raise ValueError('fresh owned canonical transcript required') from error
    with os.fdopen(fd, 'w', encoding='utf-8') as stream:
        stream.write(text)
    return candidate.verify_candidate(ROOT, path, ROOT / '.tools/go/bin/go', directory / BINARY,
                                      source_commit=build['source_commit'], compiler_sha256=build['compiler_sha256'],
                                      artifact_sha256=build['sha256'])


def build_isolation(go, env, directory, source):
    candidate.verify_source(ROOT, source)
    selections = []
    for tags in ([], ['-tags=horvium_native_fixture']):
        selection = {}
        for name, package in [('storage', './internal/storage'), ('host', './internal/host/freebsd')]:
            record = json.loads(subprocess.check_output([go, 'list', '-json', *tags, package], cwd=ROOT, env=env, text=True))
            selection[name] = {'go_files': record.get('GoFiles', []), 'test_files': record.get('TestGoFiles', [])}
        selections.append(selection)
    normal, tagged = selections
    check_selections(normal, tagged)
    binary = directory / NORMAL
    try:
        subprocess.run([go, 'test', '-c', '-trimpath', '-o', str(binary), './internal/host/freebsd'], cwd=ROOT, env=env, check=True)
        digest, _ = candidate.read_verified_file(binary, 128 << 20, executable=True)
    finally:
        binary.unlink(missing_ok=True)
    candidate.verify_source(ROOT, source)
    return {'schema': 1, 'normal_binary_sha256': digest, 'normal': normal, 'tagged': tagged}
