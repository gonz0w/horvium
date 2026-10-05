#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
import argparse
from pathlib import Path
import re

CASES={'distinct_serial','duplicate_serial','missing_serial','sector_variants',
       'diskinfo_correlation','backing_relationships','boot_exclusion','read_preservation'}
CORRELATION_CASES={'unique_serial','duplicate_denial','missing_denial','geometry_binding'}
ATTACHMENT_CASES={'native_alias','partition_denial','blank_table_denial','fixture_teardown','foreign_preservation'}
SYSTEM_USE_CASES={'root_backing','mounted_disk','active_swap','unsupported_denial','fixture_teardown','foreign_preservation'}

IMPORTED_POOL_CASES={'unmounted_pool','catalogue_binding','missing_denial','fixture_teardown','foreign_preservation','root_preservation'}

SIGNATURE_UFS_CASES={'ufs_signature','probe_preservation','unsupported_denial'}
SIGNATURE_ZFS_CASES={'zfs_signature','removed_signature','probe_preservation','binding_denial'}

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
    correlation=re.findall(r'DEVICE_CORRELATION_RESULT ([^\n]+)',text)
    if len(correlation)!=1:
        raise ValueError('one executed device correlation result required')
    fields=correlation[0].split()
    expected={case+'=pass' for case in CORRELATION_CASES}
    if len(fields)!=len(expected) or set(fields)!=expected:
        raise ValueError('missing, duplicate, unknown or failed correlation assertion')
    passes=re.findall(r'(?m)^--- PASS: TestNativeDiskAttachments \([0-9.]+s\)$',text)
    records=re.findall(r'DEVICE_ATTACHMENT_RESULT ([^\n]+)',text)
    if len(passes)!=1 or len(records)!=1:
        raise ValueError('one executed native attachment result required')
    fields=records[0].split()
    expected={case+'=pass' for case in ATTACHMENT_CASES}
    if len(fields)!=len(expected) or set(fields)!=expected:
        raise ValueError('missing, duplicate, unknown or failed attachment assertion')
    passes=re.findall(r'(?m)^--- PASS: TestNativeSystemDeviceUse \([0-9.]+s\)$',text)
    records=re.findall(r'DEVICE_SYSTEM_USE_RESULT ([^\n]+)',text)
    if len(passes)!=1 or len(records)!=1:
        raise ValueError('one executed native system-use result required')
    fields=records[0].split()
    expected={case+'=pass' for case in SYSTEM_USE_CASES}
    if len(fields)!=len(expected) or set(fields)!=expected:
        raise ValueError('missing, duplicate, unknown or failed system-use assertion')
    passes=re.findall(r'(?m)^--- PASS: TestNativeImportedPoolUse \([0-9.]+s\)$',text)
    records=re.findall(r'DEVICE_IMPORTED_POOL_RESULT ([^\n]+)',text)
    if len(passes)!=1 or len(records)!=1:
        raise ValueError('one executed native imported-pool result required')
    fields=records[0].split()
    expected={case+'=pass' for case in IMPORTED_POOL_CASES}
    if len(fields)!=len(expected) or set(fields)!=expected:
        raise ValueError('missing, duplicate, unknown or failed imported-pool assertion')
    for marker, cases in [('DEVICE_SIGNATURE_UFS_RESULT', SIGNATURE_UFS_CASES),
                          ('DEVICE_SIGNATURE_ZFS_RESULT', SIGNATURE_ZFS_CASES)]:
        records=re.findall(marker+r' ([^\n]+)',text)
        if len(records)!=1:
            raise ValueError('one complete native signature result required')
        fields=records[0].split()
        expected={case+'=pass' for case in cases}
        if len(fields)!=len(expected) or set(fields)!=expected:
            raise ValueError('missing, duplicate, unknown or failed signature assertion')
    if not guest and len(re.findall(r'(?m)^DEVICE_FIXTURE_CLEANUP pass$',text))!=1:
        raise ValueError('complete owned host fixture teardown required')

if __name__=='__main__':
    parser=argparse.ArgumentParser()
    parser.add_argument('--guest',action='store_true')
    parser.add_argument('log',type=Path)
    args=parser.parse_args();validate(args.log.read_text(),guest=args.guest)
