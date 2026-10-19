#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Short-lived lease supervisor for foreground development process groups."""
import argparse
import hashlib
import json
import os
import selectors
import signal
import socket
import stat
import subprocess
import sys
import tempfile
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RUNS = ROOT / '.tools/process-runs'


def private_directory(path):
    path = Path(path)
    info = path.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        raise ValueError('expected an owned private directory, without symlinks')
    if path.resolve() != path.absolute():
        raise ValueError('symlinked parent directory refused')
    return path


def write_record(path, record):
    temporary = path.with_suffix('.tmp')
    with temporary.open('w') as output:
        os.chmod(temporary, 0o600)
        json.dump(record, output, indent=2)
        output.write('\n')
    temporary.replace(path)


def live_members(group, deadline):
    # Read only group/state columns, never persist a host inventory. The leader
    # remains unreaped until cleanup, reserving its PID against reuse.
    remaining = deadline - time.monotonic()
    if remaining <= 0:
        raise TimeoutError('owned group observation deadline')
    output = subprocess.check_output(['ps', '-ax', '-o', 'pgid=', '-o', 'stat='],
                                     text=True, timeout=min(0.25, remaining))
    rows = [line.split() for line in output.splitlines() if line.strip()]
    if not rows or any(len(fields) != 2 for fields in rows):
        raise ValueError('incomplete group observation')
    members = [(int(fields[0]), fields[1]) for fields in rows]
    return any(pgid == group and not state.startswith('Z') for pgid, state in members)


def stop_group(process, grace, deadline, failures):
    """Signal only the fresh, unreaped launcher group; never adopt a receipt PID.

    Observation failure retains debt but cannot suppress termination. Detached
    sessions are outside this group proof and need independent containment.
    """
    escalated = False
    def observed():
        try:
            return live_members(process.pid, deadline)
        except Exception as error:
            failures.append('group observation: ' + type(error).__name__)
            return None
    if observed() is False:
        return True, escalated
    for sig, seconds in ((signal.SIGTERM, grace), (signal.SIGKILL, 5)):
        try:
            os.killpg(process.pid, sig)
        except ProcessLookupError:
            pass
        except PermissionError:
            # macOS can reject a signal to a group containing only zombies.
            if observed() is False:
                return not failures, escalated
            failures.append('group signal: PermissionError')
        except OSError as error:
            failures.append('group signal: ' + type(error).__name__)
        phase_end = min(deadline, time.monotonic() + seconds)
        while time.monotonic() < phase_end:
            active = observed()
            if active is False:
                return not failures, escalated
            if active is None:
                break # Failed observer cannot delay the independent KILL.
            time.sleep(0.05)
        escalated = True
    return False, escalated


def native_cleanup(directory, interrupted):
    """Fallback touches only fixed names in our private disposable fixture."""
    import shutil
    directory = private_directory(directory)
    if directory.parent != ROOT / '.tools/lab-runs':
        raise ValueError('native cleanup outside local lab refused')
    device_error = None
    try:
        devices = directory / 'devices'
        if devices.exists() or devices.is_symlink():
            if devices.is_symlink():
                devices.unlink()
            else:
                import importlib.util
                spec = importlib.util.spec_from_file_location('device_fixture', ROOT / 'tools/device-fixture.py')
                module = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(module)
                module.remove(devices)
    except Exception as error:
        device_error = str(error)
    for name in ('fixture-key', 'vars.fd', 'state-native.test', 'pool-properties-native.test', 'storage-job-status-native.test', 'action-peer-native.test', 'policy-startup-envelope-native.test', 'policy-startup-lifecycle-native.test', 'policy-startup-provenance-native.test', 'policy-startup-provenance-normal.test'):
        (directory / name).unlink(missing_ok=True)
    image = directory / 'image'
    if image.is_symlink():
        image.unlink()
    elif image.exists():
        shutil.rmtree(image)
    # The dashboard candidate is deterministically paired with this fresh
    # performance lab directory. Never follow a record-supplied deletion path.
    if directory.name.startswith('performance-'):
        candidate = ROOT / '.tools/dashboard-candidates' / directory.name
        if candidate.exists() or candidate.is_symlink():
            private_directory(candidate)
            for name in ('source.tar', 'horvium', 'reader-fixture.test', 'dashboard-native.test'):
                (candidate / name).unlink(missing_ok=True)
            build_record = candidate / 'result.json'
            if build_record.exists():
                build_state = json.loads(build_record.read_text())
                build_state['artifacts_removed'] = True
                write_record(build_record, build_state)
    result = directory / 'result.json'
    record = json.loads(result.read_text()) if result.exists() else {}
    if interrupted:
        record['qualified'] = False
        # The worker may have completed its own failure/cleanup record before
        # exiting nonzero. Preserve that original diagnostic during fallback.
        if not isinstance(record.get('failure'), str) or not record['failure']:
            record['failure'] = 'supervisor interrupted the run or worker failed'
    if device_error:
        record.update(qualified=False, cleanup_failure=device_error)
    record['supervisor_fixture_cleanup'] = True
    record['supervisor_group_cleaned'] = True
    write_record(result, record)
    if device_error:
        raise ValueError('device fixture cleanup failed: ' + device_error)


