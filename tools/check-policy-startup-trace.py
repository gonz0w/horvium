#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Diagnostic completeness only; an original startup error is not acceptance."""
CHILD = set('cpu_affinity child_budget caller_deadline original_call sampler_joined samples_bounded owner_release deadline_unchanged'.split())
ROOT = set('cpu_affinity parent_budget seed_closed child_waited child_inventory snapshots_verified final_journal_observed ledger_preserved nonjournal_preserved child_locks_released raw_removed domain_teardown mounts_restored'.split()) | {'child_' + x for x in CHILD}

def validate(text):
    raise ValueError('startup diagnostic consumer not implemented')
