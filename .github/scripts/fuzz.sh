#!/usr/bin/env bash
# Runs every fuzz target in the server module, each for the time given as the
# first argument (a Go duration, for example 30s). Every target runs even if an
# earlier one fails; the exit status is non-zero if any failed. Go saves a
# failing input under testdata/fuzz/ next to the target's package.
set -euo pipefail

fuzztime=${1:?usage: fuzz.sh <fuzztime>}

# "go test -list" prints each package's matching test names, then an "ok" line
# naming the package.
targets=$(go test -list '^Fuzz' ./... | awk '
	/^Fuzz/ { names[n++] = $1 }
	/^ok/   { for (i = 0; i < n; i++) print $2, names[i]; n = 0 }
')

if [ -z "$targets" ]; then
	echo "no fuzz targets found" >&2
	exit 1
fi

failed=0
while read -r pkg target; do
	echo "::group::fuzz $pkg $target for $fuzztime"
	go test "$pkg" -run '^$' -fuzz "^${target}\$" -fuzztime "$fuzztime" \
		-fuzzminimizetime 5s || failed=1
	echo "::endgroup::"
done <<<"$targets"
exit "$failed"
