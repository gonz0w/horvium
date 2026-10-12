#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Run one suite in an owned snapshot guest; never edits host SSH configuration."""
import argparse
import importlib.util
import json
import platform
import re
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import time
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TOOLS = ROOT / 'tools'
SUITES = {
    'policy-startup-envelope': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_POLICY_FACTS_VM=1 HORVIUM_POLICY_STARTUP_ENVELOPE_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1 sh tools/bsd-policy-startup-envelope-prebuilt.sh'),
    'policy-startup-trace': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_POLICY_FACTS_VM=1 HORVIUM_POLICY_STARTUP_TRACE_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1 sh tools/bsd-policy-startup-trace-prebuilt.sh'),
    'policy-startup-diagnostics': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_POLICY_FACTS_VM=1 HORVIUM_POLICY_STARTUP_DIAGNOSTICS_VM=1 sh tools/bsd-policy-startup-diagnostics-prebuilt.sh'),
    'policy-startup-lifecycle': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_POLICY_FACTS_VM=1 HORVIUM_POLICY_ROOT_REVIEW_VM=1 HORVIUM_POLICY_ROOT_LIFECYCLE_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1 sh tools/bsd-policy-startup-lifecycle-prebuilt.sh'),
    'policy-startup-review': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_POLICY_FACTS_VM=1 HORVIUM_POLICY_ROOT_REVIEW_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1 sh tools/bsd-policy-startup-review-prebuilt.sh'),
    'policy-startup-facts': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_POLICY_FACTS_VM=1 sh tools/bsd-policy-startup-facts-prebuilt.sh'),
    'policy-startup-publication': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_STATE_VM=1 HORVIUM_RESTART_VM=1 sh tools/bsd-policy-startup-publication-prebuilt.sh'),
    'policy-startup-owner': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_BACKING_VM=1 HORVIUM_PROPERTY_VM=1 HORVIUM_EXECUTION_STARTUP_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1 sh tools/bsd-policy-startup-owner-prebuilt.sh'),
    'closed-startup-publication': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_STATE_VM=1 HORVIUM_RESTART_VM=1 sh tools/bsd-closed-startup-publication-prebuilt.sh'),
    'closed-startup-owner': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_BACKING_VM=1 HORVIUM_PROPERTY_VM=1 HORVIUM_EXECUTION_STARTUP_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1 sh tools/bsd-closed-startup-owner-prebuilt.sh'),
    'executor-recovery': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_PROPERTY_VM=1 HORVIUM_RESEARCH_VM=1 sh tools/research/m2-executor-recovery.sh'),
    'action-peer': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_ACTION_PEER_VM=1 sh tools/bsd-action-peer-prebuilt.sh'),
    'foundation': ('git', 'sh tools/bsd-foundation-smoke.sh FreeBSD'),
    'research': ('python314', 'env HORVIUM_RESEARCH_VM=1 sh tools/research/m1-native.sh'),
    'discovery': ('python314', 'env HORVIUM_NATIVE_VM=1 sh tools/bsd-native-discovery.sh'),
    'devices': ('python314', 'env HORVIUM_NATIVE_VM=1 HORVIUM_DEVICE_VM=1 sh tools/bsd-devices.sh'),
    'execution-startup': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_BACKING_VM=1 HORVIUM_PROPERTY_VM=1 HORVIUM_EXECUTION_STARTUP_VM=1 sh tools/bsd-execution-startup-prebuilt.sh'),
    'execution-system': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_BACKING_VM=1 HORVIUM_PROPERTY_VM=1 HORVIUM_EXECUTION_SYSTEM_VM=1 sh tools/bsd-execution-system-prebuilt.sh'),
    'joint-leases': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_BACKING_VM=1 sh tools/bsd-joint-leases-prebuilt.sh'),
    'original-backing': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_BACKING_VM=1 sh tools/bsd-original-backing-prebuilt.sh'),
    'persistent-backing': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_BACKING_VM=1 sh tools/bsd-persistent-backing-prebuilt.sh'),
    'state': ('python314', 'env HORVIUM_NATIVE_VM=1 HORVIUM_STATE_VM=1 sh tools/bsd-state.sh'),
    'properties': ('python314', 'env HORVIUM_NATIVE_VM=1 HORVIUM_PROPERTY_VM=1 sh tools/bsd-properties.sh'),
    'job-status': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_STATUS_VM=1 sh tools/bsd-storage-job-status-prebuilt.sh'),
    'pool-properties': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_PROPERTY_VM=1 sh tools/bsd-pool-properties-prebuilt.sh'),
    'continuity': ('python314 samba423', 'env HORVIUM_NATIVE_VM=1 HORVIUM_CONTINUITY_VM=1 sh tools/bsd-continuity.sh'),
    'performance': ('python314', 'env HORVIUM_NATIVE_VM=1 sh tools/bsd-native-discovery.sh && env HORVIUM_NATIVE_VM=1 HORVIUM_PERF_VM=1 sh tools/bsd-performance.sh'),
}

