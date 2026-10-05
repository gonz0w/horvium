#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
import argparse
from pathlib import Path
import re

CASES={'distinct_serial','duplicate_serial','missing_serial','sector_variants',
       'diskinfo_correlation','backing_relationships','boot_exclusion','read_preservation'}

def validate(text, guest=False):
    if len(text)>1<<20 or re.search(r'(?m)^(?:FAIL|--- (?:FAIL|SKIP):)',text):
        raise ValueError('native device evidence failed, skipped or oversized')
    passes=re.findall(r'(?m)^--- PASS: TestNativeDeviceFixtures \([0-9.]+s\)$',text)
    records=re.findall(r'DEVICE_OBSERVATION_RESULT ([^\n]+)',text)
    if len(passes)!=1 or len(records)!=1:
        raise ValueError('one executed native device result required')
    fields=records[0].split()
    expected={case+'=pass' for case in CASES}
    if len(fields)!=len(expected) or set(fields)!=expected:
        raise ValueError('missing, duplicate, unknown or failed device assertion')
    if not guest and len(re.findall(r'(?m)^DEVICE_FIXTURE_CLEANUP pass$',text))!=1:
        raise ValueError('complete owned host fixture teardown required')

if __name__=='__main__':
    parser=argparse.ArgumentParser()
    parser.add_argument('--guest',action='store_true')
    parser.add_argument('log',type=Path)
    args=parser.parse_args();validate(args.log.read_text(),guest=args.guest)
