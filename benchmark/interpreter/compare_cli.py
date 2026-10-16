#!/usr/bin/env python3
"""Compare complete CLI invocations using identical input and output."""

import argparse
import hashlib
import json
from pathlib import Path
import statistics
import subprocess
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("reference", type=Path)
parser.add_argument("candidate", type=Path)
parser.add_argument("oracle", type=Path)
parser.add_argument("source", type=Path)
parser.add_argument("--runs", type=int, default=11)
parser.add_argument("--warmups", type=int, default=2)
parser.add_argument("--output", type=Path, required=True)
args = parser.parse_args()
if args.runs < 1 or args.warmups < 0:
    parser.error("runs must be positive and warmups must be nonnegative")

binaries = {
    name: getattr(args, name).resolve()
    for name in ("reference", "candidate", "oracle")
}
source = args.source.resolve()
expected = subprocess.check_output([binaries["reference"], source])
timings = {name: [] for name in binaries}
names = list(binaries)
for iteration in range(args.warmups + args.runs):
    # Rotate execution order so no binary always runs first.
    order = names[iteration % len(names):] + names[:iteration % len(names)]
    for name in order:
        start = time.perf_counter_ns()
        result = subprocess.run(
            [binaries[name], source], capture_output=True, check=True
        )
        elapsed = (time.perf_counter_ns() - start) / 1_000_000_000
        if result.stdout != expected:
            raise RuntimeError(f"{name} produced different output")
        if iteration >= args.warmups:
            timings[name].append(elapsed)

medians = {name: statistics.median(values) for name, values in timings.items()}
report = {
    "seconds": timings,
    "medians": medians,
    "candidate_speedup": medians["reference"] / medians["candidate"],
    "oracle_speedup": medians["oracle"] / medians["candidate"],
    "binaries": {
        name: {
            "path": str(path),
            "sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
        }
        for name, path in binaries.items()
    },
    "source": {
        "path": str(source),
        "sha256": hashlib.sha256(source.read_bytes()).hexdigest(),
    },
}
args.output.write_text(json.dumps(report, indent=2) + "\n")
print(json.dumps({key: report[key] for key in (
    "medians", "candidate_speedup", "oracle_speedup"
)}, indent=2))
