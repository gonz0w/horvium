# SPDX-License-Identifier: BSD-3-Clause
"""Portable control-flow fixtures; never run FreeBSD daemon/pkg or signal a PID."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
FILES = ROOT / 'packaging/ports/horvium/files'


class PortsProfileTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='horvium-port-test-')
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name).resolve()
        self.run = self.root / 'run'
        self.run.mkdir(mode=0o750)
        self.prefix = self.root / 'prefix'
        (self.prefix / 'sbin').mkdir(parents=True)
        (self.prefix / 'etc').mkdir()
        (self.prefix / 'sbin/horvium').touch(mode=0o555)
        (self.prefix / 'etc/horvium.json').write_text('{}\n')
        self.calls = self.root / 'calls.jsonl'
        stub = self.root / 'stub'
        stub.write_text('''#!/usr/bin/env python3
import json,os,pathlib,sys
name=pathlib.Path(sys.argv[0]).name
args=sys.argv[1:]
with open(os.environ['PORT_TEST_CALLS'],'a') as f:f.write(json.dumps([name,*args])+'\\n')
if name=='id': print('0' if args==['-u'] else '521')
elif name=='stat':
 p=pathlib.Path(args[-1]); mode=os.environ.get('PORT_TEST_MODE','')
 if p.is_dir():
  values=['0','521','750','1'] if p.name=='run' else ['0','0',os.environ.get('PORT_TEST_ANCESTOR_MODE','755'),'1']
  count=len(args[1].split(':'));print(':'.join([values[0],values[2]]) if count==2 else ':'.join(values[:count]))
 elif p.name.endswith('.pid'):print('0:0:'+ (mode or '600') +':1')
 else:print('0:521:640:1')
elif name=='daemon':pathlib.Path(args[args.index('-P')+1]).write_text('4242\\n')
elif name=='kill':
 p=pathlib.Path(os.environ['PORT_TEST_PIDFILE'])
 if args[0]=='-TERM' and os.environ.get('PORT_TEST_STUCK')!='1':p.unlink()
 elif args[0]=='-0':sys.exit(0 if p.exists() else 1)
''')
        stub.chmod(0o755)
        common = (FILES / 'horvium_rc.subr.in').read_text()
        common = common.replace('%%PREFIX%%', str(self.prefix))
        common = common.replace('/var/run/horvium', str(self.run))
        for command in ['id', 'stat', 'daemon', 'kill', 'sleep']:
            path = self.root / command
            path.symlink_to(stub)
            common = common.replace('/usr/bin/' + command, str(path))
            common = common.replace('/usr/sbin/' + command, str(path))
            common = common.replace('/bin/' + command, str(path))
        self.common = self.root / 'rc.subr'
        self.common.write_text(common)

    def invoke(self, operation, *, role='serve', pid=False, **updates):
        pidfile = self.run / ('helper.pid' if role == 'helper' else 'management.pid')
        if pid:
            pidfile.write_text('4242\n')
            pidfile.chmod(0o600)
        env = dict(os.environ, PORT_TEST_CALLS=str(self.calls),
                   PORT_TEST_PIDFILE=str(pidfile), **updates)
        script = '''name=horvium
hv_mode="$1"
check_pidfile() { [ "${PORT_TEST_MATCH:-1}" = 1 ] && [ -f "$1" ] && printf '4242\\n'; }
. "$2"
horvium_user=_horvium
horvium_stop_wait=2
"$3"
'''
        result = subprocess.run(['/bin/sh', '-c', script, 'fixture', role,
                                 str(self.common), operation], env=env,
                                capture_output=True, text=True, timeout=8)
        calls = [json.loads(line) for line in self.calls.read_text().splitlines()] if self.calls.exists() else []
        return result, calls

    def test_launch_uses_only_fixed_program_and_expected_identity(self):
        result, calls = self.invoke('horvium_start')
        self.assertEqual(result.returncode, 0, result.stderr)
        launch = next(c for c in calls if c[0] == 'daemon')
        self.assertIn('-u', launch)
        self.assertIn('_horvium', launch)
        self.assertEqual(launch[-3:], ['serve', '-config', str(self.prefix / 'etc/horvium.json')])
        self.assertNotIn('-r', launch)
        self.assertNotIn('-R', launch)

    def test_helper_launch_keeps_root_and_separate_pidfile(self):
        result, calls = self.invoke('horvium_start', role='helper')
        self.assertEqual(result.returncode, 0, result.stderr)
        launch = next(c for c in calls if c[0] == 'daemon')
        self.assertNotIn('-u', launch)
        self.assertIn(str(self.run / 'helper.pid'), launch)
        self.assertIn('helper', launch)

    def test_writable_ancestor_prevents_privileged_pidfile_creation(self):
        for mode in ['777', '2755', '4755']:
            with self.subTest(mode=mode):
                result, calls = self.invoke('horvium_start', PORT_TEST_ANCESTOR_MODE=mode)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(c[0] == 'daemon' for c in calls))
                self.calls.unlink(missing_ok=True)

    def test_package_guards_preserve_unknown_active_state(self):
        for hook in ['pkg-pre-install.in', 'pkg-pre-deinstall.in']:
            script = (FILES / hook).read_text().replace('/var/run/horvium', str(self.run))
            inactive = subprocess.run(['/bin/sh', '-c', script], capture_output=True, timeout=4)
            self.assertEqual(inactive.returncode, 0)
            unknown = self.run / 'inventory.sock'
            unknown.symlink_to(self.root / 'absent')
            active = subprocess.run(['/bin/sh', '-c', script], capture_output=True, timeout=4)
            self.assertNotEqual(active.returncode, 0)
            self.assertTrue(unknown.is_symlink())
            unknown.unlink()

    def test_unsafe_command_prefixes_are_rejected(self):
        for operation in ['forcestart', 'faststop', 'reload', 'enable', 'delete']:
            result = subprocess.run(['/bin/sh', '-c', '. "$1"; horvium_command_allowed "$2"',
                                     'fixture', str(self.common), operation],
                                    env=dict(os.environ, name='horvium', hv_mode='serve'),
                                    capture_output=True, timeout=4)
            self.assertNotEqual(result.returncode, 0)

    def test_existing_unknown_or_symlink_pidfile_blocks_start(self):
        for symlink in [False, True]:
            with self.subTest(symlink=symlink):
                p = self.run / 'management.pid'
                if symlink:
                    p.symlink_to(self.root / 'absent')
                else:
                    p.write_text('4242\n')
                result, calls = self.invoke('horvium_start')
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(c[0] == 'daemon' for c in calls))
                self.assertTrue(p.exists() or p.is_symlink())
                p.unlink()

    def test_stop_targets_only_validated_supervisor(self):
        result, calls = self.invoke('horvium_stop', pid=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual([c for c in calls if c[:2] == ['kill', '-TERM']], [['kill', '-TERM', '4242']])

    def test_unsafe_or_unmatched_pidfile_never_signals(self):
        for updates in [{'PORT_TEST_MODE': '666'}, {'PORT_TEST_MATCH': '0'}]:
            with self.subTest(updates=updates):
                result, calls = self.invoke('horvium_stop', pid=True, **updates)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(c[0] == 'kill' for c in calls))
                self.calls.unlink(missing_ok=True)

    def test_uncertain_stop_does_not_escalate_or_restart(self):
        result, calls = self.invoke('horvium_restart', pid=True, PORT_TEST_STUCK='1')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual([c for c in calls if c[0] == 'kill' and c[1] != '-0'], [['kill', '-TERM', '4242']])
        self.assertFalse(any(c[0] == 'daemon' for c in calls))
        self.assertTrue((self.run / 'management.pid').exists())


if __name__ == '__main__':
    unittest.main()