PREBUILT = {
    'policy-startup-envelope': ('policy-startup-envelope-native.test', './internal/host/freebsd', 'POLICY_STARTUP_ENVELOPE_PREBUILT_SHA256', 'HORVIUM_POLICY_FACTS_VM=1 HORVIUM_POLICY_STARTUP_ENVELOPE_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1', 'bsd-policy-startup-envelope-prebuilt.sh'),
    'policy-startup-trace': ('policy-startup-trace-native.test', './internal/host/freebsd', 'POLICY_STARTUP_TRACE_PREBUILT_SHA256', 'HORVIUM_POLICY_FACTS_VM=1 HORVIUM_POLICY_STARTUP_TRACE_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1', 'bsd-policy-startup-trace-prebuilt.sh'),
    'policy-startup-diagnostics': ('policy-startup-diagnostics-native.test', './internal/host/freebsd', 'POLICY_STARTUP_COST_PREBUILT_SHA256', 'HORVIUM_POLICY_FACTS_VM=1 HORVIUM_POLICY_STARTUP_DIAGNOSTICS_VM=1', 'bsd-policy-startup-diagnostics-prebuilt.sh'),
    'policy-startup-lifecycle': ('policy-startup-lifecycle-native.test', './internal/host/freebsd', 'POLICY_ROOT_LIFECYCLE_PREBUILT_SHA256', 'HORVIUM_POLICY_FACTS_VM=1 HORVIUM_POLICY_ROOT_REVIEW_VM=1 HORVIUM_POLICY_ROOT_LIFECYCLE_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1', 'bsd-policy-startup-lifecycle-prebuilt.sh'),
    'policy-startup-review': ('policy-startup-review-native.test', './internal/host/freebsd', 'POLICY_ROOT_REVIEW_PREBUILT_SHA256', 'HORVIUM_POLICY_FACTS_VM=1 HORVIUM_POLICY_ROOT_REVIEW_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1', 'bsd-policy-startup-review-prebuilt.sh'),
    'policy-startup-facts': ('policy-startup-facts-native.test', './internal/host/freebsd', 'POLICY_FACTS_PREBUILT_SHA256', 'HORVIUM_POLICY_FACTS_VM=1', 'bsd-policy-startup-facts-prebuilt.sh'),
    'policy-startup-publication': ('policy-startup-publication-native.test', './internal/storage', 'POLICY_STARTUP_PUBLICATION_PREBUILT_SHA256', 'HORVIUM_STATE_VM=1 HORVIUM_RESTART_VM=1', 'bsd-policy-startup-publication-prebuilt.sh'),
    'policy-startup-owner': ('policy-startup-owner-native.test', './internal/host/freebsd', 'POLICY_STARTUP_OWNER_PREBUILT_SHA256', 'HORVIUM_BACKING_VM=1 HORVIUM_PROPERTY_VM=1 HORVIUM_EXECUTION_STARTUP_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1', 'bsd-policy-startup-owner-prebuilt.sh'),
    'closed-startup-publication': ('closed-startup-publication-native.test', './internal/storage', 'CLOSED_STARTUP_PUBLICATION_PREBUILT_SHA256', 'HORVIUM_STATE_VM=1 HORVIUM_RESTART_VM=1', 'bsd-closed-startup-publication-prebuilt.sh'),
    'closed-startup-owner': ('closed-startup-owner-native.test', './internal/host/freebsd', 'CLOSED_STARTUP_OWNER_PREBUILT_SHA256', 'HORVIUM_BACKING_VM=1 HORVIUM_PROPERTY_VM=1 HORVIUM_EXECUTION_STARTUP_VM=1 HORVIUM_CLOSED_STARTUP_OWNER_VM=1', 'bsd-closed-startup-owner-prebuilt.sh'),
    'execution-startup': ('execution-startup-native.test', './internal/host/freebsd', 'EXECUTION_STARTUP_PREBUILT_SHA256', 'HORVIUM_BACKING_VM=1 HORVIUM_PROPERTY_VM=1 HORVIUM_EXECUTION_STARTUP_VM=1', 'bsd-execution-startup-prebuilt.sh'),
    'execution-system': ('execution-system-native.test', './internal/host/freebsd', 'EXECUTION_SYSTEM_PREBUILT_SHA256', 'HORVIUM_BACKING_VM=1 HORVIUM_PROPERTY_VM=1 HORVIUM_EXECUTION_SYSTEM_VM=1', 'bsd-execution-system-prebuilt.sh'),
    'joint-leases': ('joint-leases-native.test', './internal/host/freebsd', 'JOINT_LEASES_PREBUILT_SHA256', 'HORVIUM_BACKING_VM=1', 'bsd-joint-leases-prebuilt.sh'),
    'original-backing': ('original-backing-native.test', './internal/host/freebsd', 'ORIGINAL_BACKING_PREBUILT_SHA256', 'HORVIUM_BACKING_VM=1', 'bsd-original-backing-prebuilt.sh'),
    'persistent-backing': ('persistent-backing-native.test', './internal/host/freebsd', 'PERSISTENT_BACKING_PREBUILT_SHA256', 'HORVIUM_BACKING_VM=1', 'bsd-persistent-backing-prebuilt.sh'),
    'executor-recovery': ('pool-properties-native.test', './internal/host/freebsd', 'EXECUTOR_RECOVERY_PREBUILT_SHA256', 'HORVIUM_PROPERTY_VM=1 HORVIUM_RESEARCH_VM=1', 'research/m2-executor-recovery.sh'),
    'action-peer': ('action-peer-native.test', './internal/helper', 'ACTION_PEER_PREBUILT_SHA256', 'HORVIUM_ACTION_PEER_VM=1', 'bsd-action-peer-prebuilt.sh'),
    'job-status': ('storage-job-status-native.test', './cmd/horvium', 'STORAGE_JOB_STATUS_PREBUILT_SHA256', 'HORVIUM_STATUS_VM=1', 'bsd-storage-job-status-prebuilt.sh'),
    'state': ('state-native.test', './internal/storage', 'STATE_PREBUILT_SHA256', 'HORVIUM_STATE_VM=1', 'bsd-state-prebuilt.sh'),
    'pool-properties': ('pool-properties-native.test', './internal/host/freebsd', 'POOL_PROPERTIES_PREBUILT_SHA256', 'HORVIUM_PROPERTY_VM=1', 'bsd-pool-properties-prebuilt.sh'),
}


