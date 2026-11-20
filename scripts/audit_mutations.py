#!/usr/bin/env python3
"""Recheck gremlins reports by compiling and executing each mutant separately."""

import json
import re
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
REPORTS = {
    "domain-trip": "internal/domain/trip",
    "usecase-trip-create": "internal/usecase/trip/create",
    "usecase-trip-finish": "internal/usecase/trip/finish",
}
REPLACEMENTS = {
    "CONDITIONALS_NEGATION": {"==": "!=", "!=": "==", ">=": "<", ">": "<=", "<=": ">", "<": ">="},
    "CONDITIONALS_BOUNDARY": {">=": ">", ">": ">=", "<=": "<", "<": "<="},
    "ARITHMETIC_BASE": {"+": "-", "-": "+", "*": "/", "/": "*", "%": "*"},
    "INVERT_NEGATIVES": {"-": "+"},
    "INVERT_LOGICAL": {"&&": "||", "||": "&&"},
}


def run(command, timeout):
    try:
        return subprocess.run(command, cwd=ROOT, text=True, stdout=subprocess.PIPE,
                              stderr=subprocess.STDOUT, timeout=timeout, check=False)
    except subprocess.TimeoutExpired as exc:
        raise RuntimeError(f"command timed out: {command}") from exc


def main():
    results = []
    with tempfile.TemporaryDirectory(prefix="trip-mutation-audit-") as directory:
        temp = Path(directory)
        for report_name, package in REPORTS.items():
            report = json.loads((ROOT / "docs/mutation-results" / f"{report_name}.json").read_text())
            for file in report["files"]:
                source = ROOT / package / file["file_name"]
                original = source.read_bytes()
                lines = original.splitlines(keepends=True)
                for mutant in file["mutations"]:
                    line, column = mutant["line"], mutant["column"]
                    offset = sum(map(len, lines[:line - 1])) + column - 1
                    mapping = REPLACEMENTS[mutant["type"]]
                    token = next((value for value in sorted(mapping, key=len, reverse=True)
                                  if original[offset:].startswith(value.encode())), None)
                    if token is None:
                        raise ValueError(f"unknown token: {source}:{line}:{column} {mutant['type']}")
                    replacement = mapping[token]
                    mutated = original[:offset] + replacement.encode() + original[offset + len(token):]
                    source_copy = temp / "mutated.go"
                    source_copy.write_bytes(mutated)
                    overlay = temp / "overlay.json"
                    overlay.write_text(json.dumps({"Replace": {str(source): str(source_copy)}}))
                    binary = temp / "test-binary"
                    compile_result = run(["go", "test", "-c", "-vet=off", "-overlay", str(overlay),
                                          "-o", str(binary), f"./{package}"], 60)
                    entry = {"package": package, "file": file["file_name"], "line": line,
                             "column": column, "type": mutant["type"], "original": token,
                             "replacement": replacement}
                    if compile_result.returncode:
                        entry.update(status="compile_error", output=compile_result.stdout.strip().replace(str(source_copy), str(source.relative_to(ROOT))))
                    else:
                        test_result = run([str(binary), "-test.v", "-test.failfast", "-test.timeout=15s"], 20)
                        failures = re.findall(r"^--- FAIL: (\S+)", test_result.stdout, re.MULTILINE)
                        entry.update(status="test_failed" if test_result.returncode else "survived",
                                     failed_tests=failures)
                        if test_result.returncode and not failures:
                            entry["output"] = test_result.stdout.strip()
                    results.append(entry)
                    print(f"{len(results):2}: {package}:{line}:{column} {mutant['type']} {entry['status']}", flush=True)
    output = ROOT / "docs/mutation-results/audit.json"
    output.write_text(json.dumps(results, ensure_ascii=False, indent=2) + "\n")
    counts = {status: sum(result["status"] == status for result in results)
              for status in ("test_failed", "compile_error", "survived")}
    print(json.dumps(counts, ensure_ascii=False))
    return 0 if all(result["status"] == "compile_error" or
                    (result["status"] == "test_failed" and result["failed_tests"])
                    for result in results) else 1


if __name__ == "__main__":
    sys.exit(main())
