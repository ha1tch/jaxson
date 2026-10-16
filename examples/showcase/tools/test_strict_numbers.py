#!/usr/bin/env python3
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the GNU General Public License, version 3.
# https://www.gnu.org/licenses/gpl-3.0.html
"""Strict numeric mode in the Python port (Jaxson core design, section 2).

The same literals as pkg/jaxson/strict_numbers_test.go: a JSON number is a
Real if, written as a plain decimal, it has at most 38 digits once leading zeros
are removed and at most 200 digits after the decimal point. One that is not is
a PARSE_ERROR, rejected in time proportional to the length of its text.
"""
import os
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import jaxsonpy  # noqa: E402

CASES = [
    ("0", True), ("-0", True), ("1", True), ("0.5", True), ("12.5E-1", True), ("1.50", True),
    ("9" * 38, True), ("-" + "9" * 38, True), ("9" * 39, False),
    ("0." + "9" * 38, True), ("0." + "9" * 39, False),
    ("1e37", True), ("1e38", False), ("-1e38", False),
    ("123456789012345678901234567890e8", True),   # 38 digits
    ("123456789012345678901234567890e9", False),  # 39 digits
    ("1000000000000000000000000000000000000e0", True),  # 10^36
    ("100e-2", True),  # 1
    ("1e-200", True), ("1e-201", False),
    ("0." + "0" * 199 + "1", True),   # 200 digits after the point
    ("0." + "0" * 200 + "1", False),   # 201
    ("1.5e-200", False), ("15e-201", False),
    ("0e999999", True), ("-0e5", True), ("0.000e-999999", True),  # zero is a Real however it is written
    ("1e999999", False), ("1e-999999", False), ("1e1000001", False), ("1e99999999999", False),
]


def rejected(doc):
    try:
        jaxsonpy.loads(doc)
    except Exception as e:  # the port raises its own error type for PARSE_ERROR
        return "number out of range" in str(e) or "PARSE_ERROR" in str(e)
    return False


def main():
    bad = 0
    for lit, is_real in CASES:
        got = not jaxsonpy.number_out_of_range(lit)
        rej = rejected("[" + lit + "]")
        if got != is_real or rej == is_real:
            print("FAIL %r: Real=%s, predicate says %s, loads rejected=%s" % (lit[:60], is_real, got, rej))
            bad += 1
    for name, doc, valid in [
        ("100 x 1e999999", "[" + ",".join(["1e999999"] * 100) + "]", False),
        ("100 x 1e-999999", "[" + ",".join(["1e-999999"] * 100) + "]", False),
        ("one 1,000,000-digit integer", "[" + "9" * 1000000 + "]", False),
        ("100 x 0e999999", "[" + ",".join(["0e999999"] * 100) + "]", True),
    ]:
        t0 = time.time()
        rej = rejected(doc)
        el = time.time() - t0
        if rej == valid or el > 5:
            print("FAIL %s: rejected=%s in %.2fs" % (name, rej, el))
            bad += 1
    if bad:
        print("%d failures" % bad)
        sys.exit(1)
    print("strict number checks: %d literals and 4 cost cases, all agree" % len(CASES))


if __name__ == "__main__":
    main()
