#!/usr/bin/env python3
"""
Package the Fuse stage 1 compiler into a distributable archive.

Builds the Go compiler binary, the C runtime library, and bundles them
with the stdlib and runtime headers into a single directory that can be
unzipped anywhere. The user adds bin/ to their PATH and everything works.

Usage:
    python dist.py              # build for current OS/arch
    python dist.py --clean      # remove dist/ directory

Output layout:
    dist/fuse-<version>-<os>-<arch>/
        bin/fuse(.exe)
        lib/libfuse_rt.a
        include/fuse_rt.h
        stdlib/core/
        stdlib/full/
        stdlib/ext/

The archive is placed at:
    dist/fuse-<version>-<os>-<arch>.zip     (Windows)
    dist/fuse-<version>-<os>-<arch>.tar.gz  (Linux/macOS)
"""

import os
import platform
import shutil
import subprocess
import sys
import tarfile
import zipfile

# ── Constants ──────────────────────────────────────────────────────

ROOT = os.path.dirname(os.path.abspath(__file__))
VERSION = "0.1.0-dev"
DIST_DIR = os.path.join(ROOT, "dist")

# ── Colors ─────────────────────────────────────────────────────────

GREEN = "\033[92m"
RED = "\033[91m"
CYAN = "\033[96m"
BOLD = "\033[1m"
DIM = "\033[2m"
RESET = "\033[0m"


def info(msg):
    print(f"  {CYAN}>{RESET} {msg}")


def success(msg):
    print(f"  {GREEN}OK{RESET} {msg}")


def fatal(msg):
    print(f"  {RED}FAIL{RESET} {msg}", file=sys.stderr)
    sys.exit(1)


# ── Platform detection ─────────────────────────────────────────────

def detect_os():
    s = platform.system().lower()
    if s == "windows":
        return "windows"
    if s == "darwin":
        return "macos"
    return "linux"


def detect_arch():
    m = platform.machine().lower()
    if m in ("x86_64", "amd64"):
        return "x86_64"
    if m in ("aarch64", "arm64"):
        return "arm64"
    return m


# ── Build steps ────────────────────────────────────────────────────

def run(cmd, cwd=None):
    """Run a shell command, abort on failure."""
    result = subprocess.run(cmd, shell=True, cwd=cwd or ROOT,
                            capture_output=True, text=True)
    if result.returncode != 0:
        fatal(f"{cmd}\n{result.stdout}\n{result.stderr}")
    return result.stdout


def build_compiler(stage_dir, target_os):
    """Build the Go compiler binary."""
    info("building compiler binary")
    ext = ".exe" if target_os == "windows" else ""
    bin_dir = os.path.join(stage_dir, "bin")
    os.makedirs(bin_dir, exist_ok=True)
    out_path = os.path.join(bin_dir, f"fuse{ext}")
    run(f"go build -o {out_path} ./cmd/fuse")
    success(f"bin/fuse{ext}")


def build_runtime(stage_dir):
    """Build the C runtime and copy artifacts."""
    info("building runtime library")
    run("make runtime")

    lib_dir = os.path.join(stage_dir, "lib")
    inc_dir = os.path.join(stage_dir, "include")
    os.makedirs(lib_dir, exist_ok=True)
    os.makedirs(inc_dir, exist_ok=True)

    rt_lib = os.path.join(ROOT, "runtime", "libfuse_rt.a")
    rt_hdr = os.path.join(ROOT, "runtime", "include", "fuse_rt.h")

    if not os.path.isfile(rt_lib):
        fatal(f"runtime library not found: {rt_lib}")
    if not os.path.isfile(rt_hdr):
        fatal(f"runtime header not found: {rt_hdr}")

    shutil.copy2(rt_lib, os.path.join(lib_dir, "libfuse_rt.a"))
    shutil.copy2(rt_hdr, os.path.join(inc_dir, "fuse_rt.h"))
    success("lib/libfuse_rt.a + include/fuse_rt.h")


def copy_stdlib(stage_dir):
    """Copy the stdlib tree."""
    info("copying stdlib")
    src = os.path.join(ROOT, "stdlib")
    dst = os.path.join(stage_dir, "stdlib")
    if os.path.exists(dst):
        shutil.rmtree(dst)
    shutil.copytree(src, dst)
    count = sum(1 for _, _, files in os.walk(dst)
                for f in files if f.endswith(".fuse"))
    success(f"stdlib/ ({count} modules)")


def create_archive(stage_dir, dist_name, target_os):
    """Create the distributable archive."""
    if target_os == "windows":
        archive_path = os.path.join(DIST_DIR, f"{dist_name}.zip")
        info(f"creating {os.path.basename(archive_path)}")
        with zipfile.ZipFile(archive_path, "w", zipfile.ZIP_DEFLATED) as zf:
            for dirpath, _, filenames in os.walk(stage_dir):
                for fname in filenames:
                    full = os.path.join(dirpath, fname)
                    arcname = os.path.relpath(full, DIST_DIR)
                    zf.write(full, arcname)
        success(archive_path)
    else:
        archive_path = os.path.join(DIST_DIR, f"{dist_name}.tar.gz")
        info(f"creating {os.path.basename(archive_path)}")
        with tarfile.open(archive_path, "w:gz") as tf:
            tf.add(stage_dir, arcname=dist_name)
        success(archive_path)

    return archive_path


# ── Main ───────────────────────────────────────────────────────────

def main():
    if "--clean" in sys.argv:
        if os.path.exists(DIST_DIR):
            shutil.rmtree(DIST_DIR)
            print(f"cleaned {DIST_DIR}")
        return

    target_os = detect_os()
    target_arch = detect_arch()
    dist_name = f"fuse-{VERSION}-{target_os}-{target_arch}"

    print(f"{BOLD}Fuse Compiler Packaging{RESET}")
    print(f"  version:  {VERSION}")
    print(f"  platform: {target_os}-{target_arch}")
    print()

    stage_dir = os.path.join(DIST_DIR, dist_name)
    if os.path.exists(stage_dir):
        shutil.rmtree(stage_dir)
    os.makedirs(stage_dir, exist_ok=True)

    build_compiler(stage_dir, target_os)
    build_runtime(stage_dir)
    copy_stdlib(stage_dir)

    print()
    archive = create_archive(stage_dir, dist_name, target_os)

    print()
    print(f"{BOLD}Done.{RESET}")
    print(f"  archive: {archive}")
    print(f"  unzip, add {dist_name}/bin to PATH, and run: fuse version")


if __name__ == "__main__":
    main()
