# Distributed Read-Write Mutex in Go

The default Go implementation of
[sync.RWMutex](https://golang.org/pkg/sync/#RWMutex) does not scale well
to multiple cores, as all readers contend on the same memory location
when they all try to atomically increment it. This repository provides
an `n`-way RWMutex, also known as a "big reader" lock, which gives each
CPU core its own RWMutex. Readers take only a read lock local to their
core, whereas writers must take all locks in order.

**Note that the current implementation only supports x86 processors on
Linux; other combinations will revert (automatically) to the old
sync.RWMutex behaviour. To support other architectures and OSes, the
appropriate `cpu_GOARCH.go` and `cpus_GOOS.go` files need to be written.
If you have a different setup available, and have the time to write one
of these, I'll happily accept patches.**

## Finding the current CPU

To determine which lock to take, readers use the CPUID instruction,
which gives the APICID of the currently active CPU without having to
issue a system call or modify the runtime. This instruction is supported
on both Intel and AMD processors; ARM CPUs should use the [CPU ID
register](http://infocenter.arm.com/help/index.jsp?topic=/com.arm.doc.ddi0360e/CACEDHJG.html)
instead. For systems with more than 256 processors, x2APIC must be used,
and the EDX register after CPUID with EAX=0xb should be used instead. A
mapping from APICID to CPU index is constructed (using CPU affinity
syscalls) when the program is started, as it is static for the lifetime
of a process.  Since the CPUID instruction can be fairly expensive,
goroutines will also only periodically update their estimate of what
core they are running on.  More frequent updates lead to less inter-core
lock traffic, but also increases the time spent on CPUID instructions
relative to the actual locking.

**Stale CPU information.**
The information of which CPU a goroutine is running on *might* be stale
when we take the lock (the goroutine could have been moved to another
core), but this will only affect performance, not correctness, as long
as the reader remembers which lock it took. Such moves are also
unlikely, as the OS kernel tries to keep threads on the same core to
improve cache hits.

## Performance

There are many parameters that affect the performance characteristics of
this scheme. In particular, the frequency of CPUID checking, the number
of readers, the ratio of readers to writers, and the time readers hold
their locks, are all important. Since only a single writer is active at
the time, the duration a writer holds a lock for does not affect the
difference in performance between sync.RWMutex and DRWMutex.

Experiments show that DRWMutex performs better the more cores the system
has, and in particular when the fraction of writers is <1%, and CPUID is
called at most every 10 locks (this changes depending on the duration a
lock is held for). Even on few cores, DRWMutex outperforms sync.RWMutex
under these conditions, which are common for applications that elect to
use sync.RWMutex over sync.Mutex.

The plot below shows mean performance across 30 runs (using
[experiment](https://github.com/jonhoo/experiment)) as the number of
cores increases using:

    drwmutex-bench -i 5000 -p 0.0001 -n 500 -w 1 -r 100 -c 100

![DRWMutex and sync.RWMutex performance comparison](benchmarks/perf.png)

Error bars denote 25th and 75th percentile.
Note the drops every 10th core; this is because 10 cores constitute a
NUMA node on the machine the benchmarks were run on, so once a NUMA node
is added, cross-core traffic becomes more expensive. Performance
increases for DRWMutex as more readers can work in parallel compared to
sync.RWMutex.

See the [go-nuts
thread](https://groups.google.com/d/msg/golang-nuts/zt_CQssHw4M/TteNG44geaEJ)
for further discussion.
