# SPDX-License-Identifier: BSD-3-Clause
import hashlib
import importlib.util
import io
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SOURCE = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('performance_vm', SOURCE / 'tools/prepare-performance-vm.py')

class PerformanceVMTests(unittest.TestCase):
    def test_download_rejects_modified_image_without_publishing_it(self):
        module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(module)
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / 'image.zst'
            with patch.object(module.urllib.request, 'urlopen', return_value=io.BytesIO(b'modified')):
                with self.assertRaises(ValueError):
                    module.download('https://example.invalid/image', hashlib.sha256(b'expected').hexdigest(), 8, target)
            self.assertFalse(target.exists())

    def test_verified_download_refuses_overwriting_an_existing_fixture(self):
        module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(module)
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / 'image.zst'
            target.write_bytes(b'owned')
            with self.assertRaises(FileExistsError):
                module.download('https://example.invalid/image', hashlib.sha256(b'expected').hexdigest(), 8, target)
            self.assertEqual(target.read_bytes(), b'owned')

if __name__ == '__main__':
    unittest.main()
