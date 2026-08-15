#!/usr/bin/env python3
"""Bump the application version, and keep every copy of it in step.

`build/config.yml` holds the one true version: it is embedded into the binary
and read back by `build/version.go`, so the app, the crash reports and the
update check all get it from there. Two things do not read it and carry their
own copy instead:

  * build/windows/info.json  - compiled into the exe's Windows resources by
                               `wails3 generate syso`, so Explorer shows it on
                               the Details tab.

This script moves the version in the canonical file and writes the copies to
match, so a release cannot end up with the About box and the file properties
disagreeing.

Deliberately left alone:

  * Platforms.json "Version"  - the version of that data file, compared against
                                the copy on GitHub to decide whether the
                                platform definitions need updating. It tracks
                                the platform data, not the app, and only looks
                                like the app version because they were last
                                released together.
  * build/windows/nsis/project.nsi - takes its version from `-DVERSION` on the
                                makensis command line, which the packaging task
                                does not pass, so installers currently carry
                                the 0.0.0.0 default. Fixing that belongs in the
                                Taskfile, not in another hardcoded copy here.

Usage:
    python tools/bump_version.py patch          # 4.0.4 -> 4.0.5
    python tools/bump_version.py minor          # 4.0.4 -> 4.1.0
    python tools/bump_version.py major          # 4.0.4 -> 5.0.0
    python tools/bump_version.py 4.2.0          # set it outright
    python tools/bump_version.py --print        # print the current version
    python tools/bump_version.py --check        # exit 1 if the copies disagree
    python tools/bump_version.py patch --dry-run
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CONFIG_FILE = ROOT / "build" / "config.yml"
WINDOWS_INFO_FILE = ROOT / "build" / "windows" / "info.json"

SEMVER_RE = re.compile(r"^(\d+)\.(\d+)\.(\d+)$")
# `version:` appears twice in config.yml: the Taskfile schema at the top level
# and the app version under `info:`. Only the indented one under `info:` counts.
INFO_HEADER_RE = re.compile(r"^(\s*)info:\s*$")
VERSION_LINE_RE = re.compile(r"^(?P<indent>\s*)version:\s*(?P<quote>[\"'])(?P<value>[^\"']*)(?P=quote)")


class VersionError(RuntimeError):
    """Raised when a version cannot be read or is not semver."""


def parse_semver(value: str) -> tuple[int, int, int]:
    match = SEMVER_RE.match(value.strip())
    if not match:
        raise VersionError(f"not a major.minor.patch version: {value!r}")
    return int(match[1]), int(match[2]), int(match[3])


def next_version(current: str, part: str) -> str:
    major, minor, patch = parse_semver(current)
    if part == "major":
        return f"{major + 1}.0.0"
    if part == "minor":
        return f"{major}.{minor + 1}.0"
    if part == "patch":
        return f"{major}.{minor}.{patch + 1}"
    # Anything else is an explicit version, validated before it is written.
    parse_semver(part)
    return part.strip()


def find_config_version_line(lines: list[str]) -> int:
    """Index of the `version:` line inside the `info:` block of config.yml.

    Mirrors what build/version.go does at runtime: enter `info:`, leave it as
    soon as the indentation returns to that level or less.
    """
    in_info = False
    info_indent = 0
    for index, raw in enumerate(lines):
        stripped = raw.strip()
        if not stripped or stripped.startswith("#"):
            continue
        indent = len(raw) - len(raw.lstrip(" "))
        header = INFO_HEADER_RE.match(raw)
        if header:
            in_info = True
            info_indent = indent
            continue
        if in_info and indent <= info_indent:
            in_info = False
        if in_info and VERSION_LINE_RE.match(raw):
            return index
    raise VersionError(f"no version under info: in {CONFIG_FILE}")


def read_config_version() -> tuple[list[str], int, str]:
    text = CONFIG_FILE.read_text(encoding="utf-8")
    lines = text.splitlines()
    index = find_config_version_line(lines)
    match = VERSION_LINE_RE.match(lines[index])
    assert match is not None
    return lines, index, match["value"]


def write_config_version(lines: list[str], index: int, version: str) -> None:
    """Replaces only the quoted value, so the trailing comment survives."""
    match = VERSION_LINE_RE.match(lines[index])
    assert match is not None
    quote = match["quote"]
    lines[index] = (
        f"{match['indent']}version: {quote}{version}{quote}{lines[index][match.end():]}"
    )
    CONFIG_FILE.write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")


def windows_info_versions(data: dict) -> list[str]:
    found = [str(data.get("fixed", {}).get("file_version", ""))]
    for block in data.get("info", {}).values():
        if isinstance(block, dict) and "ProductVersion" in block:
            found.append(str(block["ProductVersion"]))
    return [v for v in found if v]


def write_windows_info(version: str) -> bool:
    """Returns True when the file needed changing."""
    original = WINDOWS_INFO_FILE.read_text(encoding="utf-8")
    data = json.loads(original)
    data.setdefault("fixed", {})["file_version"] = version
    for block in data.get("info", {}).values():
        if isinstance(block, dict) and "ProductVersion" in block:
            block["ProductVersion"] = version
    # Tabs, escaped non-ASCII and a trailing newline, matching how wails3 writes
    # this file. Without ensure_ascii the copyright sign would be rewritten and
    # the diff would show more than the version.
    updated = json.dumps(data, indent="\t", ensure_ascii=True) + "\n"
    if updated == original:
        return False
    WINDOWS_INFO_FILE.write_text(updated, encoding="utf-8", newline="\n")
    return True


def check(current: str) -> int:
    mismatches = []
    data = json.loads(WINDOWS_INFO_FILE.read_text(encoding="utf-8"))
    for found in windows_info_versions(data):
        if found != current:
            mismatches.append(f"{WINDOWS_INFO_FILE.relative_to(ROOT)}: {found}")
    if mismatches:
        print(f"version in {CONFIG_FILE.relative_to(ROOT)} is {current}, but:", file=sys.stderr)
        for line in mismatches:
            print(f"  {line}", file=sys.stderr)
        print("run: python tools/bump_version.py " + current, file=sys.stderr)
        return 1
    print(f"{current} everywhere")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument(
        "part",
        nargs="?",
        help="major, minor, patch, or an explicit x.y.z version",
    )
    parser.add_argument("--print", action="store_true", help="print the current version and exit")
    parser.add_argument("--check", action="store_true", help="verify every copy matches")
    parser.add_argument("--dry-run", action="store_true", help="report what would change")
    args = parser.parse_args(argv)

    try:
        lines, index, current = read_config_version()
    except (OSError, VersionError) as e:
        print(f"error: {e}", file=sys.stderr)
        return 1

    if args.print:
        print(current)
        return 0

    if args.check:
        return check(current)

    if not args.part:
        parser.error("give major, minor, patch, an explicit version, --print or --check")

    try:
        target = next_version(current, args.part)
    except VersionError as e:
        print(f"error: {e}", file=sys.stderr)
        return 1

    if args.dry_run:
        print(f"{current} -> {target} (dry run, nothing written)")
        return 0

    write_config_version(lines, index, target)
    write_windows_info(target)
    print(f"{current} -> {target}")
    print(f"  {CONFIG_FILE.relative_to(ROOT)}")
    print(f"  {WINDOWS_INFO_FILE.relative_to(ROOT)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
