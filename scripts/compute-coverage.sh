#!/bin/bash
# Computes the average test coverage across library packages (excludes cmd/)
set -euo pipefail

# Run tests with coverage (force re-run to get coverage output)
go test -count=1 -cover ./... 2>&1 | \
  grep "coverage:" | \
  grep -v "cmd/" | \
  awk '{
    # Extract percentage from "coverage: X.Y% of statements"
    for (i = 1; i <= NF; i++) {
      if ($i == "coverage:") {
        gsub(/%/, "", $(i+1));
        sum += $(i+1);
        n++;
        break;
      }
    }
  } END {
    if (n > 0) {
      printf "%.1f%%\n", sum/n;
    } else {
      print "0.0%";
    }
  }'
