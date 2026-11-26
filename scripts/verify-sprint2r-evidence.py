"""Verify local Sprint 2R evidence using only Git and Python's standard library."""
import argparse
import hashlib
import json
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EVIDENCE = ROOT / "docs/audit/sprint-2r-evidence"


def sha(data):
    return hashlib.sha256(data).hexdigest()


def source_fingerprint():
    """Hash the sorted runtime input paths and their LF-normalized content hashes."""
    raw = subprocess.check_output(
        ["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard"], cwd=ROOT
    )
    inputs = []
    for name in sorted(x.decode() for x in raw.split(b"\0") if x):
        if name.startswith("docs/") or name in {".gitignore", "README.md"}:
            continue
        data = (ROOT / name).read_bytes().replace(b"\r\n", b"\n")
        inputs.append(f"{sha(data)}  {name}\n")
    return sha("".join(inputs).encode())


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--fingerprint", action="store_true")
    args = parser.parse_args()
    fingerprint = source_fingerprint()
    print("Candidate source SHA-256", fingerprint)
    if args.fingerprint:
        return
    index = json.loads((EVIDENCE / "index.json").read_text(encoding="utf-8"))
    if fingerprint != index["source_sha256"]:
        raise SystemExit("Candidate source differs from the evidence register")
    for run in index["runs"]:
        if sha((EVIDENCE / run["log"]).read_bytes()) != run["sha256"]:
            raise SystemExit(f"Evidence checksum mismatch in {run['log']}")
    print(f"Verified {len(index['runs'])} evidence files")


if __name__ == "__main__":
    main()
