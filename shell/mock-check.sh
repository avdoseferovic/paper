#!/usr/bin/env bash
#
# Enforces the mock constructor pattern: mocks must be created via
# m := mocks.NewConstructor(t), never via struct literals or new().
# Uses ripgrep when available, falls back to POSIX grep otherwise.

RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

PATTERN='(&mocks\.[[:upper:]][[:alnum:]_]*\{|[^[:alnum:]_]mocks\.[[:upper:]][[:alnum:]_]*\{|new\(mocks\.[[:upper:]][[:alnum:]_]*\))'

if command -v rg >/dev/null 2>&1; then
  mocksCreatedWithoutNew="$(rg -n --glob '*.go' --glob '!vendor/**' "$PATTERN" . || true)"
else
  mocksCreatedWithoutNew="$(grep -rnE --include='*.go' --exclude-dir=vendor --exclude-dir=.git "$PATTERN" . || true)"
fi

if [ -z "$mocksCreatedWithoutNew" ];
then
  exit 0
else
  printf "${RED}ERROR:${NC} there are mocks that doesn't follow the constructor pattern ${CYAN}m := mocks.NewConstructor(t)${NC}:\n"
  printf "${RED}%s${NC}\n" "$mocksCreatedWithoutNew"
  exit 1
fi