def cleanup_prebuilt(directory):
    # Only fixed registered executable names in this owned run directory.
    for filename in {record[0] for record in PREBUILT.values()}:
        (directory / filename).unlink(missing_ok=True)


def load_tool(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), TOOLS / (name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def suite_command(suite, source_build=False, artifact_digest=None):
    if suite in PREBUILT and not source_build:
        if not isinstance(artifact_digest, str) or not re.fullmatch('[a-f0-9]{64}', artifact_digest):
            raise ValueError('prebuilt suite requires an exact artifact checksum')
        _, _, _, optin, script = PREBUILT[suite]
        return ('', 'env HORVIUM_NATIVE_VM=1 ' + optin + ' '
                'HORVIUM_TEST_BINARY_SHA256=' + artifact_digest + ' sh tools/' + script)
    return SUITES[suite]


def uses_device_fixture(suite):
    return suite in ('policy-startup-envelope', 'policy-startup-trace', 'policy-startup-diagnostics', 'policy-startup-lifecycle', 'policy-startup-review', 'policy-startup-facts', 'policy-startup-owner', 'devices', 'pool-properties', 'executor-recovery', 'execution-system', 'execution-startup', 'closed-startup-owner')


def recovery_boot_identity(text):
    try:
        data = json.loads(text)
        boot = [data['boot_sec'], data['boot_usec']]
        if any(type(v) is not int for v in boot) or boot[0] < 1 or not 0 <= boot[1] < 1000000:
            raise ValueError('invalid observed boot timeval')
        return boot
    except (KeyError, TypeError, json.JSONDecodeError) as error:
        raise ValueError('missing observed boot timeval') from error


def run_executor_recovery(ssh, guest, remote, log):
    """Research-only two phases in one owned guest; no service or reboot retry."""
    subprocess.run(ssh + [remote + ' before'], stdout=log, stderr=subprocess.STDOUT, check=True, timeout=420)
    observer = 'cd /root/work && .tools/executor-observer 1'
    before = subprocess.run(ssh + [observer], capture_output=True, text=True, check=True, timeout=15)
    old_boot = recovery_boot_identity(before.stdout)
    # SSH may disconnect before shutdown returns. Only a separately observed
    # changed boot and successful readback phase can establish completion.
    shutdown = subprocess.run(ssh + ['/sbin/shutdown -r now'], capture_output=True, timeout=30)
    if shutdown.returncode not in (0, 255):
        raise ValueError('owned guest reboot request failed')
    deadline = time.monotonic() + 300
    new_boot = old_boot
    while time.monotonic() < deadline:
        if guest.poll() is not None:
            raise ValueError('owned QEMU exited during reboot')
        try:
            probe = subprocess.run(ssh + [observer], capture_output=True, text=True, timeout=15)
            if probe.returncode == 0:
                new_boot = recovery_boot_identity(probe.stdout)
                if new_boot != old_boot:
                    break
        except subprocess.TimeoutExpired:
            pass
        time.sleep(2)
    if new_boot == old_boot:
        raise ValueError('owned guest reboot was not independently observed')
    log.write('EXECUTOR_REBOOT ' + json.dumps({'before': old_boot, 'after': new_boot}, separators=(',', ':')) + '\n')
    log.flush()
    subprocess.run(ssh + [remote + ' after'], stdout=log, stderr=subprocess.STDOUT, check=True, timeout=420)


def source_identity(source):
    if (subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip() != source
            or subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT)):
        raise ValueError('source changed during native qualification')


