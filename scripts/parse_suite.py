#!/usr/bin/env python3

from __future__ import annotations

import os
import sys

try:
    import yaml
except ImportError:  # pragma: no cover - execution environment dependency
    yaml = None


def parse_suite(path: str) -> dict[str, str]:
    if yaml is None:
        raise RuntimeError(
            "PyYAML is required for suite parsing. Install with: python3 -m pip install pyyaml"
        )

    with open(path, encoding="utf-8") as handle:
        try:
            data = yaml.safe_load(handle)
        except yaml.YAMLError as exc:  # type: ignore[attr-defined]
            raise ValueError(f"Invalid YAML in {path}: {exc}") from exc

    if not isinstance(data, dict):
        raise ValueError(f"Suite file {path} must define a YAML mapping at the top level.")

    allowed_keys = {"image", "workdir", "select", "run", "manifests", "service_account", "required_env"}
    required_keys = {"image", "select", "run"}
    unknown_keys = set(data.keys()) - allowed_keys
    if unknown_keys:
        unknown_list = ", ".join(sorted(unknown_keys))
        raise ValueError(f"Unsupported keys in {path}: {unknown_list}")

    missing_keys = required_keys - set(data.keys())
    if missing_keys:
        missing_list = ", ".join(sorted(missing_keys))
        raise ValueError(f"Missing required keys in {path}: {missing_list}")

    parsed: dict[str, str] = {}
    for key in allowed_keys:
        value = data.get(key, "")
        if key == "manifests":
            if value is None:
                value = ""
            if isinstance(value, list):
                entries: list[str] = []
                for entry in value:
                    if not isinstance(entry, str):
                        raise ValueError(f"Manifest entries in {path} must be strings.")
                    trimmed = entry.strip()
                    if trimmed:
                        entries.append(trimmed)
                value = "\n".join(entries)
            elif not isinstance(value, str):
                raise ValueError(f"Value for '{key}' in {path} must be a string or list of strings.")
        elif key == "required_env":
            if value is None or value == "":
                value = ""
            elif isinstance(value, list):
                entries: list[str] = []
                for entry in value:
                    if not isinstance(entry, str):
                        raise ValueError(f"Required env entries in {path} must be strings.")
                    trimmed = entry.strip()
                    if trimmed:
                        entries.append(trimmed)
                value = "\n".join(entries)
            else:
                raise ValueError(f"Value for '{key}' in {path} must be a list of strings.")
        else:
            if value is None:
                value = ""
            if not isinstance(value, str):
                raise ValueError(f"Value for '{key}' in {path} must be a string.")
            if key in required_keys and not value.strip():
                raise ValueError(f"Value for '{key}' in {path} cannot be empty.")
        parsed[key] = value

    return parsed


def main() -> int:
    if len(sys.argv) != 3:
        print("Usage: parse_suite.py <suite.yaml> <output_dir>", file=sys.stderr)
        return 1

    suite_path = sys.argv[1]
    output_dir = sys.argv[2]
    try:
        data = parse_suite(suite_path)
    except (RuntimeError, ValueError) as exc:
        print(str(exc), file=sys.stderr)
        return 1
    os.makedirs(output_dir, exist_ok=True)

    for key in ("image", "workdir", "select", "run", "manifests", "service_account", "required_env"):
        value = data.get(key, "")
        output_path = os.path.join(output_dir, key)
        with open(output_path, "w", encoding="utf-8") as handle:
            handle.write(value)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
