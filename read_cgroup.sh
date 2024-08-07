#!/bin/bash

# Specify the path to your cgroup directory
CGROUP_PATH="/sys/fs/cgroup/fukumu"

# List of files to read
FILES=(
    "cgroup.controllers"
    "cgroup.type"
    "cpuset.mems.effective"
    "io.pressure"
    "memory.min"
    "memory.swap.peak"
    "cgroup.events"
    "cpu.idle"
    "cpu.stat"
    "io.prio.class"
    "memory.numa_stat"
    "memory.zswap.current"
    "cgroup.freeze"
    "cpu.max"
    "cpu.stat.local"
    "io.stat"
    "memory.oom.group"
    "memory.zswap.max"
    "cgroup.kill"
    "cpu.max.burst"
    "cpu.uclamp.max"
    "io.weight"
    "memory.peak"
    "memory.zswap.writeback"
    "cgroup.max.depth"
    "cpu.pressure"
    "cpu.uclamp.min"
    "irq.pressure"
    "memory.pressure"
    "pids.current"
    "cgroup.max.descendants"
    "cpuset.cpus"
    "cpu.weight"
    "memory.current"
    "memory.reclaim"
    "pids.events"
    "cgroup.pressure"
    "cpuset.cpus.effective"
    "cpu.weight.nice"
    "memory.events"
    "memory.stat"
    "pids.max"
    "cgroup.procs"
    "cpuset.cpus.exclusive"
    "io.bfq.weight"
    "memory.events.local"
    "memory.swap.current"
    "pids.peak"
    "cgroup.stat"
    "cpuset.cpus.exclusive.effective"
    "io.latency"
    "memory.high"
    "memory.swap.events"
    "cgroup.subtree_control"
    "cpuset.cpus.partition"
    "io.low"
    "memory.low"
    "memory.swap.high"
    "cgroup.threads"
    "cpuset.mems"
    "io.max"
    "memory.max"
    "memory.swap.max"
)

# Loop through each file and print its contents
for FILE in "${FILES[@]}"; do
    FILE_PATH="${CGROUP_PATH}/${FILE}"
    if [ -f "$FILE_PATH" ]; then
        echo "Contents of $FILE:"
        cat "$FILE_PATH"
        echo
    else
        echo "File $FILE does not exist."
  
    fi
done
