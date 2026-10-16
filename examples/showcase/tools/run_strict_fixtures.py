# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the GNU General Public License, version 3.
# https://www.gnu.org/licenses/gpl-3.0.html
"""Runs the strict numeric mode conformance fixtures through the Python cross-check executor.

The fixtures are the ones pkg/jaxson runs (jaxson-v0.1.0-strict-numbers-fixtures.json).
A fixture with `packageText` gives the package as text, in place of the package
members; the runtime is handed that text as it would be handed a file.
"""
import os
import sys

here = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, here)
import jaxsonpy as J  # noqa: E402

path = os.path.join(here, "..", "..", "..", "src", "jaxson-shaxon-v0.3.1", "pkg", "jaxson",
                    "jaxson-v0.1.0-strict-numbers-fixtures.json")
cases = J.loads(open(path, encoding="utf-8").read())
bad = 0
for c in cases:
    if "packageText" in c:
        try:
            pkg = J.loads(c["packageText"])
        except J.JaxsonError as e:
            out, err = None, e
        else:
            out, err = J.run_package(pkg)
    else:
        out, err = J.run_package(c)
    exp = c["expect"]
    if "error" in exp:
        w = exp["error"]
        ok = err is not None and err.cat == w["category"] and w.get("code", err.code) == err.code
    else:
        ok = err is None and J.jx_equal(out, exp["output"])
    if not ok:
        bad += 1
        print("FAIL", c["name"], "->", err if err is not None else J.format_decimal(out) if out is not None else out)
print("%d/%d passed" % (len(cases) - bad, len(cases)))
sys.exit(1 if bad else 0)
