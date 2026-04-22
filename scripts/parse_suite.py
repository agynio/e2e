#!/usr/bin/env python3

from __future__ import annotations

import os
import sys


def parse_suite(path: str) -> dict[str, str]:
    data: dict[str, str] = {}
    current_key: str | None = None
    block_lines: list[str] = []
    block_indent: int | None = None

    def finalize_block() -> None:
        nonlocal current_key, block_lines, block_indent
        if current_key is not None:
            data[current_key] = "\n".join(block_lines).rstrip()
        current_key = None
        block_lines = []
        block_indent = None

    def process_line(line: str) -> None:
        nonlocal current_key, block_indent
        if ":" not in line:
            return
        key, value = line.split(":", 1)
        key = key.strip()
        value = value.strip()
        if value == "|":
            current_key = key
            block_indent = None
            return
        data[key] = value.strip("\"'")

    with open(path, encoding="utf-8") as handle:
        for raw_line in handle:
            line = raw_line.rstrip("\n")
            stripped = line.strip()
            if current_key is not None:
                if stripped == "":
                    block_lines.append("")
                    continue
                if block_indent is None:
                    block_indent = len(line) - len(line.lstrip(" "))
                if len(line) - len(line.lstrip(" ")) >= block_indent:
                    block_lines.append(line[block_indent:])
                    continue
                finalize_block()
                if stripped and not stripped.startswith("#"):
                    process_line(line)
                continue

            if not stripped or stripped.startswith("#"):
                continue
            process_line(line)

    finalize_block()
    return data


def main() -> int:
    if len(sys.argv) != 3:
        print("Usage: parse_suite.py <suite.yaml> <output_dir>", file=sys.stderr)
        return 1

    suite_path = sys.argv[1]
    output_dir = sys.argv[2]
    data = parse_suite(suite_path)
    os.makedirs(output_dir, exist_ok=True)

    for key in ("image", "workdir", "select", "run"):
        value = data.get(key, "")
        output_path = os.path.join(output_dir, key)
        with open(output_path, "w", encoding="utf-8") as handle:
            handle.write(value)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
