# Bounded development storage

This tooling adoption is separate from the pinned method bundle. The optional
read-only guard comes from Agent Temper revision
`1c9e9fb7e5e1152691dbc669d5f7222a7c0c377d` under BSD-3-Clause.

## Allocation gates

`make storage-preflight` accounts for allocated blocks in the resolved `.tools`
location and existing `bin`, including external cache targets. Ordinary developer
output has a 20 GB budget and a 2 GB additional peak allowance. Remaining free
space must exceed 10% of filesystem capacity, capped at 200 GB. This retains a
200 GB reserve on a 2 TB workstation without requiring a small guest to have more
free space than its whole disk. All values are decimal bytes. New ordinary builds
and tool bootstrap use this gate before allocation. Different filesystems require
separate reviewed budgets rather than a misleading combined free-space figure.

The normal source check invokes the staged Git guard before other checks. It
rejects new or changed blobs over 10 MiB and common disk-image extensions. Existing
unchanged history is not rewritten or purged. Real source-asset exceptions need
an exact path and reviewed justification; ignored files are not automatically safe.

This is a snapshot, not a disk reservation or continuous monitor. Concurrent jobs
need combined peak accounting, and a larger image/native campaign requires its
own bounded profile and exact output roots, including directories outside the
repository. Do not omit external output or lower an estimate to pass a failure.
The gate never deletes files to make room. Unavailable measurement stops allocation.
Copy-on-write allocation can overcount shared blocks; actual free-space changes,
not apparent file size, determine reclaimed space.

## Retention and cleanup

Keep source, needed small fixtures and compact text command/result evidence.
Retain diagnostics and reproducer inputs for unresolved failures. Do not retain
full dated copies by default. Reuse one digest-verified canonical input per
content identity, and preserve every base referenced by surviving VM overlays.
A work item records retained exceptions, their finite budget and review event.

Existing run-owned cleanup remains in effect. Cleanup may remove only files that
the current run explicitly owns as disposable, after process/mount cleanup and
required evidence capture. Uncertain cleanup or active dependencies preserve the
output for reconciliation. Never purge a whole cache, `.git`, dated directory or
VM family on age alone. Interrupted runs require ownership/dependency checks.
At handoff report output allocation, free space and retained exceptions. This
source change grants no retrospective deletion, runner activation or native
execution authority and does not claim that all outside task workspaces are wired.
