#!/usr/bin/env python3
"""Compare Go benchmark results of a base and a head build.

Usage: benchcmp.py BASE.txt HEAD.txt [--threshold PERCENT] [--regressed FILE]

Each file holds "go test -bench" output, usually several runs of each
benchmark. For every benchmark in both files it compares the median ns/op and
prints a Markdown table. A benchmark is a regression if the head median is more
than PERCENT (default 20) slower than the base median. The names of the
top-level benchmarks that regressed are written to FILE, one per line. The exit
status is 1 if any benchmark regressed, otherwise 0. Benchmarks in only one
file are listed but never fail.
"""

import argparse
import re
import statistics
import sys

LINE = re.compile(r"^(Benchmark\S+?)(?:-\d+)?\s+\d+\s+([\d.]+) ns/op")


def read(path):
    results = {}
    with open(path) as f:
        for line in f:
            m = LINE.match(line)
            if m:
                results.setdefault(m.group(1), []).append(float(m.group(2)))
    return results


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("base")
    parser.add_argument("head")
    parser.add_argument("--threshold", type=float, default=20.0)
    parser.add_argument("--regressed")
    args = parser.parse_args()

    base, head = read(args.base), read(args.head)
    regressed = []
    print("| Benchmark | Base ns/op | Head ns/op | Change |")
    print("| --- | ---: | ---: | ---: |")
    for name in sorted(set(base) | set(head)):
        if name not in base or name not in head:
            where = "head" if name in head else "base"
            print(f"| `{name}` | | | only in {where} |")
            continue
        b, h = statistics.median(base[name]), statistics.median(head[name])
        change = (h - b) / b * 100 if b else 0.0
        flag = ""
        if change > args.threshold:
            flag = " **regression**"
            regressed.append(name)
        print(f"| `{name}` | {b:.1f} | {h:.1f} | {change:+.1f}%{flag} |")

    if args.regressed:
        with open(args.regressed, "w") as f:
            for top in sorted({n.split("/")[0] for n in regressed}):
                f.write(top + "\n")
    return 1 if regressed else 0


if __name__ == "__main__":
    sys.exit(main())
