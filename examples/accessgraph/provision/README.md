# External datasets for checking accessgraph

Two datasets the accessgraph generator did not produce, from different
organizations and built for different purposes, so that any agreement with the
Jaxson program is not an artifact of one data source. `./fetch.sh [DATA_DIR]`
downloads about 13 MB, verifies it, and keeps about 70 MB, by default under
`examples/accessgraph/data/`, which git ignores.

| | Amazon Access Samples | OpenFGA sample stores |
|---|---|---|
| Origin | Anonymized access provisioned inside one company, published through the UCI Machine Learning Repository (dataset 216, 2011) | Authorization models written by a vendor to teach and test a Zanzibar-style engine |
| Nature | Real data | Hand-built models with expected answers |
| Terms | CC BY 4.0 | Apache-2.0 |
| Pinned by | SHA-256 `d493e337...b8228d224` of the zip (12,268,663 bytes) | commit `6802b4d8...2893a0` (2026-10-09) |
| Content | 36,063 people with 12 attributes each (manager, three groupings, department, title, location and others), and 33,252 binary columns: 21,921 `SYSTEM_SUPPORT` resources, 9,117 `PERMGROUP` groups, 2,214 `HOST`s. A second file holds 716,063 access change events (action, target, login, request date, authorization date). | 40 stores. 481 relationship tuples, 556 `check` assertions and 143 `list_objects` assertions in all. Includes GitHub, Google Drive, multi-tenant RBAC with nested groups, custom roles and temporal access. |
| What it can test | The shape of real data: grants per person, people per group, resources per person, deep manager and grouping chains. It has no role inheritance, resource tree, deny or delegation. | Independent expected answers from a third party, for checking the rules themselves. Its model (relationships) differs from ours, so each store needs a mapping and some will not map. |

## Do not extract the Amazon CSV

The archive is 12 MB compressed. Its main file expands to 4.8 GB because it is
a wide matrix that is almost entirely zeros. `amazon_sample.py` reads it from
the compressed stream in one pass (one to three minutes, nothing extracted) and
writes two files into `amazon-access/derived/` for all 36,063 people: `person_attrs.tsv` (2 MB) and
`edges.tsv` (21 MB, 595,506 edges: 406,661 system, 185,318 permission group,
3,527 host).

The file is not segmented. Every row holds a person's attributes and all of
their access, in person-id order, and the statistics of its three thirds agree
(groups per person 5.1 to 5.2, people with no group 12.8% to 13.5%). Set
`SAMPLE_PEOPLE` to read only the first rows. The first 2,000 are not
representative of group sizes: the largest group has 225 people there against
3,882 in the whole file, and the median group has 1 person there against 7.

The OpenFGA checkout is sparse: the models, the licence and the readme only.

## From the Amazon data to a database

`python3 amazon_to_db.py` writes `../data/amazon-db/flat.json`, `nested.json` and
`record.json` from the derived files, in about 12 seconds, with a fixed seed. The
people, their permission-group memberships and their direct grants of systems and
hosts come from the data; the roles, a three-level resource tree, group grants,
denies, delegations and requests are added from the generator's distributions, and
`record.json` counts each. `python3 check_amazon.py` checks the databases against the
policy limits and prints the declared step limit the cost model derives; with
`--shaxonrun`, `--run-program --jaxrun` and `--steps-jaxrun` it also runs the
integrity gate and the program. The results are in `FINDINGS.md`, section 2.
