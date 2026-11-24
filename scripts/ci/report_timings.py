#!/usr/bin/env python3
"""Report the slowest recorded Go test and package events as JSON.

Samples are individual terminal events, not averages. Parent test times include
their subtests, and package times include test execution and other overhead.
"""

import argparse
import heapq
import json
import math
from pathlib import Path
import sys


def report(paths, limit):
    """Stream timing samples, retaining at most limit results per event kind."""
    samples = {"test": [], "package": []}
    counts = {"test": 0, "package": 0}
    missing = 0
    for path in paths:
        with open(path, encoding="utf-8") as stream:
            for line in stream:
                record = json.loads(line)
                kind = record.get("kind")
                if kind not in samples:
                    continue
                elapsed = record.get("elapsed")
                if (isinstance(elapsed, bool) or not isinstance(elapsed, (int, float)) or
                        not math.isfinite(elapsed) or elapsed < 0):
                    missing += 1
                    continue
                counts[kind] += 1
                identity = tuple(str(record.get(key) or "") for key in
                                 ("module", "package", "test", "status", "job", "case", "invocation"))
                item = (elapsed, identity)
                heap = samples[kind]
                if len(heap) < limit:
                    heapq.heappush(heap, item)
                elif item > heap[0]:
                    heapq.heapreplace(heap, item)
    return {"counts": counts, "missing_or_invalid_elapsed": missing,
            "slowest": {kind: [{"elapsed": elapsed, "module": identity[0], "package": identity[1],
                                "test": identity[2] or None, "status": identity[3], "job": identity[4],
                                "case": identity[5], "invocation": identity[6]}
                               for elapsed, identity in sorted(heap, reverse=True)]
                        for kind, heap in samples.items()}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("paths", nargs="*", help="record JSONL files")
    parser.add_argument("--execution-dir", help="directory of downloaded ci-records artifacts")
    parser.add_argument("--limit", type=int, default=20, help="maximum samples per kind (1-1000)")
    args = parser.parse_args()
    if not 1 <= args.limit <= 1000:
        parser.error("--limit must be between 1 and 1000")
    if bool(args.paths) == bool(args.execution_dir):
        parser.error("provide record files or --execution-dir")
    try:
        paths = sorted(Path(args.execution_dir).glob("ci-records-*/*.jsonl")) if args.execution_dir else args.paths
        if not paths:
            raise ValueError("no record files found")
        print(json.dumps(report(paths, args.limit), indent=2))
    except (OSError, ValueError, TypeError, AttributeError) as exc:
        print(f"cannot read timing records: {str(exc)[:200]}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
