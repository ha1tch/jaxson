# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Stream the first N people out of the Amazon Access Samples archive.

The archive is 12 MB, but its main CSV is a 4.8 GB wide 0/1 matrix (36,063
people by 33,252 columns). This reads it straight from the compressed stream,
stops after N people, and writes two small files. The wide CSV is never written
to disk.

  usage: amazon_sample.py ARCHIVE.tgz OUTDIR [N]      (N defaults to 2000)

Outputs, in OUTDIR:
  person_attrs.tsv   the 12 PERSON_* attributes, one row per person
  edges.tsv          PERSON_ID, KIND (system|permgroup|host), COLUMN, VALUE
                     for every nonzero cell

The people are the first N rows of the file, not a random sample.
"""
import collections
import os
import subprocess
import sys

MEMBER = "amzn-anon-access-samples-2.0.csv"


def main(argv):
    if len(argv) < 3:
        sys.stderr.write(__doc__)
        return 2
    archive, outdir = argv[1], argv[2]
    limit = int(argv[3]) if len(argv) > 3 else 2000
    os.makedirs(outdir, exist_ok=True)
    p = subprocess.Popen(["tar", "-xzOf", archive, MEMBER], stdout=subprocess.PIPE, bufsize=1 << 20)
    hdr = p.stdout.readline().decode().rstrip("\n").split(",")
    pidx = [i for i, c in enumerate(hdr) if c.startswith("PERSON_")]
    pset = set(pidx)
    kind = {}
    for i, c in enumerate(hdr):
        if i in pset:
            continue
        kind[i] = ("system" if c.startswith("SYSTEM_SUPPORT")
                   else "permgroup" if c.startswith("PERMGROUP")
                   else "host" if c.startswith("HOST") else "other")
    idcol = hdr.index("PERSON_ID")
    per = collections.Counter()
    people = edges = 0
    with open(os.path.join(outdir, "person_attrs.tsv"), "w") as pa, \
            open(os.path.join(outdir, "edges.tsv"), "w") as ed:
        pa.write("\t".join(hdr[i] for i in pidx) + "\n")
        ed.write("PERSON_ID\tKIND\tCOLUMN\tVALUE\n")
        for raw in p.stdout:
            # cells are quoted in the data rows ("0"), unlike the header
            cells = raw.decode().rstrip("\n").replace('"', "").split(",")
            if len(cells) != len(hdr):
                raise SystemExit("ragged row %d: %d cells, expected %d" % (people, len(cells), len(hdr)))
            pid = cells[idcol]
            pa.write("\t".join(cells[i] for i in pidx) + "\n")
            for i, v in enumerate(cells):
                if i not in pset and v != "0" and v != "":
                    ed.write("%s\t%s\t%s\t%s\n" % (pid, kind[i], hdr[i], v))
                    per[kind[i]] += 1
                    edges += 1
            people += 1
            if people >= limit:
                break
    p.stdout.close()
    p.terminate()
    p.wait()
    print("people %d, edges %d %s" % (people, edges, dict(per)))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
