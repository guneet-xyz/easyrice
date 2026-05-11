#!/bin/bash
set -euo pipefail

THRESHOLD_OVERALL=${THRESHOLD_OVERALL:-95}
THRESHOLD_PKG=${THRESHOLD_PKG:-90}
COVERAGE_FILE="${1:-coverage.out}"

if [[ ! -f "$COVERAGE_FILE" ]]; then
    echo "Error: Coverage file not found: $COVERAGE_FILE"
    exit 1
fi

OVERALL=$(go tool cover -func="$COVERAGE_FILE" | grep '^total:' | awk '{print $NF}' | sed 's/%//')

if [[ -z "$OVERALL" ]]; then
    echo "Error: Could not extract overall coverage from $COVERAGE_FILE"
    exit 1
fi

tmpdir=$(mktemp -d)
trap "rm -rf $tmpdir" EXIT

pkg_file="$tmpdir/packages"

go tool cover -func="$COVERAGE_FILE" | grep -v '^total:' | awk '
    {
        path = $1
        cov = $NF
        gsub(/%/, "", cov)
        
        if (path ~ /github\.com\/guneet-xyz\/easyrice/) {
            split(path, parts, "/")
            split(parts[5], file_parts, ":")
            file = file_parts[1]
            
            n = NF - 1
            fname = $n
            
            if (file == "main.go" && fname == "main") next
            if (file == "root.go" && fname == "Execute") next
            
            pkg = parts[4]
            print pkg " " cov
        }
    }
' >> "$pkg_file"

echo "Coverage Report:"
echo "================"
echo ""

failed=0

if [[ -s "$pkg_file" ]]; then
    awk '{pkg=$1; cov=$2; if (!(pkg in pkgs)) {pkgs[pkg]=0; counts[pkg]=0} pkgs[pkg]+=cov; counts[pkg]++} 
         END {for (pkg in pkgs) {avg=pkgs[pkg]/counts[pkg]; printf "%s %.1f\n", pkg, avg}}' "$pkg_file" | sort | while read pkg avg; do
        status="✓"
        if (( $(echo "$avg < $THRESHOLD_PKG" | bc -l) )); then
            status="✗"
            echo "$status" > "$tmpdir/failed"
        fi
        echo "$status $pkg: $avg%"
    done
fi

echo ""
echo "Overall: $OVERALL%"
echo ""

if (( $(echo "$OVERALL < $THRESHOLD_OVERALL" | bc -l) )); then
    echo "Error: Overall coverage $OVERALL% is below threshold $THRESHOLD_OVERALL%"
    exit 1
fi

if [[ -f "$tmpdir/failed" ]]; then
    echo "Error: One or more packages below threshold $THRESHOLD_PKG%"
    exit 1
fi

echo "✓ All coverage checks passed"
exit 0
