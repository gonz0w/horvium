#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Strict scalar transcript consumer; never establishes native acceptance."""
import importlib.util
from pathlib import Path


def load(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


envelope = load('check-policy-startup-envelope')
facts = load('check-policy-facts-evidence')
CHILD_EXTRA = set('trace_enabled error_preserved snapshots_before_cancel trace_inventory original_contexts'.split())
PARENT_EXTRA = set('trace_extracted trace_sanitized source_isolated'.split())
CHILD = envelope.CHILD | CHILD_EXTRA
ROOT = envelope.ROOT | {'child_' + name for name in CHILD_EXTRA} | PARENT_EXTRA


def validate(text, *, source_commit, compiler_sha256, artifact_sha256):
    raise ValueError('strict provenance consumer not implemented')
