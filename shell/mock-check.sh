#!/usr/bin/env bash

RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

mocksCreatedWithoutNew="$(rg -n --glob '*.go' --glob '!vendor/**' '(&mocks\.[[:upper:]][[:alnum:]_]*\{|[^[:alnum:]_]mocks\.[[:upper:]][[:alnum:]_]*\{|new\(mocks\.[[:upper:]][[:alnum:]_]*\))' . || true)"

if [ -z "$mocksCreatedWithoutNew" ];
then
  exit 0
else
  printf "${RED}ERROR:${NC} there are mocks that doesn't follow the constructor pattern ${CYAN}m := mocks.NewConstructor(t)${NC}:\n"
  printf "${RED}$mocksCreatedWithoutNew"
  exit 1
fi
