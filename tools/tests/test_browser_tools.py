# SPDX-License-Identifier: BSD-3-Clause
import hashlib
import importlib.util
import tempfile
import unittest
from unittest import mock
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('bootstrap_browser', ROOT / 'tools/bootstrap-browser.py')

class BrowserToolTests(unittest.TestCase):
    def module(self):
        module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(module)
        return module

    def test_checksum_mismatch_rejected(self):
        tool = self.module()
        tool.verify(b'fixture', hashlib.sha256(b'fixture').hexdigest())
        with self.assertRaises(ValueError):
            tool.verify(b'changed', hashlib.sha256(b'fixture').hexdigest())

    def test_existing_install_is_preserved_without_network(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory)
            sentinel = target / 'owned'
            sentinel.write_text('preserve')
            with self.assertRaises(ValueError):
                self.module().install({}, target)
            self.assertEqual(sentinel.read_text(), 'preserve')

class BrowserMeasurementTests(unittest.TestCase):
    def test_known_contrast_and_alpha_rejection(self):
        spec = importlib.util.spec_from_file_location('browser_qualification', ROOT / 'tools/browser-qualification.py')
        tool = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(tool)
        self.assertAlmostEqual(tool.contrast((0, 0, 0), (255, 255, 255)), 21)
        self.assertEqual(tool.contrast((25, 25, 25), (25, 25, 25)), 1)
        with self.assertRaises(AssertionError):
            tool.rgb('rgba(0, 0, 0, 0.5)')

class BrowserCleanupTests(unittest.TestCase):
    def test_failed_session_delete_still_stops_owned_driver(self):
        spec = importlib.util.spec_from_file_location('browser_cleanup', ROOT / 'tools/browser-qualification.py')
        tool = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(tool)
        driver = mock.Mock(session='synthetic')
        driver.command.side_effect = RuntimeError('fixture transport failed')
        process = object()
        with mock.patch.object(tool, 'stop_process') as stop:
            with self.assertRaises(RuntimeError):
                tool.close_driver(driver, process)
            stop.assert_called_once_with(process)

if __name__ == '__main__':
    unittest.main()