def supervise(directory):
    directory = private_directory(directory)
    launch = directory / 'launch.json'
    config = json.loads(launch.read_text())
    launch.unlink()
    record = {'completed': False, 'group_cleaned': False,
              'command_sha256': hashlib.sha256(json.dumps(config['argv']).encode()).hexdigest()}
    control_path = directory / 'control.sock'
    reason = None
    def interrupted(_signum, _frame):
        nonlocal reason
        reason = 'supervisor-signal'
    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGINT, interrupted)
    process = None
    reserved = False
    try:
        with socket.socket(socket.AF_UNIX) as control, selectors.DefaultSelector() as selector:
            control.bind(str(control_path))
            os.chmod(control_path, 0o600)
            control.listen(4)
            selector.register(control, selectors.EVENT_READ, 'control')
            selector.register(sys.stdin.buffer, selectors.EVENT_READ, 'lease')
            process = subprocess.Popen(config['argv'], cwd=config['cwd'], stdin=subprocess.DEVNULL,
                                       start_new_session=True)
            reserved = True
            record.update(group=process.pid, started_at=time.time())
            write_record(directory / 'state.json', record)
            deadline = time.monotonic() + config['timeout']
            while reason is None:
                try:
                    exited = os.waitid(os.P_PID, process.pid, os.WEXITED | os.WNOHANG | os.WNOWAIT)
                except ChildProcessError:
                    reserved = False # A consumed/lost wait cannot authorize signals.
                    raise
                if exited:
                    reason = 'exit'
                    break
                if time.monotonic() >= deadline:
                    reason = 'timeout'
                    break
                for key, _events in selector.select(0.1):
                    if key.data == 'lease':
                        if not os.read(sys.stdin.fileno(), 1):
                            reason = 'owner-disconnected'
                    else:
                        with control.accept()[0] as connection:
                            connection.settimeout(1)
                            if connection.recv(32) == b'stop\n':
                                reason = 'operator-stop'
                                connection.sendall(b'accepted\n')
    finally:
        failures = []
        # One original cleanup end, including observations and the sole reap.
        cleanup_end = time.monotonic() + config['grace'] + 10
        if process is not None:
            if reserved:
                try:
                    record['group_cleaned'], record['escalated'] = stop_group(
                        process, config['grace'], cleanup_end - 5, failures)
                except BaseException as error:
                    failures.append('group cleanup: ' + type(error).__name__)
                    # A callback/observer failure must not suppress the final
                    # owned signal. The fresh leader has not been reaped here.
                    try:
                        os.killpg(process.pid, signal.SIGKILL)
                    except ProcessLookupError:
                        pass
                    except BaseException as kill_error:
                        failures.append('fallback kill: ' + type(kill_error).__name__)
                try:
                    record['worker_returncode'] = process.wait(timeout=max(.001, cleanup_end - time.monotonic()))
                except BaseException as error:
                    failures.append('sole wait: ' + type(error).__name__)
            else:
                failures.append('leader reservation lost; no signal or second wait')
        else:
            # Startup failed before a worker/group existed. Preserve failure,
            # but do not invent a group to kill or strand prepared native files.
            record['group_cleaned'] = True
        record.update(reason=reason or 'supervisor-failure', completed=True, completed_at=time.time())
        if failures:
            record.update(group_cleaned=False, cleanup_failures=failures)
        record['scope'] = 'launcher-group-only; detached sessions require independent containment'
        if config.get('native_fixture') and record['group_cleaned']:
            try:
                native_cleanup(Path(config['native_fixture']), reason != 'exit' or record.get('worker_returncode') != 0)
            except BaseException as error:
                record['cleanup_failure'] = str(error)
                record['group_cleaned'] = False
        try:
            control_path.unlink(missing_ok=True)
        except BaseException as error:
            record['group_cleaned'] = False
            record.setdefault('cleanup_failures', []).append('control unlink: ' + type(error).__name__)
        write_record(directory / 'state.json', record)
    return record.get('worker_returncode', 1) if reason == 'exit' and record['group_cleaned'] else 1


