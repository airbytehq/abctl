#!/usr/bin/env python3

from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent


class ReviewHarnessTest(unittest.TestCase):
    def run_script(self, script: str, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(SCRIPT_DIR / script), *args],
            check=False,
            capture_output=True,
            text=True,
        )

    def test_extracts_valid_findings(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            response = root / "review-agent-lifecycle.txt"
            output = root / "findings.json"
            finding = {
                "file": "internal/cmd/install.go",
                "line": 12,
                "severity": "P2",
                "confidence": "VERIFIED",
            }
            response.write_text(
                "summary\nBEGIN_FINDINGS_JSON\n"
                + json.dumps([finding])
                + "\nEND_FINDINGS_JSON\n",
                encoding="utf-8",
            )
            result = self.run_script(
                "extract_findings.py",
                "--agent-response-files",
                str(response),
                "--output",
                str(output),
            )
            self.assertEqual(0, result.returncode, result.stderr)
            self.assertEqual([finding], json.loads(output.read_text(encoding="utf-8")))

    def test_extraction_failure_remains_visible(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            response = root / "broken.txt"
            output = root / "findings.json"
            response.write_text("no sentinels", encoding="utf-8")
            result = self.run_script(
                "extract_findings.py",
                "--agent-response-files",
                str(response),
                "--output",
                str(output),
            )
            self.assertEqual(0, result.returncode, result.stderr)
            findings = json.loads(output.read_text(encoding="utf-8"))
            self.assertTrue(findings[0]["extraction_failed"])
            raw_response = Path(findings[0]["raw_response"])
            self.assertTrue(raw_response.exists())
            raw_response.unlink()

    def test_validation_buckets_added_removed_causal_and_unmatched(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            diff = root / "diff.patch"
            findings = root / "findings.json"
            diff.write_text(
                "diff --git a/internal/example.go b/internal/example.go\n"
                "--- a/internal/example.go\n"
                "+++ b/internal/example.go\n"
                "@@ -10,2 +10,2 @@\n"
                "-oldOperation()\n"
                "+newOperation()\n"
                " contextLine()\n",
                encoding="utf-8",
            )
            base = {
                "file": "internal/example.go",
                "severity": "P2",
                "confidence": "VERIFIED",
                "title": "Example finding",
                "issue": "The changed operation breaks the example path.",
                "failure_scenario": "The example command reaches this operation.",
                "suggestion": "Preserve the old behavior.",
                "category": "behavior",
                "diff_quote": "",
                "causal_diff_quote": "",
            }
            values = [
                {**base, "line": 10, "diff_quote": "+newOperation()"},
                {**base, "line": 10, "diff_quote": "-oldOperation()"},
                {**base, "line": 99, "causal_diff_quote": "+newOperation()"},
                {**base, "line": 10, "diff_quote": "+doesNotExist()"},
            ]
            findings.write_text(json.dumps(values), encoding="utf-8")
            result = self.run_script("validate_findings.py", str(diff), str(findings))
            self.assertEqual(0, result.returncode, result.stderr)
            buckets = json.loads(result.stdout)
            self.assertEqual(2, len(buckets["anchored"]))
            self.assertEqual(1, len(buckets["causal"]))
            self.assertEqual(1, len(buckets["needs_review"]))


if __name__ == "__main__":
    unittest.main()
