# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Cross-check the Go pipeline against the Python reference oracle.

Generates databases with `agogo gen`, evaluates each with oracle.py and with
both Go oracles, and requires identical decisions.

  ACCESSGRAPH_DIR  directory holding oracle.py and synth.py (default: the example directory above)
  AGOGO            path of the built agogo binary (default: the binary next to this script)
"""
import json, os, subprocess, sys, tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.environ.get("ACCESSGRAPH_DIR", os.path.join(HERE, "..")))
import oracle  # noqa: E402

AG = os.environ.get("AGOGO", os.path.join(HERE, "agogo"))


def sh(args):
    r = subprocess.run(args, capture_output=True, text=True)
    if r.returncode:
        raise RuntimeError(" ".join(args) + ": " + r.stderr[:300])
    return r.stdout


plan = [(s, z) for s in range(12) for z in ("tiny", "small", "medium")]
plan += [(0, "large"), (1, "large"), (0, "xlarge")]
bad = dec = 0
for seed, size in plan:
    txt = sh([AG, "gen", "-seed", str(seed), "-size", size] + (["-nocycles"] if seed % 5 == 4 else []))
    db = json.loads(txt)
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        f.write(txt)
    try:
        want = oracle.evaluate(db)
        fast = json.loads(sh([AG, "eval", "-impl", "fast", f.name]))
        ref = json.loads(sh([AG, "eval", "-impl", "ref", f.name]))
    finally:
        os.unlink(f.name)
    dec += len(want)
    if not (want == fast == ref):
        bad += 1
        print("MISMATCH", seed, size)
print("%d databases, %d decisions, %d mismatches" % (len(plan), dec, bad))
sys.exit(1 if bad else 0)
