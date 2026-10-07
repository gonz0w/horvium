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
    'foundation': ('git', 'sh tools/bsd-foundation-smoke.sh FreeBSD'),
    'research': ('python314', 'env HORVIUM_RESEARCH_VM=1 sh tools/research/m1-native.sh'),
    'discovery': ('python314', 'env HORVIUM_NATIVE_VM=1 sh tools/bsd-native-discovery.sh'),
    'devices': ('python314', 'env HORVIUM_NATIVE_VM=1 HORVIUM_DEVICE_VM=1 sh tools/bsd-devices.sh'),
    'state': ('python314', 'env HORVIUM_NATIVE_VM=1 HORVIUM_STATE_VM=1 sh tools/bsd-state.sh'),
    'properties': ('python314', 'env HORVIUM_NATIVE_VM=1 HORVIUM_PROPERTY_VM=1 sh tools/bsd-properties.sh'),
    'pool-properties': ('', 'env HORVIUM_NATIVE_VM=1 HORVIUM_PROPERTY_VM=1 sh tools/bsd-pool-properties-prebuilt.sh'),
    'continuity': ('python314 samba423', 'env HORVIUM_NATIVE_VM=1 HORVIUM_CONTINUITY_VM=1 sh tools/bsd-continuity.sh'),
    'performance': ('python314', 'env HORVIUM_NATIVE_VM=1 sh tools/bsd-native-discovery.sh && env HORVIUM_NATIVE_VM=1 HORVIUM_PERF_VM=1 sh tools/bsd-performance.sh'),
}

PREBUILT = {
    'state': ('state-native.test', './internal/storage', 'STATE_PREBUILT_SHA256', 'HORVIUM_STATE_VM=1', 'bsd-state-prebuilt.sh'),
    'pool-properties': ('pool-properties-native.test', './internal/host/freebsd', 'POOL_PROPERTIES_PREBUILT_SHA256', 'HORVIUM_PROPERTY_VM=1', 'bsd-pool-properties-prebuilt.sh'),
}


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
    command = [go, 'test', '-c', '-trimpath', '-o', str(binary), package]
    source_identity(source)
    subprocess.run(command, cwd=ROOT, env=env, check=True)
    source_identity(source)
    return {'source_commit': source, 'sha256': plan.digest(binary), 'bytes': binary.stat().st_size,
            'go_version': version, 'compiler_sha256': plan.digest(Path(go)),
            'go_pins_sha256': plan.digest(TOOLS / 'go-pins.json'),
            'go_mod_sha256': plan.digest(ROOT / 'go.mod'), 'go_sum_sha256': plan.digest(ROOT / 'go.sum'),
            'target': 'freebsd/amd64', 'goamd64': 'v1', 'cgo_enabled': False,
            'build_arguments': ['go', 'test', '-c', '-trimpath', '-o', '<owned-artifact>', package]}


def validate_prebuilt(log, artifact_digest, suite='state'):
    marker = PREBUILT[suite][2]
    if re.findall(r'(?m)^' + marker + r' ([a-f0-9]{64})$', log) != [artifact_digest]:
        raise ValueError('guest did not verify the expected executable')


def qemu_command(qemu, base, code, variables, port):
    return [qemu, '-machine', 'q35,accel=tcg', '-cpu', 'qemu64', '-smp', '2',
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
    if suite == 'state':
        load_tool('check-state-evidence').validate(log)
        return
    if suite == 'pool-properties':
        load_tool('check-pool-properties-evidence').validate(log)
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
        print(f'{args.suite}: FreeBSD {args.release} amd64, 2 CPUs, 6144 MiB, snapshot, TCG, loopback SSH. Add --run to execute.')
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
    guest, key = None, directory / 'fixture-key'
    record = {'source_commit': source, 'suite': args.suite, 'release': args.release,
              'started_at': datetime.now(timezone.utc).isoformat(),
              'host_os': platform.system(), 'host_arch': platform.machine(),
              'qemu_sha256': vm.digest(Path(qemu)),
              'qemu_version': subprocess.check_output([qemu, '--version'], text=True).splitlines()[0],
              'firmware_code_sha256': vm.digest(code), 'firmware_vars_sha256': vm.digest(vars_template),
              'cpu_model': 'qemu64', 'cpu_count': 2, 'memory_mib': 6144,
              'accelerator': 'tcg', 'qualified': False, 'host_teardown': False,
              'base_preserved': False,
              'execution_mode': 'host-cross-built' if args.suite in PREBUILT and not args.source_build else 'guest-source-build'}
    supervisor.write_record(directory / 'result.json', record)
    base_before = None
    image, variables = directory / 'image', directory / 'vars.fd'
    devices = directory / 'devices'
    device_tool = load_tool('device-fixture') if args.suite == 'devices' else None
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
        command = qemu_command(qemu, base, code, variables, port)
        if device_tool:
            device_tool.prepare(devices)
            record['device_backings'] = device_tool.verify(devices)
            supervisor.write_record(directory / 'result.json', record)
            command = [qemu] + device_tool.command(command[1:], devices)
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
                device_tool.cleanup(devices)
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
            (directory / 'state-native.test').unlink(missing_ok=True)
            (directory / 'pool-properties-native.test').unlink(missing_ok=True)
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
            (directory / 'state-native.test').unlink(missing_ok=True)
            (directory / 'pool-properties-native.test').unlink(missing_ok=True)
            record['completed_at'] = datetime.now(timezone.utc).isoformat()
            (directory / 'result.json').write_text(json.dumps(record, indent=2) + '\n')
            print(f'Qualification record: {directory / "result.json"}', flush=True)
    print('Guest assertions and fixture cleanup complete; supervisor is verifying process teardown', flush=True)


if __name__ == '__main__':
    main()