def build_native_binary(directory, source, suite):
    plan = load_tool('review-plan')
    subprocess.run([sys.executable, str(TOOLS / 'bootstrap-go.py')], check=True)
    go = str(ROOT / '.tools/go/bin/go')
    env = plan.go_environment(ROOT, native=True)
    version = subprocess.check_output([go, 'version'], env=env, text=True).strip()
    if version.split()[2] != json.loads((TOOLS / 'go-pins.json').read_text())['version']:
        raise ValueError('unexpected host Go compiler version')
    subprocess.run([go, 'mod', 'download'], cwd=ROOT, env=env, check=True)
    subprocess.run([go, 'mod', 'verify'], cwd=ROOT, env=env, check=True)
    filename, package, _, _, _ = PREBUILT[suite]
    binary = directory / filename
    tags = ['-tags=horvium_native_fixture'] if suite in ('policy-startup-envelope', 'policy-startup-trace', 'policy-startup-owner', 'policy-startup-facts', 'policy-startup-review', 'policy-startup-lifecycle', 'policy-startup-diagnostics') else []
    command = [go, 'test', '-c', '-trimpath', *tags, '-o', str(binary), package]
    source_identity(source)
    subprocess.run(command, cwd=ROOT, env=env, check=True)
    source_identity(source)
    return {'source_commit': source, 'sha256': plan.digest(binary), 'bytes': binary.stat().st_size,
            'go_version': version, 'compiler_sha256': plan.digest(Path(go)),
            'go_pins_sha256': plan.digest(TOOLS / 'go-pins.json'),
            'go_mod_sha256': plan.digest(ROOT / 'go.mod'), 'go_sum_sha256': plan.digest(ROOT / 'go.sum'),
            'target': 'freebsd/amd64', 'goamd64': 'v1', 'cgo_enabled': False,
            'build_arguments': ['go', 'test', '-c', '-trimpath', *tags, '-o', '<owned-artifact>', package]}


def validate_prebuilt(log, artifact_digest, suite='state'):
    marker = PREBUILT[suite][2]
    if re.findall(r'(?m)^' + marker + r' ([a-f0-9]{64})$', log) != [artifact_digest]:
        raise ValueError('guest did not verify the expected executable')


def cpu_count_for_suite(suite):
    if suite not in SUITES:
        raise ValueError('unknown native suite CPU profile')
    return 1 if suite in ('policy-startup-envelope', 'policy-startup-trace', 'policy-startup-lifecycle', 'policy-startup-diagnostics') else 2


def lifecycle_budget_for_suite(suite):
    if suite not in SUITES:
        raise ValueError('unknown suite budget profile')
    return {'child_seconds': 330, 'parent_work_seconds': 1080, 'phase_minutes': 40} if suite == 'policy-startup-lifecycle' else None


def startup_envelope_budget_for_suite(suite):
    if suite not in SUITES:
        raise ValueError('unknown suite envelope profile')
    if suite != 'policy-startup-envelope':
        return None
    return {'child_seconds': 210, 'parent_seconds': 480, 'phase_minutes': 18,
            'caller_seconds': 180, 'product_seconds': 90, 'sample_seconds': [30, 75, 110],
            'snapshot_bytes': 65536, 'join_seconds': 5}


def qemu_command(qemu, base, code, variables, port, cpu_count=2):
    if type(cpu_count) is not int or cpu_count not in (1, 2):
        raise ValueError('undeclared native fixture CPU count')
    return [qemu, '-machine', 'q35,accel=tcg', '-cpu', 'qemu64', '-smp', str(cpu_count),
            '-m', '6144', '-snapshot', '-display', 'none', '-monitor', 'none',
            '-serial', 'stdio', '-drive', f'if=pflash,format=raw,readonly=on,file={code}',
            '-drive', f'if=pflash,format=raw,file={variables}',
            '-drive', f'if=virtio,format=qcow2,file={base}',
            '-netdev', f'user,id=net0,hostfwd=tcp:127.0.0.1:{port}-:22',
            '-device', 'virtio-net-pci,netdev=net0']


