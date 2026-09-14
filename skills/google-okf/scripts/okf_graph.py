#!/usr/bin/env python3
"""
Python Fallback Query & Audit Script for OKF Knowledge Vaults.
Compliant with Google Open Knowledge Format (OKF v0.2).
"""
import argparse, os, re, sys
from pathlib import Path

try:
    import yaml
except ImportError:
    yaml = None

DEFAULT_VAULT = os.environ.get("OBSIDIAN_VAULT_PATH", ".")

def extract_frontmatter(path):
    try:
        with open(path, "r", encoding="utf-8", errors="replace") as f:
            first = f.readline()
            if not first.startswith("---"):
                return None
            lines = []
            for line in f:
                if line.startswith("---") or line.startswith("..."):
                    break
                lines.append(line)
            content = "".join(lines)
            if yaml is not None:
                return yaml.safe_load(content) or {}
            # Fallback simple parser when PyYAML is not installed
            res = {}
            for line in lines:
                if ":" in line and not line.strip().startswith("#"):
                    k, _, v = line.partition(":")
                    k = k.strip()
                    v = v.strip().strip("'\"")
                    if k:
                        res[k] = v
            return res
    except Exception:
        return None

def main():
    parser = argparse.ArgumentParser(description="OKF Vault CLI (Python Fallback)")
    parser.add_argument("--vault", default=DEFAULT_VAULT, help="Path to OKF vault")
    parser.add_argument("command", choices=["list", "get", "audit"], help="Command to run")
    parser.add_argument("target", nargs="?", default="", help="Target node for get command")
    args = parser.parse_args()

    vault = Path(args.vault)
    if not vault.exists():
        print(f"Vault path does not exist: {vault}", file=sys.stderr)
        sys.exit(1)

    files = list(vault.rglob("*.md"))
    if args.command == "audit":
        total = len(files)
        valid = sum(1 for f in files if extract_frontmatter(f) is not None)
        print(f"Audited {total} files: {valid}/{total} have valid OKF frontmatter.")
    elif args.command == "list":
        for f in sorted(files):
            fm = extract_frontmatter(f) or {}
            print(f"[{fm.get('type', 'untyped')}] {f.relative_to(vault)} ({fm.get('status', 'unknown')})")
    elif args.command == "get":
        if not args.target:
            print("Error: 'get' command requires a target note name or path.", file=sys.stderr)
            sys.exit(1)
        target = args.target.lower().strip()
        matched = None
        for f in files:
            if f.stem.lower() == target or f.name.lower() == target or target in str(f.relative_to(vault)).lower():
                matched = f
                break
        if not matched:
            print(f"Node not found: {args.target}", file=sys.stderr)
            sys.exit(1)
        fm = extract_frontmatter(matched) or {}
        print(f"Path:        {matched.relative_to(vault)}")
        for k, v in fm.items():
            print(f"{k.capitalize() + ':':<13} {v}")

if __name__ == "__main__":
    main()
