#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Adopted separate diagnostic profile; no lifecycle acceptance."""
def validate(log, build, complete=False):
    raise NotImplementedError('provenance runner pending')

def command(build):
    raise NotImplementedError('provenance runner pending')

def check_build(build):
    raise NotImplementedError('provenance build isolation pending')