def ssh_command(key, known, port):
    return ['ssh', '-F', '/dev/null', '-i', str(key), '-p', str(port),
            '-o', 'IdentitiesOnly=yes', '-o', 'IdentityAgent=none',
            '-o', 'BatchMode=yes', '-o', 'ForwardAgent=no',
            '-o', 'StrictHostKeyChecking=accept-new', '-o', f'UserKnownHostsFile={known}',
            '-o', 'ConnectTimeout=5', '-o', 'ServerAliveInterval=15',
            '-o', 'ServerAliveCountMax=4', 'root@127.0.0.1']


def stop_guest(process):
    if process is None:
        return
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=10)
    else:
        process.wait(timeout=10)


def validate_result(suite, log, complete=False):
    if '--- FAIL:' in log or '\nFAIL\n' in log:
        raise ValueError('guest test failure')
    if suite == 'policy-startup-envelope':
        load_tool('check-policy-startup-envelope').validate(log)
        if complete and re.findall('DEVICE_FIXTURE_CLEANUP ([^\n]+)', log) != ['pass']:
            raise ValueError('independent caller-envelope device teardown required')
        return
    if suite == 'policy-startup-trace':
        load_tool('check-policy-startup-trace').validate(log)
        if complete and re.findall('DEVICE_FIXTURE_CLEANUP ([^\n]+)', log) != ['pass']:
            raise ValueError('independent startup trace device teardown required')
        return
    if suite == 'policy-startup-diagnostics':
        load_tool('check-policy-startup-costs').validate(log)
        if complete and re.findall('DEVICE_FIXTURE_CLEANUP ([^\n]+)', log) != ['pass']:
            raise ValueError('independent diagnostic device teardown required')
        return
    if suite == 'policy-startup-lifecycle':
        load_tool('check-policy-root-lifecycle-evidence').validate(log)
        if complete and re.findall('DEVICE_FIXTURE_CLEANUP ([^\n]+)', log) != ['pass']:
            raise ValueError('independent lifecycle device teardown required')
        return
    if suite == 'policy-startup-review':
        load_tool('check-policy-root-review-evidence').validate(log)
        if complete and log.count('DEVICE_FIXTURE_CLEANUP pass\n') != 1:
            raise ValueError('original policy root review device teardown required')
        return
    if suite == 'policy-startup-facts':
        load_tool('check-policy-facts-evidence').validate(log)
        if complete and log.count('DEVICE_FIXTURE_CLEANUP pass\n') != 1:
            raise ValueError('two-pool facts device teardown required')
        return
    if suite == 'executor-recovery':
        load_tool('check-executor-recovery-evidence').validate(log)
        if complete and log.count('DEVICE_FIXTURE_CLEANUP pass\n') != 1:
            raise ValueError('executor research device teardown required')
        return
    if suite == 'action-peer':
        load_tool('check-action-peer-evidence').validate(log)
        return
    if suite == 'job-status':
        load_tool('check-storage-job-status-evidence').validate(log)
        return
    if suite in ('policy-startup-publication', 'policy-startup-owner'):
        load_tool('check-policy-startup-evidence').validate(log, 'publication' if suite.endswith('publication') else 'owner')
        if complete and suite == 'policy-startup-owner' and log.count('DEVICE_FIXTURE_CLEANUP pass\n') != 1:
            raise ValueError('policy startup owner device teardown required')
        return
    if suite in ('closed-startup-publication', 'closed-startup-owner'):
        load_tool('check-closed-startup-evidence').validate(log, 'publication' if suite == 'closed-startup-publication' else 'owner')
        if complete and suite == 'closed-startup-owner' and log.count('DEVICE_FIXTURE_CLEANUP pass\n') != 1:
            raise ValueError('closed startup owner device teardown required')
        return
    if suite == 'execution-startup':
        load_tool('check-execution-startup-evidence').validate(log)
        if complete and log.count('DEVICE_FIXTURE_CLEANUP pass\n') != 1:
            raise ValueError('execution startup device teardown required')
        return
    if suite == 'execution-system':
        load_tool('check-execution-system-evidence').validate(log)
        if complete and log.count('DEVICE_FIXTURE_CLEANUP pass\n') != 1:
            raise ValueError('execution system device teardown required')
        return
    if suite == 'joint-leases':
        load_tool('check-joint-leases-evidence').validate(log)
        return
    if suite == 'original-backing':
        load_tool('check-original-backing-evidence').validate(log)
        return
    if suite == 'persistent-backing':
        load_tool('check-persistent-backing-evidence').validate(log)
        return
    if suite == 'state':
        load_tool('check-state-evidence').validate(log)
        return
    if suite == 'pool-properties':
        load_tool('check-pool-properties-evidence').validate(log)
        if complete and log.count('DEVICE_FIXTURE_CLEANUP pass\n') != 1:
            raise ValueError('focused property device fixture teardown required')
        return
    if suite == 'devices':
        load_tool('check-device-evidence').validate(log, guest=not complete)
        return
    if suite in ('properties', 'performance'):
        load_tool('check-property-evidence' if suite == 'properties' else 'check-performance-evidence').validate(log)
        return
    required = {
        'foundation': ['Native shell and DCO hook checks passed on FreeBSD'],
        'research': ['PASS: generated md unit verified by exact backing-file query',
                     'PASS: TSV fields, numeric units, and GUID preserved through fixture rename',
                     'PASS: native research contracts; production adapter remains unimplemented'],
        'continuity': ['--- PASS: TestNativeIndependentSMBNFSContinuity ',
                       'PASS: actual independent SMB/NFS fixture reads through helper and management outages'],
        'discovery': ['--- PASS: ' + name + ' ' for name in (
            'TestNativeDiscovery', 'TestNativeHelperKernelAuthorization',
            'TestNativeHelperActualReader', 'TestNativeCollectorHelperRecovery',
            'TestNativeCredentialFiles', 'TestNativeForegroundCLIConsole',
            'TestNativeBuiltProcessRestart', 'TestNativeBuiltHelperDenialLogging',
            'TestNativeProbeOwnerDeath', 'TestNativeGuardedProbeBounds',
            'TestNativeBuiltHTTPAbuseAndRecovery')],
    }[suite]
    if any(marker not in log for marker in required):
        raise ValueError('required native assertions did not execute/pass')


