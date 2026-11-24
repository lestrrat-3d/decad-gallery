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
    excluded = {"cached": 0, "unknown_cache": 0}
    pending = {}

    def retain(heap, item):
        if len(heap) < limit:
            heapq.heappush(heap, item)
        elif item > heap[0]:
            heapq.heapreplace(heap, item)

    for path in paths:
        with open(path, encoding="utf-8") as stream:
            for line in stream:
                record = json.loads(line)
                kind = record.get("kind")
                if kind not in samples:
                    continue
                elapsed = record.get("elapsed")
                valid = (not isinstance(elapsed, bool) and isinstance(elapsed, (int, float)) and
                         math.isfinite(elapsed) and elapsed >= 0)
                identity = tuple(str(record.get(key) or "") for key in
                                 ("module", "package", "test", "status", "job", "case", "invocation"))
                key = (str(path), identity[6], identity[1])
                state = pending.setdefault(key, {"count": 0, "missing": 0, "samples": []})
                if kind == "test":
                    if valid:
                        state["count"] += 1
                        retain(state["samples"], (elapsed, identity))
                    else:
                        state["missing"] += 1
                    continue
                cache = record.get("cached")
                if cache is not False:
                    excluded["cached" if cache is True else "unknown_cache"] += state["count"] + state["missing"] + 1
                else:
                    counts["test"] += state["count"]
                    missing += state["missing"]
                    for item in state["samples"]:
                        retain(samples["test"], item)
                    if valid:
                        counts["package"] += 1
                        retain(samples["package"], (elapsed, identity))
                    else:
                        missing += 1
                del pending[key]
    excluded["unknown_cache"] += sum(state["count"] + state["missing"] for state in pending.values())
    return {"counts": counts, "missing_or_invalid_elapsed": missing,
            "excluded": excluded,
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
