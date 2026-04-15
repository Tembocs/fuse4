#!/usr/bin/env python3
"""
Fuse compiler test suite — runs all tests in dependency order.

Usage:
    python test_all.py          # run all steps
    python test_all.py --from 3 # start from step 3
    python test_all.py --step 5 # run only step 5
    python test_all.py -v       # verbose (show stdout on pass)
"""

import subprocess
import sys
import time
import os
import argparse

# ── Colors ──────────────────────────────────────────────────────────

GREEN = "\033[92m"
RED = "\033[91m"
YELLOW = "\033[93m"
CYAN = "\033[96m"
BOLD = "\033[1m"
RESET = "\033[0m"

def ok(msg):
    print(f"  {GREEN}PASS{RESET} {msg}")

def fail(msg):
    print(f"  {RED}FAIL{RESET} {msg}")

def header(step, title):
    print(f"\n{BOLD}{CYAN}Step {step}: {title}{RESET}")

# ── Runner ──────────────────────────────────────────────────────────

def run(cmd, cwd=None, timeout=300):
    """Run a command, return (success, stdout, stderr, elapsed)."""
    start = time.time()
    try:
        result = subprocess.run(
            cmd, shell=True, cwd=cwd, timeout=timeout,
            capture_output=True, text=True
        )
        elapsed = time.time() - start
        success = result.returncode == 0
        return success, result.stdout, result.stderr, elapsed
    except subprocess.TimeoutExpired:
        return False, "", "TIMEOUT", time.time() - start

# ── Steps ───────────────────────────────────────────────────────────

ROOT = os.path.dirname(os.path.abspath(__file__))

def step1_build(verbose):
    """Go build — does the compiler source compile?"""
    header(1, "Go build (compiler source compiles?)")
    success, out, err, elapsed = run("go build ./...", cwd=ROOT)
    if success:
        ok(f"go build ./... ({elapsed:.1f}s)")
    else:
        fail(f"go build ./... ({elapsed:.1f}s)")
        print(err)
    return success

def step2_unit_tests(verbose):
    """Unit tests per package in dependency order."""
    header(2, "Unit tests (per package, dependency order)")
    packages = [
        "compiler/lex",
        "compiler/parse",
        "compiler/ast",
        "compiler/resolve",
        "compiler/typetable",
        "compiler/diagnostics",
        "compiler/check",
        "compiler/hir",
        "compiler/liveness",
        "compiler/lower",
        "compiler/mir",
        "compiler/monomorph",
        "compiler/codegen",
        "compiler/cc",
        "cmd/fuse",
    ]
    all_pass = True
    for pkg in packages:
        success, out, err, elapsed = run(
            f"go test -count=1 ./{pkg}/", cwd=ROOT, timeout=60
        )
        name = pkg.split("/")[-1]
        if success:
            ok(f"{name} ({elapsed:.1f}s)")
            if verbose:
                print(out)
        else:
            fail(f"{name} ({elapsed:.1f}s)")
            print(out + err)
            all_pass = False
            break  # stop on first failure
    return all_pass

def step3_runtime(verbose):
    """Build the C runtime library."""
    header(3, "Runtime build (C runtime compiles?)")
    success, out, err, elapsed = run("make runtime", cwd=ROOT, timeout=60)
    if success:
        ok(f"make runtime ({elapsed:.1f}s)")
        if verbose:
            print(out)
    else:
        fail(f"make runtime ({elapsed:.1f}s)")
        print(out + err)
    return success

def step4_driver_tests(verbose):
    """Driver tests — stdlib loading, doc coverage, bootstrap gate."""
    header(4, "Driver tests (stdlib loading, bootstrap gate)")
    success, out, err, elapsed = run(
        "go test -count=1 ./compiler/driver/", cwd=ROOT, timeout=120
    )
    if success:
        ok(f"compiler/driver ({elapsed:.1f}s)")
        if verbose:
            print(out)
    else:
        fail(f"compiler/driver ({elapsed:.1f}s)")
        print(out + err)
    return success

