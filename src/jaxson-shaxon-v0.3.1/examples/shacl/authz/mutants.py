#!/usr/bin/env python3
"""Mutation check for the SHACL rules: break one rule at a time in a scratch
copy of examples/ and require run_cases.py to notice (exit non-zero).

    .venv/bin/python mutants.py

Each mutant is (shapes file, exact text, replacement); the text must occur
exactly once. Exit 0 if every mutant is killed, 1 if any survives.
"""
import os, shutil, subprocess, sys, tempfile
EXAMPLES = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..")
PY = sys.executable
M=[
("rolling-quota","?t > ?rt - ?window && ?t <= ?rt)\n            }\n        }\n        GROUP BY $this ?rmb","?t >= ?rt - ?window && ?t <= ?rt)\n            }\n        }\n        GROUP BY $this ?rmb"),
("rolling-quota","?t > ?rt - ?window && ?t <= ?rt)\n            }\n        }\n        GROUP BY $this ?rmb","?t > ?rt - ?window && ?t < ?rt)\n            }\n        }\n        GROUP BY $this ?rmb"),
("rolling-quota","+ ?rmb > ?limit","+ ?rmb >= ?limit"),
("rolling-quota","COALESCE(SUM(?mb), 0) + ?rmb","COALESCE(SUM(?mb), 0)"),
("rolling-quota","HAVING (COUNT(?e) >= ?max)","HAVING (COUNT(?e) > ?max)"),
("rolling-quota","?e ex:actor ?actor ; ex:t ?t ; ex:mb ?mb .","?e ex:t ?t ; ex:mb ?mb ."),
("four-eyes-release","?approver != ?creator && ?approver != ?releaser && ?aseq > ?cseq","?approver != ?releaser && ?aseq > ?cseq"),
("four-eyes-release","?approver != ?creator && ?approver != ?releaser && ?aseq > ?cseq","?approver != ?creator && ?aseq > ?cseq"),
("four-eyes-release","?approver != ?creator && ?approver != ?releaser && ?aseq > ?cseq","?approver != ?creator && ?approver != ?releaser"),
("four-eyes-release","FILTER (?mseq > ?aseq)","FILTER (?mseq >= 0)"),
("four-eyes-release","ex:actor ?approver ; ex:seq ?vseq . FILTER (?vseq > ?aseq)","ex:seq ?vseq . FILTER (?vseq > ?aseq)"),
("chinese-wall","FILTER (?other != ?company)","FILTER (true)"),
("chinese-wall","?k2 ex:id ?other   ; ex:class ?class .","?k2 ex:id ?other ."),
("break-glass","FILTER (?reviewer != ?actor)","FILTER (true)"),
("break-glass","HAVING (COUNT(?e) >= 2)","HAVING (COUNT(?e) >= 3)"),
("break-glass","?root ex:request ?req .\n            ?req ex:actor ?actor .","?root ex:request ?req ."),
("delegation-chain","FILTER (?expires <= ?at)","FILTER (?expires < ?at)"),
("delegation-chain","HAVING (COUNT(?use) >= ?max)","HAVING (COUNT(?use) > ?max)"),
("delegation-chain","FILTER (?holder != ?owner)","FILTER (true)"),
]
killed=0; surv=[]
for i,(n,old,new) in enumerate(M,1):
    d=tempfile.mkdtemp(prefix="mutant")
    shutil.copytree(EXAMPLES,d+"/examples",ignore=shutil.ignore_patterns('__pycache__','.venv'))
    p=f"{d}/examples/shacl/authz/{n}.shapes.ttl"
    t=open(p).read()
    if t.count(old)!=1: print(i,n,"ANCHOR COUNT",t.count(old)); sys.exit(2)
    open(p,"w").write(t.replace(old,new))
    r=subprocess.run([PY,f"{d}/examples/shacl/authz/run_cases.py"],capture_output=True,text=True)
    if r.returncode!=0: killed+=1; print(i,n,"killed")
    else: surv.append((i,n,new[:50])); print(i,n,"SURVIVED")
    shutil.rmtree(d,ignore_errors=True)
print("tried",len(M),"killed",killed,"survived",surv)
sys.exit(1 if surv else 0)
