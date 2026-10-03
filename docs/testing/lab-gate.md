# Lab gates

## Environments

- E0: local Linux/macOS foundation tooling; no native NAS support claim.
- E1: isolated unit/component fixtures when product code exists.
- E2: disposable native FreeBSD VMs with file-backed/disposable storage and clients.
- E3: physical storage/NIC lab for driver/controller/replacement/performance evidence.
- E4: two independent NAS hosts plus clients for peer/replication/restore evidence.

Developer VMs are test infrastructure; they do not add VM hosting to Horvium.
Pin FreeBSD/Go/packages/clients at milestone entry. A FreeBSD amd64 shell smoke
is weaker evidence than product integration or a supported hardware configuration.

## Small reference

HP MicroServer Gen8, 16 GB RAM, four 3 TB SATA disks. CPU, disk models/health,
sector sizes, controller mode, boot arrangement, NIC speed, and workload are not
yet measured. Record them before test claims. Pool layout is an operator choice,
not hardcoded RAIDZ2. Use disposable media or an explicitly isolated test pool.

## Larger configuration

Select a representative commercial host and workload before claiming commercial
scale qualification. Define measured inventory sizes, client concurrency, metadata
load, job rates, latency, memory/CPU budgets, and throughput. Synthetic large
inventories supplement, but do not replace, native storage/client tests.

E4 proves replication and restoration to a replacement primary. It does not prove
automatic failover, fencing, multi-host control, or shared-storage HA.