def step5_stdlib_compile(verbose):
    """Stdlib compilation — all .fuse files parse + resolve + check."""
    header(5, "Stdlib compilation (53 modules parse + resolve + check)")
    tests = [
        "TestStdlibCoreCompiles",
        "TestStdlibFullCompiles",
        "TestStdlibExtCompiles",
    ]
    all_pass = True
    for test in tests:
        success, out, err, elapsed = run(
            f"go test -count=1 -run {test} -v ./tests/e2e/", cwd=ROOT, timeout=60
        )
        label = test.replace("TestStdlib", "").replace("Compiles", "").lower()
        if success:
            count = out.count("--- PASS:")
            ok(f"stdlib/{label} ({count} modules, {elapsed:.1f}s)")
        else:
            fail(f"stdlib/{label} ({elapsed:.1f}s)")
            print(out + err)
            all_pass = False
            break
    return all_pass

def step6_e2e_compile_and_run(verbose):
    """E2e tests — compile Fuse -> C -> gcc -> run -> verify exit codes + stdout."""
    header(6, "E2e tests (Fuse -> C -> gcc -> execute -> verify)")
    success, out, err, elapsed = run(
        "go test -count=1 -run TestE2E$ -v ./tests/e2e/", cwd=ROOT, timeout=600
    )
    if success:
        pass_count = out.count("--- PASS: TestE2E/")
        fail_count = out.count("--- FAIL: TestE2E/")
        ok(f"{pass_count} programs compiled and ran correctly ({elapsed:.1f}s)")
        if verbose:
            for line in out.splitlines():
                if "--- PASS" in line or "--- FAIL" in line:
                    print(f"    {line.strip()}")
    else:
        fail(f"e2e tests ({elapsed:.1f}s)")
        # Show failing tests
        for line in out.splitlines():
            if "--- FAIL" in line or "FAIL" in line:
                print(f"    {line.strip()}")
        if verbose:
            print(out)
    return success

def step7_bootstrap(verbose):
    """Bootstrap gate — stage1 compiles stage2, stage2 compiles itself."""
    header(7, "Bootstrap gate (stage1 -> stage2 -> stage2)")
    tests = [
        ("TestStage1CompilesStage2", "stage1 -> stage2"),
        ("TestStage2CompilesItself", "stage2 -> stage2"),
        ("TestBootstrapHealthGate", "bootstrap health"),
    ]
    all_pass = True
    for test, label in tests:
        success, out, err, elapsed = run(
            f"go test -count=1 -run {test} ./compiler/driver/", cwd=ROOT, timeout=120
        )
        if success:
            ok(f"{label} ({elapsed:.1f}s)")
        else:
            fail(f"{label} ({elapsed:.1f}s)")
            if verbose:
                print(out + err)
            all_pass = False
    return all_pass

# ── Main ────────────────────────────────────────────────────────────

STEPS = [
    (1, "Go build",             step1_build),
    (2, "Unit tests",           step2_unit_tests),
    (3, "Runtime build",        step3_runtime),
    (4, "Driver tests",         step4_driver_tests),
    (5, "Stdlib compilation",   step5_stdlib_compile),
    (6, "E2e compile & run",    step6_e2e_compile_and_run),
    (7, "Bootstrap gate",       step7_bootstrap),
]

def main():
    parser = argparse.ArgumentParser(description="Fuse compiler test suite")
    parser.add_argument("--from", type=int, default=1, dest="start",
                        help="Start from step N")
    parser.add_argument("--step", type=int, default=None,
                        help="Run only step N")
    parser.add_argument("-v", "--verbose", action="store_true",
                        help="Show stdout on pass")
    args = parser.parse_args()

    print(f"{BOLD}Fuse Compiler Test Suite{RESET}")
    print(f"Root: {ROOT}")

    total_start = time.time()
    passed = 0
    failed = 0

    for num, name, fn in STEPS:
        if args.step is not None and num != args.step:
            continue
        if num < args.start:
            continue

        success = fn(args.verbose)
        if success:
            passed += 1
        else:
            failed += 1
            if args.step is None:
                print(f"\n{RED}{BOLD}Stopped at step {num}: {name}{RESET}")
                break

    total = time.time() - total_start
    print(f"\n{BOLD}{'='*50}{RESET}")
    if failed == 0:
        print(f"{GREEN}{BOLD}ALL {passed} STEPS PASSED{RESET} ({total:.1f}s)")
    else:
        print(f"{RED}{BOLD}{failed} STEP(S) FAILED{RESET}, {passed} passed ({total:.1f}s)")

    sys.exit(0 if failed == 0 else 1)

if __name__ == "__main__":
    main()
