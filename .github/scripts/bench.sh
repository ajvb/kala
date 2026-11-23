#!/usr/bin/env bash
# Checks the benchmarks of a pull request against its base branch:
#   bench.sh <base-checkout> <head-checkout> <output-directory>
# Both checkouts are benchmarked on the same machine, alternating between
# them for ROUNDS rounds (default 5) to spread out noise from the machine. A
# benchmark more than 20% slower than the base (median ns/op) is run again, the
# same way, and fails the check only if it is still that much slower.
set -euo pipefail

base=$(cd "${1:?usage: bench.sh <base> <head> <out>}" && pwd)
head=$(cd "${2:?usage: bench.sh <base> <head> <out>}" && pwd)
out=${3:?usage: bench.sh <base> <head> <out>}
rounds=${ROUNDS:-5}
benchtime=${BENCHTIME:-1s}
scripts=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$out"

# The packages of the head checkout that have benchmarks: "go test -list"
# prints a package's benchmark names, then an "ok" line naming the package.
packages=$(cd "$head" && go test -list '^Benchmark' ./... | awk '
	/^Benchmark/ { has = 1 }
	/^ok/        { if (has) print $2; has = 0 }
')
if [ -z "$packages" ]; then
	echo "no benchmarks found" >&2
	exit 1
fi

# run <checkout> <pattern> <file>: one round of the benchmarks matching
# pattern, in each package that exists in the checkout. A build failure or a
# failing benchmark stops the check.
run() {
	local pkgs=() output
	for p in $packages; do
		if (cd "$1" && go list "$p" >/dev/null 2>&1); then pkgs+=("$p"); fi
	done
	[ "${#pkgs[@]}" -gt 0 ] || return 0
	output=$(mktemp)
	if ! (cd "$1" && go test -run '^$' -bench "$2" -benchtime "$benchtime" -count 1 "${pkgs[@]}") >"$output" 2>&1; then
		cat "$output" >&2
		echo "the benchmarks failed in $1" >&2
		exit 1
	fi
	grep '^Benchmark' "$output" >>"$3" || true
	rm -f "$output"
}

# measure <pattern> <name>: alternate base and head for the given rounds.
measure() {
	: >"$out/$2-base.txt"
	: >"$out/$2-head.txt"
	for i in $(seq 1 "$rounds"); do
		echo "round $i of $rounds ($2)"
		run "$base" "$1" "$out/$2-base.txt"
		run "$head" "$1" "$out/$2-head.txt"
	done
}

measure . first
if python3 "$scripts/benchcmp.py" "$out/first-base.txt" "$out/first-head.txt" \
	--regressed "$out/regressed.txt" | tee "$out/report.md"; then
	exit 0
fi

if [ ! -s "$out/regressed.txt" ]; then
	echo "the comparison failed without naming a slower benchmark" >&2
	exit 1
fi
pattern="^($(paste -sd'|' "$out/regressed.txt"))\$"
echo "re-running the benchmarks that were slower: $pattern"
measure "$pattern" rerun
{
	echo
	echo "Re-run of the benchmarks that were slower:"
	echo
} >>"$out/report.md"
python3 "$scripts/benchcmp.py" "$out/rerun-base.txt" "$out/rerun-head.txt" | tee -a "$out/report.md"