def firmware(qemu):
    # Homebrew QEMU and common Linux OVMF locations. No host disk paths.
    share = Path(qemu).resolve().parents[1] / 'share'
    candidates = [(share / 'qemu/edk2-x86_64-code.fd', share / 'qemu/edk2-i386-vars.fd'),
                  (Path('/usr/share/OVMF/OVMF_CODE_4M.fd'), Path('/usr/share/OVMF/OVMF_VARS_4M.fd')),
                  (Path('/usr/share/OVMF/OVMF_CODE.fd'), Path('/usr/share/OVMF/OVMF_VARS.fd'))]
    for code, variables in candidates:
        if code.is_file() and variables.is_file():
            return code, variables
    raise ValueError('QEMU UEFI firmware not found; install QEMU/OVMF before running')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('suite', choices=SUITES)
    parser.add_argument('release', choices=['15.1', '14.5'])
    parser.add_argument('--source-build', action='store_true', help='Compile state tests inside the guest instead of using the host cross-build')
    parser.add_argument('--owned-directory', type=Path, help=argparse.SUPPRESS)
    parser.add_argument('--run', action='store_true', help='Explicitly opt into this disposable guest')
    args = parser.parse_args()
    if args.source_build and args.suite != 'state':
        raise ValueError('--source-build applies only to the state suite')
    if not args.run:
        print(f'{args.suite}: FreeBSD {args.release} amd64, {cpu_count_for_suite(args.suite)} CPUs, 6144 MiB, snapshot, TCG, loopback SSH. Add --run to execute.')
        return
    if platform.system() not in ('Darwin', 'Linux'):
        raise ValueError('local runner requires macOS or Linux')
    for name in ('qemu-system-x86_64', 'unzstd', 'ssh', 'git'):
        if not shutil.which(name):
            raise ValueError(f'missing prerequisite: {name}')
    # Record exactly the committed source; ignored/private host files never sync.
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT):
        raise ValueError('commit changes and use a clean worktree before qualification')
    source = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    vm = load_tool('prepare-performance-vm')
    pins = json.loads((TOOLS / 'performance-vm-pins.json').read_text())
    keypins = json.loads((TOOLS / 'local-vm-pins.json').read_text())
    if keypins['builder'] != pins['builder']:
        raise ValueError('fixture key and image builder mismatch')
    qemu = str(Path(shutil.which('qemu-system-x86_64')).resolve())
    code, vars_template = firmware(qemu)
    runs = ROOT / '.tools/lab-runs'
    runs.mkdir(parents=True, exist_ok=True)
    supervisor = load_tool('owned-process')
    if args.owned_directory is None:
        directory = Path(tempfile.mkdtemp(prefix=f'{args.suite}-{args.release}-', dir=runs))
        directory.chmod(0o700)
        process = supervisor.OwnedProcess(
            [sys.executable, str(Path(__file__).resolve()), args.suite, args.release,
             '--run', '--owned-directory', str(directory)] + (['--source-build'] if args.source_build else []), native_fixture=directory)
        print(f'Owned lab fixture: {directory}; supervisor: {process.directory}', flush=True)
        try:
            status = process.wait()
        finally:
            process.stop()
        result = json.loads((directory / 'result.json').read_text())
        if status or not result.get('qualified') or not process.record()['group_cleaned']:
            raise ValueError('qualification failed or interrupted; inspect private records')
        print('PASS: native assertions, preserved base/runtime and supervised teardown', flush=True)
        return
    directory = supervisor.private_directory(args.owned_directory)
    if directory.parent != runs:
        raise ValueError('worker directory is outside the owned local lab')
    def interrupted(_signum, _frame):
        raise InterruptedError('owned lab interrupted')
    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGINT, interrupted)
    device_profile = 'policy-facts' if args.suite in ('policy-startup-envelope', 'policy-startup-trace', 'policy-startup-facts', 'policy-startup-review', 'policy-startup-lifecycle', 'policy-startup-diagnostics') else 'baseline'
    cpu_count = cpu_count_for_suite(args.suite)
    guest, key = None, directory / 'fixture-key'
    record = {'source_commit': source, 'suite': args.suite, 'release': args.release,
              'started_at': datetime.now(timezone.utc).isoformat(),
              'host_os': platform.system(), 'host_arch': platform.machine(),
              'qemu_sha256': vm.digest(Path(qemu)),
              'qemu_version': subprocess.check_output([qemu, '--version'], text=True).splitlines()[0],
              'firmware_code_sha256': vm.digest(code), 'firmware_vars_sha256': vm.digest(vars_template),
              'cpu_model': 'qemu64', 'cpu_count': cpu_count, 'memory_mib': 6144,
              'accelerator': 'tcg', 'qualified': False, 'host_teardown': False,
              'base_preserved': False,
              'execution_mode': 'host-cross-built' if args.suite in PREBUILT and not args.source_build else 'guest-source-build'}
    if args.suite == 'policy-startup-lifecycle':
        record['lifecycle_fixture_budget'] = lifecycle_budget_for_suite(args.suite)
    if args.suite == 'policy-startup-envelope':
        record['native_acceptance'] = False
        record['qualification_scope'] = 'separate caller-envelope diagnostic completeness only; no lifecycle acceptance'
        record['startup_envelope_budget'] = startup_envelope_budget_for_suite(args.suite)
    if args.suite == 'policy-startup-trace':
        record['native_acceptance'] = False
        record['qualification_scope'] = 'diagnostic completeness only; original operation outcome and lifecycle acceptance separate'
        record['startup_trace_budget'] = {'child_seconds': 150, 'parent_seconds': 480, 'phase_minutes': 18, 'caller_seconds': 120, 'product_seconds': 90, 'sample_seconds': [30, 75, 110], 'snapshot_bytes': 65536, 'join_seconds': 5}
    supervisor.write_record(directory / 'result.json', record)
    base_before = None
    image, variables = directory / 'image', directory / 'vars.fd'
    devices = directory / 'devices'
    device_tool = load_tool('device-fixture') if uses_device_fixture(args.suite) else None
    try:
        if args.suite in PREBUILT and not args.source_build:
            record['prebuilt'] = build_native_binary(directory, source, args.suite)
            supervisor.write_record(directory / 'result.json', record)
        subprocess.run([sys.executable, str(TOOLS / 'prepare-performance-vm.py'), args.release,
                        str(image), '--cache', str(ROOT / '.tools/lab-cache')], check=True)
        record['image'] = json.loads((image / 'image-hashes.json').read_text())
        base = image / 'freebsd' / ('v' + pins['builder']) / ('freebsd-' + args.release + '.qcow2')
        base_before = vm.digest(base)
        shutil.copyfile(vars_template, variables)
        pin = keypins['fixture_keys'][args.release]
        vm.download(pin['url'], pin['sha256'], pin['bytes'], key)
        key.chmod(0o600)
        with socket.socket() as listener:
            listener.bind(('127.0.0.1', 0))
            port = listener.getsockname()[1]
        ssh = ssh_command(key, directory / 'known_hosts', port)
        command = qemu_command(qemu, base, code, variables, port, cpu_count=cpu_count)
        if device_tool:
            device_tool.prepare(devices, profile=device_profile)
            record['device_profile'] = device_profile
            record['device_backings'] = device_tool.verify(devices, profile=device_profile)
            supervisor.write_record(directory / 'result.json', record)
            command = [qemu] + device_tool.command(command[1:], devices, profile=device_profile)
        with (directory / 'serial.log').open('w') as serial:
            guest = subprocess.Popen(command,
                                     stdin=subprocess.DEVNULL, stdout=serial, stderr=serial)
            deadline = time.monotonic() + 900
            while True:
                if guest.poll() is not None:
                    raise ValueError('QEMU exited during boot; inspect private serial.log')
                probe = subprocess.run(ssh + ['true'], capture_output=True, timeout=15)
                if probe.returncode == 0:
                    break
                if time.monotonic() >= deadline:
                    raise ValueError('guest SSH boot deadline expired')
                time.sleep(3)
            archive = subprocess.Popen(['git', 'archive', '--format=tar', source], cwd=ROOT, stdout=subprocess.PIPE)
            try:
                subprocess.run(ssh + ['mkdir -p /root/work && tar -xf - -C /root/work'],
                               stdin=archive.stdout, check=True, timeout=180)
                archive.stdout.close()
                if archive.wait(timeout=10):
                    raise ValueError('source archive failed')
            finally:
                stop_guest(archive)
            artifact_digest = record.get('prebuilt', {}).get('sha256')
            if artifact_digest:
                filename = PREBUILT[args.suite][0]
                with (directory / filename).open('rb') as binary:
                    subprocess.run(ssh + ['umask 077; mkdir -p /root/work/.tools && cat > /root/work/.tools/' + filename + ' && chmod 700 /root/work/.tools/' + filename],
                                   stdin=binary, check=True, timeout=180)
            packages, command = suite_command(args.suite, args.source_build, artifact_digest)
            remote = 'cd /root/work && ' + ('pkg install -y ' + packages + ' && ' if packages else '') + command
            print(f'Running {args.suite} on FreeBSD {args.release}; private logs: {directory}', flush=True)
            with (directory / 'qualification.log').open('w') as log:
                if args.suite == 'executor-recovery':
                    run_executor_recovery(ssh, guest, remote, log)
                else:
                    subprocess.run(ssh + [remote], stdout=log, stderr=subprocess.STDOUT, check=True, timeout=7200)
            validate_result(args.suite, (directory / 'qualification.log').read_text())
            if artifact_digest:
                validate_prebuilt((directory / 'qualification.log').read_text(), artifact_digest, args.suite)
            source_identity(source)
            record['guest_assertions'] = True
    except BaseException as error:
        record['failure'] = type(error).__name__ + ': ' + str(error)
        raise
    finally:
        try:
            stop_guest(guest)
            if device_tool and devices.exists():
                device_tool.cleanup(devices, profile=device_profile)
                record['device_backings_preserved'] = True
                record['device_fixture_teardown'] = not devices.exists()
            if base_before:
                record['base_preserved'] = vm.digest(base) == base_before
                subprocess.run([sys.executable, str(TOOLS / 'prepare-performance-vm.py'), args.release,
                                str(image), '--verify'], check=True)
                if not record['base_preserved']:
                    raise ValueError('base changed; discard this fixture')
            record['runtime_preserved'] = (vm.digest(Path(qemu)) == record['qemu_sha256']
                                           and vm.digest(code) == record['firmware_code_sha256']
                                           and vm.digest(vars_template) == record['firmware_vars_sha256'])
            if not record['runtime_preserved']:
                raise ValueError('QEMU/firmware changed; evidence invalid')
            key.unlink(missing_ok=True)
            if image.exists():
                shutil.rmtree(image)
            variables.unlink(missing_ok=True)
            cleanup_prebuilt(directory)
            record['host_teardown'] = True
            if device_tool and record.get('guest_assertions'):
                if not record.get('device_backings_preserved') or not record.get('device_fixture_teardown'):
                    raise ValueError('device preservation/teardown incomplete')
                with (directory / 'qualification.log').open('a') as log:
                    log.write('DEVICE_FIXTURE_CLEANUP pass\n')
                validate_result(args.suite, (directory / 'qualification.log').read_text(), complete=True)
            record['qualified'] = bool(record.get('guest_assertions') and record['base_preserved']
                                       and record['host_teardown'] and not record.get('failure'))
        except BaseException as error:
            record['qualified'] = False
            record['cleanup_failure'] = type(error).__name__ + ': ' + str(error)
            raise
        finally:
            key.unlink(missing_ok=True)
            cleanup_prebuilt(directory)
            record['completed_at'] = datetime.now(timezone.utc).isoformat()
            (directory / 'result.json').write_text(json.dumps(record, indent=2) + '\n')
            print(f'Qualification record: {directory / "result.json"}', flush=True)
    print('Guest assertions and fixture cleanup complete; supervisor is verifying process teardown', flush=True)


if __name__ == '__main__':
    main()