def request_stop(directory):
    directory = private_directory(directory)
    state = directory / 'state.json'
    if state.exists() and json.loads(state.read_text()).get('completed'):
        return json.loads(state.read_text()).get('group_cleaned', False)
    path = directory / 'control.sock'
    info = path.lstat()
    if not stat.S_ISSOCK(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        raise ValueError('untrusted control socket')
    with socket.socket(socket.AF_UNIX) as connection:
        connection.settimeout(2)
        connection.connect(str(path))
        connection.sendall(b'stop\n')
        return connection.recv(32) == b'accepted\n'


class OwnedProcess:
    def __init__(self, argv, *, cwd=ROOT, env=None, stdout=None, stderr=None,
                 runs=RUNS, timeout=9000, grace=15, native_fixture=None):
        runs = Path(runs).absolute()
        runs.mkdir(parents=True, exist_ok=True, mode=0o700)
        private_directory(runs)
        self.directory = Path(tempfile.mkdtemp(prefix='', dir=runs))
        config = {'argv': argv, 'cwd': str(cwd), 'timeout': timeout, 'grace': grace,
                  'native_fixture': str(native_fixture) if native_fixture else None}
        write_record(self.directory / 'launch.json', config)
        self.process = subprocess.Popen([sys.executable, str(Path(__file__).resolve()), '_supervise', str(self.directory)],
                                        stdin=subprocess.PIPE, stdout=stdout, stderr=stderr,
                                        env=env, start_new_session=True)

    def poll(self):
        return self.process.poll()

    def wait(self, timeout=None):
        result = self.process.wait(timeout=timeout)
        self.process.stdin.close()
        return result

    def record(self):
        return json.loads((self.directory / 'state.json').read_text())

    def stop(self):
        self.process.stdin.close()
        self.process.wait(timeout=35)
        if not self.record()['group_cleaned']:
            raise RuntimeError('owned process cleanup failed; inspect supervisor record')


def manage(stop=False):
    failed = False
    if not RUNS.exists():
        print('No owned development sessions')
        return 0
    for directory in sorted(RUNS.iterdir()):
        try:
            private_directory(directory)
            path = directory / 'state.json'
            record = json.loads(path.read_text()) if path.exists() else {}
            if stop and not record.get('completed'):
                request_stop(directory)
                deadline = time.monotonic() + 35
                while time.monotonic() < deadline:
                    record = json.loads(path.read_text()) if path.exists() else {}
                    if record.get('completed'):
                        break
                    time.sleep(0.1)
            completed = record.get('completed')
            if completed and record.get('group_cleaned'):
                print(f'{directory.name}: stopped; teardown verified')
            elif completed:
                raise ValueError('cleanup failed')
            else:
                # A responsive socket is required; stale metadata is never a
                # license to signal a PID from a prior run.
                with socket.socket(socket.AF_UNIX) as probe:
                    probe.settimeout(1)
                    probe.connect(str(directory / 'control.sock'))
                print(f'{directory.name}: running; group {record.get("group", "starting")}')
                failed = True
        except (OSError, ValueError) as error:
            print(f'{directory.name}: UNVERIFIED ({error}); no stale PID signalled', file=sys.stderr)
            failed = True
    return int(failed)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=['_supervise', 'run', 'status', 'stop'])
    parser.add_argument('argv', nargs=argparse.REMAINDER)
    args = parser.parse_args()
    if args.mode == '_supervise':
        return supervise(Path(args.argv[0]))
    if args.mode in ('status', 'stop'):
        return manage(args.mode == 'stop')
    argv = args.argv[1:] if args.argv[:1] == ['--'] else args.argv
    if not argv:
        parser.error('run requires a command after --')
    process = OwnedProcess(argv)
    try:
        return process.wait()
    finally:
        process.stop()


if __name__ == '__main__':
    sys.exit(main())
