# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""The policy of the accessgraph example: limits that are part of the rules.

Jaxson has no unbounded loop, so "follow memberships until nothing changes" is
written as a fixed number of rounds. Those numbers are declared here, once,
and are not measured from any database:

  * the generated Jaxson program is the same for every database;
  * the reference (oracle.py) is defined on databases within these limits, and
    refuses a deeper one;
  * the generator only produces databases within them;
  * the program asserts, with one extra round, that its input fits, so data
    deeper than a limit fails with ASSERTION_FAILED and is never answered from
    a truncated closure.

Every database pays for the full number of rounds, deep or shallow; that is
the price of a program that does not change with its data. A larger limit
costs steps for everyone, a smaller one rejects more data.

Grant and delegation ids are also policy: ID_WIDTH digits after their letter
(G000001, D000001). They appear in chain strings and in sorted lists, where a
mixed width would sort in an order nobody intended and an engine that encodes
a chain as numbers would overflow, so a database with a grant or delegation id
of another width is refused by the reference, the generator and the gate. The
other ids (users, groups, roles, resources, requests) are opaque.

Measured depths behind the values (true bounds over generated databases): the
longest membership chain grows slowly with size, to 12 rounds at 5,000 users;
role inheritance reached 5; the generator caps resource depth at 6.
"""

MAX_HOPS = 3          # a delegation chain may be at most this long
GROUP_ROUNDS = 16     # membership: longest shortest path from a user to a subject
ROLE_ROUNDS = 8       # role inheritance: longest shortest path between roles
RES_DEPTH = 6         # deepest resource below its root (the root is depth 0)

ID_WIDTH = 6          # digits in a grant or delegation id, after the letter: G000001, D000001
MAX_IDS = 10 ** ID_WIDTH - 1

LIMITS = {"group_rounds": GROUP_ROUNDS, "role_rounds": ROLE_ROUNDS, "res_depth": RES_DEPTH}


def grant_id(n):
    return "G%0*d" % (ID_WIDTH, n)


def deleg_id(n):
    return "D%0*d" % (ID_WIDTH, n)
