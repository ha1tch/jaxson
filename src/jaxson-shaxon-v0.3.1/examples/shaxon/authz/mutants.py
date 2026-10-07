#!/usr/bin/env python3
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the GNU General Public License, version 3.
# https://www.gnu.org/licenses/gpl-3.0.html
"""Mutation check for the five Shaxon authorisation packages.

Breaks one rule at a time in a scratch copy of the module and requires
TestAuthzExamples to fail. A mutant that survives means a rule can be broken
without any case noticing, which points at a missing case, not at the rule.

    python3 mutants.py            # all mutants
    python3 mutants.py -v         # also name the first case that killed each one

Each mutant is (package, text, replacement, occurrences). The package JSON is
read, written back compactly, and the text must occur exactly that many times
there before every occurrence is replaced; so an anchor that no longer matches
fails loudly instead of testing nothing. Exit status 0 if every mutant is
killed, 1 if one survives, 2 on a bad anchor or a missing Go toolchain.

Needs Go on PATH. It runs `go test` once per mutant (a few seconds each); it is
not part of the default test run. See examples/shacl/authz/SETUP.md.
"""
import json, os, re, shutil, subprocess, sys, tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
MODULE = os.path.abspath(os.path.join(HERE, "..", "..", ".."))

# (package, anchor, replacement, occurrences, what is being broken)
M = [
    # four-eyes-release
    ("four-eyes-release", '["ne",{"$v":"a"},{"$v":"c"}]', '["eq",{"$v":"a"},{"$v":"c"}]', 1,
     "an approval by the creator counts (comparison flipped)"),
    ("four-eyes-release",
     '["ne",["get_or",["get_or",{"$v":"pays"},{"$v":"p"},{"$v":"none"}],"creator",""],{"$v":"who"}]',
     '["eq",["get_or",["get_or",{"$v":"pays"},{"$v":"p"},{"$v":"none"}],"creator",""],{"$v":"who"}]', 1,
     "only the creator may release (comparison flipped)"),
    ("four-eyes-release", '["eq",{"$v":"t"},"amend"]', '["eq",{"$v":"t"},"amendX"]', 1,
     "an amendment no longer voids approvals"),
    ("four-eyes-release", '["eq",{"$v":"t"},"revoke"]', '["eq",{"$v":"t"},"revokeX"]', 1,
     "a withdrawal no longer cancels an approval"),
    ("four-eyes-release", '["has",{"$v":"ap"},{"$v":"a"}]', '["eq",1,1]', 1,
     "a withdrawal by anyone is applied, not only by an approver"),
    ("four-eyes-release",
     '["select",["has",["get_or",["get_or",{"$v":"pays"},{"$v":"p"},{"$v":"none"}],"approvers",{"$v":"none"}],{"$v":"who"}],1,0]',
     '["select",["has",["get_or",["get_or",{"$v":"pays"},{"$v":"p"},{"$v":"none"}],"approvers",{"$v":"none"}],{"$v":"who"}],0,0]', 1,
     "the releaser's own approval is not discounted"),
    # chinese-wall
    ("chinese-wall", '["eq",["sub",["len",{"$v":"inClass"}],["len",{"$v":"same"}]],0]',
     '["ge",["sub",["len",{"$v":"inClass"}],["len",{"$v":"same"}]],0]', 1,
     "the wall never fires (always true)"),
    ("chinese-wall", '["sub",["len",{"$v":"inClass"}],["len",{"$v":"same"}]]', '["len",{"$v":"inClass"}]', 1,
     "returning to the same company counts as a conflict"),
    # delegation-chain
    ("delegation-chain", '["gt",{"$v":"x"},{"$v":"at"}]', '["ge",{"$v":"x"},{"$v":"at"}]', 1,
     "a grant expiring at this instant still counts"),
    ("delegation-chain", '["lt",["len",{"$v":"u"}],{"$v":"m"}]', '["le",["len",{"$v":"u"}],{"$v":"m"}]', 1,
     "one use too many allowed"),
    ("delegation-chain", '"maxDepth":3', '"maxDepth":4', 5,
     "the chain horizon is one hop longer (all five closures)"),
    ("delegation-chain", '["eq",{"$v":"h"},{"$v":"o"}]', '["eq",{"$v":"h"},{"$v":"h"}]', 1,
     "the chain may end at anyone, not only the owner"),
    ("delegation-chain",
     '["or",["has",{"$v":"l"},"grantedBy"],["eq",{"$v":"h"},{"$v":"o"}]]', '["eq",{"$v":"h"},{"$v":"o"}]', 1,
     "the chain-continues test is dropped"),
    # rolling-quota
    ("rolling-quota", '["gt",{"$v":"t"},["sub",{"$v":"rt"},{"$v":"w"}]]', '["ge",{"$v":"t"},["sub",{"$v":"rt"},{"$v":"w"}]]', 1,
     "the window includes its lower bound"),
    ("rolling-quota", '["le",{"$v":"t"},{"$v":"rt"}]', '["lt",{"$v":"t"},{"$v":"rt"}]', 1,
     "an export at the instant of the request is ignored"),
    ("rolling-quota", '["le",["add",{"$v":"sum"},{"$v":"mb"}],{"$v":"lim"}]', '["lt",["add",{"$v":"sum"},{"$v":"mb"}],{"$v":"lim"}]', 1,
     "reaching the limit exactly is refused"),
    ("rolling-quota", '["lt",{"$v":"n"},{"$v":"max"}]', '["le",{"$v":"n"},{"$v":"max"}]', 1,
     "one export too many allowed"),
    ("rolling-quota", '["eq",{"$v":"a"},{"$v":"ra"}]', '["eq",{"$v":"a"},{"$v":"a"}]', 1,
     "every actor's exports count against this actor"),
    ("rolling-quota", '"sum":{"sum":{"$path":["local","e","mb"]}}', '"sum":{"sum":{"$lit":0}}', 1,
     "the volume is never summed"),
    # break-glass
    ("break-glass", '["lt",["len",{"$v":"prior"}],2]', '["lt",["len",{"$v":"prior"}],3]', 1,
     "a third use allowed"),
    ("break-glass", '["gt",["sub",["len",{"$v":"reviews"}],["len",{"$v":"selfReviews"}]],0]',
     '["gt",["len",{"$v":"reviews"}],0]', 1,
     "a review by oneself counts"),
    ("break-glass", '["ne",["get_or",{"$v":"e"},"actor",""],{"$v":"me"}]', '["eq",["get_or",{"$v":"e"},"actor",""],{"$v":"me"}]', 1,
     "the review rule applies to other people's events"),
    ("break-glass", '["ne",["get_or",{"$v":"e"},"type",""],"breakglass"]', '["ne",["get_or",{"$v":"e"},"type",""],"review"]', 1,
     "the review rule applies to review events, not break-glass ones"),
]


