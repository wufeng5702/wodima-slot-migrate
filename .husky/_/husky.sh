#!/usr/bin/env sh
# husky shim - run lint-staged via pnpm if available
if command -v pnpm >/dev/null 2>&1; then
  pnpm dlx --silent husky@latest -- run-hook "$1"
else
  exit 0
fi
