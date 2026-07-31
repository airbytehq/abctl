#!/usr/bin/env python3
"""Extract structured findings from one or more review response files."""

from __future__ import annotations

import argparse
import json
import re
import tempfile
from pathlib import Path


BEGIN = "BEGIN_FINDINGS_JSON"
END = "END_FINDINGS_JSON"


def agent_name(path: Path) -> str:
    name = path.stem
    for prefix in ("review-agent-", "agent-"):
        if name.startswith(prefix):
            name = name[len(prefix) :]
    return re.sub(r"[^A-Za-z0-9_.-]+", "-", name) or "unknown"


def save_raw(name: str, raw: str) -> str:
    with tempfile.NamedTemporaryFile(
        mode="w", encoding="utf-8", prefix=f"abctl-review-raw-{name}.", suffix=".txt", delete=False
    ) as handle:
        handle.write(raw)
        return handle.name


def failure_marker(name: str, raw: str, reason: str) -> dict[str, object]:
    raw_path = save_raw(name, raw)
    return {
        "file": "__extraction_failure__",
        "line": 0,
        "severity": "P0",
        "title": f"Review output extraction failed for {name}",
        "issue": reason,
        "failure_scenario": "A reviewer response could be silently omitted from the verdict.",
        "suggestion": f"Inspect and re-parse {raw_path}.",
        "confidence": "VERIFIED",
        "category": "review_harness",
        "diff_quote": "",
        "causal_diff_quote": "",
        "extraction_failed": True,
        "raw_response": raw_path,
    }


def extract(path: Path) -> list[dict[str, object]]:
    name = agent_name(path)
    try:
        raw = path.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as exc:
        return [failure_marker(name, f"<source unreadable: {exc!r}>\n", f"Cannot read response: {exc}")]

    lines = raw.replace("\r\n", "\n").replace("\r", "\n").splitlines()
    starts = [index for index, line in enumerate(lines) if line == BEGIN]
    ends = [index for index, line in enumerate(lines) if line == END]
    if len(starts) != 1 or len(ends) != 1 or ends[0] <= starts[0]:
        return [failure_marker(name, raw, "Expected exactly one ordered findings sentinel pair.")]

    payload = "\n".join(lines[starts[0] + 1 : ends[0]]).strip()
    if not payload:
        return [failure_marker(name, raw, "The findings sentinel block is empty.")]
    try:
        parsed = json.loads(payload)
    except json.JSONDecodeError as exc:
        return [failure_marker(name, raw, f"The findings block is not valid JSON: {exc}")]
    if not isinstance(parsed, list):
        return [failure_marker(name, raw, "The findings block must contain a JSON array.")]
    if any(not isinstance(item, dict) for item in parsed):
        return [failure_marker(name, raw, "Every finding must be a JSON object.")]
    return parsed


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--agent-response-files", nargs="+", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    findings: list[dict[str, object]] = []
    for response in args.agent_response_files:
        findings.extend(extract(response))
    try:
        args.output.write_text(json.dumps(findings, indent=2) + "\n", encoding="utf-8")
    except OSError as exc:
        parser.error(f"cannot write {args.output}: {exc}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