def main():
    verbose = "-v" in sys.argv
    if shutil.which("go") is None:
        sys.exit("mutants.py: go not found on PATH")
    survived, bad = [], 0
    with tempfile.TemporaryDirectory() as tmp:
        work = os.path.join(tmp, "module")
        shutil.copytree(MODULE, work, ignore=shutil.ignore_patterns(".git", ".ed-journal.json", "__pycache__", ".venv", ".jena"))
        # The check means nothing unless the untouched copy passes.
        base = subprocess.run(["go", "test", "-count=1", "-run", "TestAuthzExamples", "./pkg/shaxon/"],
                              cwd=work, capture_output=True, text=True)
        if base.returncode != 0:
            sys.exit("mutants.py: the unmodified copy fails TestAuthzExamples, so no mutant can be judged:\n"
                     + (base.stdout + base.stderr)[-800:])
        pristine = {}
        for i, (name, old, new, n, what) in enumerate(M, 1):
            path = os.path.join(work, "examples", "shaxon", "authz", name + ".json")
            if name not in pristine:
                with open(path) as f:
                    pristine[name] = f.read()
            text = json.dumps(json.loads(pristine[name]), separators=(",", ":"))
            if text.count(old) != n:
                print(f"{i:2} {name}: ANCHOR FOUND {text.count(old)} TIMES, EXPECTED {n}: {old[:60]}")
                bad += 1
                continue
            mutated = json.loads(text.replace(old, new))
            with open(path, "w") as f:
                json.dump(mutated, f, indent=2)
                f.write("\n")
            r = subprocess.run(["go", "test", "-count=1", "-run", "TestAuthzExamples", "./pkg/shaxon/"],
                               cwd=work, capture_output=True, text=True)
            with open(path, "w") as f:
                f.write(pristine[name])
            killed = r.returncode != 0
            first = ""
            if killed:
                # A case that disagrees prints "pkg: case: got X, want Y" or an error
                # mismatch; anything else (a load or compile failure) is a weaker kill.
                lines = re.findall(r"authz_examples_test\.go:\d+: (.*)", r.stdout)
                judged = [l for l in lines if ": got " in l or "want " in l or ": failed with " in l]
                if not judged:
                    first = "  (WEAK: no case judged it; the package or the test failed to run)"
                elif verbose:
                    first = "  first: " + judged[0][:110]
            print(f"{i:2} {name:18} {'killed  ' if killed else 'SURVIVED'} {what}{first}", flush=True)
            if not killed:
                survived.append((i, name, what))
    print(f"\ntried {len(M)}, killed {len(M) - len(survived) - bad}, survived {len(survived)}, bad anchors {bad}")
    sys.exit(2 if bad else (1 if survived else 0))


if __name__ == "__main__":
    main()
