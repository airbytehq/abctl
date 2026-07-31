#!/usr/bin/env python3
"""Anchor structured review findings to changed lines in a unified Git diff."""

from __future__ import annotations

import argparse
import json
import re
import shlex
import sys
from dataclasses import dataclass, field
from pathlib import Path


HUNK = re.compile(r"^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@")
MIN_QUOTE_LENGTH = 8
VALID_SEVERITIES = {"P0", "P1", "P2"}
VALID_CONFIDENCE = {"VERIFIED", "LIKELY"}


@dataclass(frozen=True)
class ChangedLine:
    number: int
    side: str
    content: str


@dataclass
class DiffFile:
    path: str
    changed: list[ChangedLine] = field(default_factory=list)


def normalize_path(path: str) -> str:
    path = path.strip()
    if path.startswith("a/") or path.startswith("b/"):
        return path[2:]
    return path


def diff_paths(header: str) -> tuple[str, str] | None:
    try:
        parts = shlex.split(header)
    except ValueError:
        return None
    if len(parts) < 4 or parts[:2] != ["diff", "--git"]:
        return None
    return normalize_path(parts[2]), normalize_path(parts[3])


def parse_diff(raw: str) -> dict[str, DiffFile]:
    files: dict[str, DiffFile] = {}
    current: DiffFile | None = None
    old_line = 0
    new_line = 0
    in_hunk = False

    for raw_line in raw.splitlines():
        paths = diff_paths(raw_line)
        if paths is not None:
            path = paths[1] if paths[1] != "/dev/null" else paths[0]
            current = files.setdefault(path, DiffFile(path))
            in_hunk = False
            continue
        match = HUNK.match(raw_line)
        if match and current is not None:
            old_line, new_line = map(int, match.groups())
            in_hunk = True
            continue
        if not in_hunk or current is None:
            continue
        if raw_line.startswith("\\ No newline at end of file"):
            continue
        if raw_line.startswith("+") and not raw_line.startswith("+++"):
            current.changed.append(ChangedLine(new_line, "added", normalize_text(raw_line[1:])))
            new_line += 1
        elif raw_line.startswith("-") and not raw_line.startswith("---"):
            current.changed.append(ChangedLine(old_line, "removed", normalize_text(raw_line[1:])))
            old_line += 1
        elif raw_line.startswith(" "):
            old_line += 1
            new_line += 1

    return files


def normalize_text(value: str) -> str:
    return " ".join(value.split())


def parse_quote(value: object) -> tuple[str | None, str] | None:
    if not isinstance(value, str):
        return None
    stripped = value.strip()
    if not stripped:
        return None
    side: str | None = None
    if stripped.startswith("+") and not stripped.startswith("+++"):
        side = "added"
        stripped = stripped[1:]
    elif stripped.startswith("-") and not stripped.startswith("---"):
        side = "removed"
        stripped = stripped[1:]
    normalized = normalize_text(stripped)
    if len(normalized) < MIN_QUOTE_LENGTH:
        return None
    return side, normalized


def valid_shape(finding: object) -> bool:
    if not isinstance(finding, dict):
        return False
    line = finding.get("line")
    string_fields = (
        "file",
        "title",
        "issue",
        "failure_scenario",
        "suggestion",
        "category",
        "diff_quote",
        "causal_diff_quote",
    )
    return (
        all(isinstance(finding.get(field), str) for field in string_fields)
        and isinstance(line, int)
        and not isinstance(line, bool)
        and finding.get("severity") in VALID_SEVERITIES
        and finding.get("confidence") in VALID_CONFIDENCE
    )


def quote_matches(lines: list[ChangedLine], quote: object, line_number: int | None) -> bool:
    parsed = parse_quote(quote)
    if parsed is None:
        return False
    required_side, text = parsed
    for changed in lines:
        if line_number is not None and changed.number != line_number:
            continue
        if required_side is not None and changed.side != required_side:
            continue
        if text in changed.content:
            return True
    return False


def validate(diff: dict[str, DiffFile], findings: list[object]) -> dict[str, list[object]]:
    buckets: dict[str, list[object]] = {"anchored": [], "causal": [], "needs_review": []}
    for finding in findings:
        if not valid_shape(finding) or finding.get("extraction_failed") is True:
            buckets["needs_review"].append(finding)
            continue
        path = normalize_path(finding["file"])
        changed_file = diff.get(path)
        if changed_file is None:
            buckets["needs_review"].append(finding)
            continue
        if quote_matches(changed_file.changed, finding.get("diff_quote"), finding["line"]):
            buckets["anchored"].append(finding)
        elif quote_matches(changed_file.changed, finding.get("causal_diff_quote"), None):
            buckets["causal"].append(finding)
        else:
            buckets["needs_review"].append(finding)
    return buckets


def load_findings(path: Path) -> list[object]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, list):
        raise ValueError("findings JSON must be an array")
    return value


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("diff", type=Path)
    parser.add_argument("findings", type=Path)
    args = parser.parse_args()
    try:
        parsed_diff = parse_diff(args.diff.read_text(encoding="utf-8"))
        findings = load_findings(args.findings)
    except (OSError, UnicodeError, json.JSONDecodeError, ValueError) as exc:
        print(f"validate_findings: {exc}", file=sys.stderr)
        return 2
    result = validate(parsed_diff, findings)
    json.dump(result, sys.stdout, indent=2)
    sys.stdout.write("\n")
    print(
        "validate_findings: "
        f"{len(findings)} in, {len(result['anchored'])} anchored, "
        f"{len(result['causal'])} causal, {len(result['needs_review'])} needs review",
        file=sys.stderr,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
