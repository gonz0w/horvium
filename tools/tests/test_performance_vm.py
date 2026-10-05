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
    def module(self):
        module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(module)
        return module

    def test_cached_artifact_is_reverified_and_reused_without_network(self):
        module = self.module()
        with tempfile.TemporaryDirectory() as directory:
            cache = Path(directory) / 'cache'
            cache.mkdir()
            expected = hashlib.sha256(b'image').hexdigest()
            (cache / expected).write_bytes(b'image')
            target = Path(directory) / 'fresh.zst'
            with patch.object(module.urllib.request, 'urlopen', side_effect=AssertionError('network')):
                module.download('https://example.invalid/image', expected, 5, target, cache)
            self.assertEqual(target.read_bytes(), b'image')
            self.assertNotEqual(target.stat().st_ino, (cache / expected).stat().st_ino)

    def test_corrupt_cache_is_rejected_without_publishing_a_fixture(self):
        module = self.module()
        with tempfile.TemporaryDirectory() as directory:
            cache = Path(directory) / 'cache'
            cache.mkdir()
            expected = hashlib.sha256(b'image').hexdigest()
            (cache / expected).write_bytes(b'wrong')
            target = Path(directory) / 'fresh.zst'
            with self.assertRaises(ValueError):
                module.download('https://example.invalid/image', expected, 5, target, cache)
            self.assertFalse(target.exists())

    def test_new_cache_entry_is_authenticated_before_reuse(self):
        module = self.module()
        with tempfile.TemporaryDirectory() as directory:
            cache = Path(directory) / 'cache'
            cache.mkdir()
            expected = hashlib.sha256(b'image').hexdigest()
            target = Path(directory) / 'fresh.zst'
            with patch.object(module.urllib.request, 'urlopen', return_value=io.BytesIO(b'wrong')):
                with self.assertRaises(ValueError):
                    module.download('https://example.invalid/image', expected, 5, target, cache)
            self.assertFalse(target.exists())
            self.assertEqual(list(cache.iterdir()), [])

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
