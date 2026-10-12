#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Diagnostic completeness only; an original startup error is not acceptance."""
import importlib.util
import json
from pathlib import Path
import re
import sys
CHILD = set('cpu_affinity child_budget caller_deadline original_call sampler_joined samples_bounded owner_release deadline_unchanged'.split())
ROOT = set('cpu_affinity parent_budget seed_closed child_waited child_inventory snapshots_verified final_journal_observed ledger_preserved nonjournal_preserved child_locks_released raw_removed domain_teardown mounts_restored'.split()) | {'child_' + x for x in CHILD}

def validate(text):
    def load(name):
        spec=importlib.util.spec_from_file_location(name,Path(__file__).with_name(name+'.py'));m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
    closed,facts=load('check-closed-startup-evidence'),load('check-policy-facts-evidence')
    if re.findall(r'(?m)^POLICY_STARTUP_ENVELOPE_PROFILE ([^\n]+)$',text)!=['cpu_count=1 child_seconds=210 parent_seconds=480 phase_minutes=18']:
        raise ValueError('separate adopted caller-envelope profile required')
    parents=closed.chunks(text);wanted=['TestNativeMultiplePolicyStartupEnvelopeUFS']
    if [n for n,_,_,_ in parents]!=wanted or re.findall(r'(?m)^=== RUN   (\S+)$',text)!=wanted or re.findall(r'(?m)^PASS$',text)!=['PASS'] or '--- FAIL:' in text or re.search(r'(?m)^goroutine \d+ \[',text):
        raise ValueError('one complete parent and no raw stacks required')
    if text.index('POLICY_STARTUP_ENVELOPE_PROFILE ')>text.index('=== RUN'):
        raise ValueError('profile before parent required')
    body=parents[0][1]
    for marker,controls in [('POLICY_FACTS_CHILD_RESULT',facts.CHILD),('POLICY_FACTS_PARENT_RESULT',facts.PARENT),('POLICY_STARTUP_ENVELOPE_UFS_RESULT',ROOT)]:
        closed.controls(body,marker,controls)
        if text.count(marker+' ')!=1:raise ValueError('extra/missing original controls')
    if re.findall(r'(?m)^POLICY_STARTUP_ENVELOPE_DOMAIN_TEARDOWN ([^\n]+)$',body)!=['ufs filesystems=pass directories=pass mounts=pass'] or text.count('POLICY_STARTUP_ENVELOPE_DOMAIN_TEARDOWN ')!=1:
        raise ValueError('independent original metadata teardown required')
    def rows(marker,count):
        raw=re.findall(r'(?m)^'+marker+r' ([^\n]+)$',body)
        if len(raw)!=count or text.count(marker+' ')!=count:raise ValueError('exact scalar records required')
        def pairs(items):
            d={}
            for k,v in items:
                if k in d:raise ValueError('duplicate JSON key')
                d[k]=v
            return d
        try:return [json.loads(x,object_pairs_hook=pairs) for x in raw]
        except (json.JSONDecodeError,TypeError) as e:raise ValueError('invalid scalar JSON') from e
    def shape(row,names):
        if type(row) is not dict or set(row)!=set(names.split()):raise ValueError('exact known fields required')
    def integer(v,low,high):
        if type(v) is not int or not low<=v<=high:raise ValueError('bounded scalar integer required')
    def digest(v):
        if type(v) is not str or not re.fullmatch('[a-f0-9]{64}',v):raise ValueError('exact private file hash required')
    s=rows('POLICY_STARTUP_ENVELOPE_SUMMARY',1)[0];shape(s,'return_ns caller_deadline_ns caller_error startup_error owner_present samples')
    integer(s['return_ns'],1,210000000000)
    integer(s['caller_deadline_ns'],180000000000,180000000000)
    if s['caller_deadline_ns']!=180000000000 or type(s['owner_present']) is not bool or s['startup_error'] not in ('none','deadline','cancelled','denied','other') or s['caller_error'] not in ('none','deadline','cancelled') or (s['startup_error']=='none')!=s['owner_present']:
        raise ValueError('separate honest operation/caller results required')
    if type(s['samples']) is not list or len(s['samples'])!=3:raise ValueError('three declared observations required')
    stacks=rows('POLICY_STARTUP_ENVELOPE_STACK',3)
    for i,(sample,stack,offset) in enumerate(zip(s['samples'],stacks,(30,75,110)),1):
        shape(sample,'id offset_ns begin_ns end_ns bytes sha256 status');shape(stack,'id status frames')
        integer(stack['id'],i,i);integer(sample['offset_ns'],offset*1000000000,offset*1000000000)
        if type(sample['id']) is not int or sample['id']!=i or stack['id']!=i or sample['offset_ns']!=offset*1000000000:raise ValueError('fixed ordered samples required')
        if sample['status']=='skipped':
            if not s['return_ns']<sample['offset_ns'] or (sample['begin_ns'],sample['end_ns'],sample['bytes'],sample['sha256'])!=(0,0,0,'') or stack['status']!='skipped' or stack['frames']!=[]:raise ValueError('early return is not a missing sample')
            continue
        if sample['status'] not in ('captured','overlap'):raise ValueError('unknown sample state')
        integer(sample['begin_ns'],sample['offset_ns'],210000000000);integer(sample['end_ns'],sample['begin_ns'],210000000000);integer(sample['bytes'],1,65535);digest(sample['sha256'])
        if (sample['status']=='overlap')!=(sample['end_ns']>s['return_ns']):raise ValueError('capture/return overlap must be explicit')
        if stack['status'] not in ('located','unlocated','ambiguous','overlap') or (sample['status']=='overlap')!=(stack['status']=='overlap'):raise ValueError('honest frame location required')
        if type(stack['frames']) is not list or len(stack['frames'])>64:raise ValueError('bounded frame vector required')
        if stack['status']!='located':
            if stack['frames']:raise ValueError('unlocated samples grant no frames')
            continue
        if not stack['frames']:raise ValueError('located sample requires source frames')
        roots=0
        for frame in stack['frames']:
            shape(frame,'file line symbol');integer(frame['line'],1,100000)
            f=frame['file'];symbol=frame['symbol']
            if type(f) is not str or len(f)>256 or not re.fullmatch(r'[A-Za-z0-9_/-]+\.(go|s)',f) or f.startswith('/') or '..' in f or type(symbol) is not str or len(symbol)>256 or not re.fullmatch(r'[A-Za-z0-9_./*()[\]{}·-]+',symbol) or re.search('0x[a-f0-9]+',symbol):raise ValueError('source-only sanitized frames required')
            roots+=symbol=='github.com/gonz0w/horvium/internal/host/freebsd.activateNativeClosedExecutionOwner'
        if roots!=1:raise ValueError('one actual original factory frame required')
    j=rows('POLICY_STARTUP_ENVELOPE_JOURNAL',1)[0];shape(j,'before_sha256 after_sha256 unchanged decoded state generation_before generation_after');digest(j['before_sha256']);digest(j['after_sha256'])
    if type(j['unchanged']) is not bool or j['unchanged']!=(j['before_sha256']==j['after_sha256']) or type(j['decoded']) is not bool or j['state'] not in ('closed','active','invalid') or j['decoded']!=(j['state']!='invalid'):raise ValueError('independent final metadata observation required')
    integer(j['generation_before'],1,(1<<64)-1);integer(j['generation_after'],0,(1<<64)-1)
    return {'native_acceptance':False,'summary':s,'stacks':stacks,'journal':j}

if __name__=='__main__':
    validate(Path(sys.argv[1]).read_text())
