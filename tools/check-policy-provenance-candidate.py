#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Offline runner-input verification; never establishes native acceptance."""


def verify_source(root, source):
    raise NotImplementedError('candidate source verification pending')


def verify_candidate(root, transcript, compiler, artifact, *, source_commit,
                     compiler_sha256, artifact_sha256):
    raise NotImplementedError('candidate file verification pending')
