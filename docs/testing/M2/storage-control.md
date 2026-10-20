# Retained storage selection and original lifecycle consumer

The [admitted operator contract](../../milestones/M2/storage-control-contract.md)
now has one coherent **unregistered source consumer**: exact selection bytes,
retained native configuration custody, original read-only inspection, finite
fresh commissioning, foreground empty-history startup, terminal cleanup and
bounded output delivery. Refs [#139](https://github.com/gonz0w/horvium/issues/139)
and [#305](https://github.com/gonz0w/horvium/issues/305); closes neither.

No `cmd/horvium`, helper, management route or service calls this consumer. M1
configuration, credential storage, inventory, `serve` and `helper` remain
unchanged. No backend, worker, global launch permit, enrollment, recovery replay,
installation, deployment or actual mutation activation is enabled.

## Selection and native custody

`ParseControlConfig` accepts only the four decoded keys, canonical version token
and separate format in the admitted contract. It bounds the complete read before
decoding, rejects escaped duplicate aliases and unknown/case-changed keys, and
refuses malformed UTF-8/BOM, unpaired surrogate replacement, wrong types, trailing
documents and noncanonical/overlapping paths. Returned paths are metadata only.

`OpenControlSelection` retains the original no-follow regular root-owned 0600
single-link file and every protected root-owned ancestor descriptor for all three
paths. Checks pin exact file metadata and SHA-256 content, ancestor identity,
name correspondence, ACL and actual mount facts before and after reads. Directory
entry timestamps/link counts may change during original fresh creation; stable
identity, mode, owner/group, birth/generation, flags, ACL and mount facts remain
pinned. File access time alone is not drift. Admission failure can return a
retired original; the consumer always closes it and checks the release result.

Live restore-scope checks compare actual filesystem kind/source/target/fsid,
retained directory identities and strict ZFS dataset-component ancestry. A config
on the ledger's ZFS filesystem, with its source, or on a descendant dataset is
refused. Source spelling inequality alone is insufficient. Unknown, read-only,
soft-update or unsupported mount facts refuse. UFS permits the same filesystem
only outside the actual ledger directory/file domain, under the admitted
**ledger-file-only** restoration procedure. These are live custody checks, not
persistent GUID or completed restore-independence evidence. A config dataset
ancestor is outside a precisely ledger-only rollback; ancestor/recursive rollback
affecting config is unsupported. Joint rollback detection is not claimed. The interval after the final retained
observation is not an atomic fence against malicious root replacement.

Custody/root loss or replacement retires the selection. There is no reload,
repair, permission change, descriptor reopening as writer authority or fallback
action. Both absent names are checked before entering fresh commissioning; the
existing exclusive producer repeats that guard before its first effect. Existing
M1 credential directories cannot become fresh targets; existing startup/inspection
also require the original complete journal and ledger, not credential files.

## Original assembly and failure boundaries

The consumer uses the existing process-global single-use registry, pure actual
real/effective-root check, original commissioned owner and journal-before-ledger
ordering. Its selection checks use the current bounded operation context; they
are never embedded as I/O in the context-free root predicate. The underlying
ownership framework and fixed native backing collector remain unchanged.

`commission-fresh` uses the original fixed fresh activation, unused observation,
one acknowledged `FinishIdle` and original release. `start-closed` fixedly uses
`ReviewExecutionStartup` followed by its original `Activate`: the actual complete
canonical empty-census guard remains under the original locks before entropy and
publication. The separate policy-aware path remains unchanged. Factory entry consumes the process registration before even read-only startup
review; a constructor failure cannot be reported as an unentered attempt.
No supplied list,
qualification boolean or copied receipt can waive #305; policies and every job
remain refused by this configured startup path.

`inspect` opens existing original pair leases without the writer registry. It
derives installation metadata from the protected journal while both original
leases/mutexes are held, then uses the complete bounded repeated census, native
backing and system observations. No startup/activate/finish is called and no old
acknowledgement is synthesized. Only a complete exact closed journal/census with
no jobs reports closed. Complete active/pending/disarmed/mismatched history reports
recovery-required; missing or uninspectable state supplies no structured result.
The existing installation-explicit observation API keeps its validation.

Foreground startup emits nothing until its first cancellation/SIGINT/SIGTERM.
It then makes one cleanup context from `context.Background`, capped at 90 seconds,
checks custody/root, attempts terminal unused finish once and always attempts
original release. That same budget remains alive for the final original selection
check after owner/pair release, immediately before config Close. Actual root is
checked around final config release as well. Later cancellation does not renew that context or retry finish.
Finite/admission/inspection work retains one whole-operation 90-second parent;
existing nested bounds are not widened. A failed observation never becomes a
successful unused-close claim. No separate output goroutine or polling loop exists.

Every result is staged privately. Both output sinks are called only after
confirmed original owner/pair and selection release. Failed finish followed by
confirmed release can emit the fixed recovery class. **Unconfirmed release calls
neither sink**, retains a private recovery outcome and never reports closed;
process/command teardown remains unregistered. Existing native constructors that
return no original after failure cannot independently confirm their internal
release, so this consumer conservatively suppresses output for those outcomes
as well. It does not retry close or infer cleanup from the absence of a PID.
Short/failing writes are terminal, never structured-result retries or permission
to replay the action. Output contains only the five admitted keys/fixed classes,
never paths, native errors, credentials, inventory or original capabilities.

## Portable evidence and remaining acceptance

The [source manifest](storage-control-manifest.json) binds the implementation
and preservation inputs. Portable tests cover literal bytes and byte/path bounds,
branch selection/refusal, the actual original model registry, retirement/root
loss, a fresh foreground cleanup context/deadline, repeated cancellation, failed
finish/release and a blocked output writer entered only after cleanup. Inspection
classification and live-scope projections use synthetic metadata; they are not
native custody, complete kernel census, storage authority or restore proof.

The initial ordinary JSON decoder reproduced behavioral failures for escaped
duplicates, wrong-case fields and extra fields. Corrected focused race checks
passed. Final signed `make review` and unexecuted FreeBSD compilation are recorded
against their exact candidate in the PR/local receipts. Required baseline review
includes unchanged M1 configuration and read-only consumers. GitHub Actions stay
disabled; there is no hosted status substituted for local evidence.

**SC-N01 through SC-N08 remain unexecuted for this assembled consumer.** Both
pinned disposable FreeBSD candidates still need actual UID/inherited-descriptor,
ACL/name/content drift, exclusive-creation cuts, original contention, mounted
backing/restore scope, lifecycle/acknowledgement/deadline and independent daemon/data
preservation qualification before any command registration. The complete #305
same-candidate both-release policy lifecycle/fault gate and full #139
startup/global/backend/recovery acceptance remain open. No new VM attempt,
privileged operation, cache admission, retained-data deletion or resource/support
claim is part of this delivery. Draft #415 and the existing failed/unknown evidence
are preserved.
