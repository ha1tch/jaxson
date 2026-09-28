"""Runs showcase-fixtures.json through the Python cross-check executor."""
import os, sys
here = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, here)
import jaxsonpy as J

path = os.path.join(here, "..", "showcase-fixtures.json")
cases = J.loads(open(path, encoding="utf-8").read())
bad = 0
for c in cases:
    out, err = J.run_package(c)
    exp = c["expect"]
    if "error" in exp:
        w = exp["error"]
        ok = err is not None and err.cat == w["category"] and w.get("code", err.code) == err.code
    else:
        ok = err is None and J.jx_equal(out, exp["output"])
    if not ok:
        bad += 1
        print("FAIL", c["name"])
print(f"{len(cases) - bad}/{len(cases)} passed")
sys.exit(1 if bad else 0)
