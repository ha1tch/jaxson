#!/usr/bin/env python3
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the Apache License, Version 2.0.
"""Generates navywars.json, a Jaxson package that plays one turn of Navy Wars.

The game is a pure function: input {action, seed | x, y, game} -> output
{game, view, messages}. The host (jaxrun play) owns the loop and the seed.
"""
import json

Z36 = [0] * 36
K36 = list(range(36))
ROWS = list(range(6))

# Fleet layouts: three ships each, cells are y*6+x. Sizes 3, 2, 2.
LAYOUTS = [
    [[0, 1, 2], [11, 17], [26, 27]],
    [[12, 18, 24], [4, 5], [33, 34]],
    [[19, 20, 21], [23, 29], [0, 1]],
    [[3, 9, 15], [30, 31], [23, 29]],
]
NAMES = ["Cruiser", "Destroyer", "Patrol boat"]
HP0 = [3, 2, 2]
for lay in LAYOUTS:
    cells = [c for ship in lay for c in ship]
    assert len(cells) == len(set(cells)) and all(0 <= c < 36 for c in cells)
    assert [len(s) for s in lay] == HP0


def P(*segs): return {"$path": list(segs)}
def V(n): return {"$v": n}
def C(expr, **binds):
    body = {"expr": expr}
    if binds:
        body["with"] = binds
    return {"$compute": body}
def SET(path, value): return {"op": "set", "path": path, "value": value}
def APPEND(path, value): return {"op": "append", "path": path, "value": value}
def ASSERT(that, msg): return {"op": "assert", "that": that, "msg": msg}
def IF(cond, then, els=None):
    d = {"op": "if", "cond": cond, "then": then}
    if els is not None:
        d["else"] = els
    return d
def FOR(inp, name, do, index=None):
    d = {"op": "for", "in": inp, "as": name}
    if index:
        d["index"] = index
    d["do"] = do
    return d


G = ["state", "game"]
MSGS = ["state", "msgs"]
LINES = ["state", "lines"]
def T(n): return P("state", "tmp", n)
def TP(n): return ["state", "tmp", n]
def SUM3(v): return ["add", ["get", V(v), 0], ["get", V(v), 1], ["get", V(v), 2]]
def AFLOAT(v): return ["add"] + [["select", ["gt", ["get", V(v), i], 0], 1, 0] for i in range(3)]


# ---------------------------------------------------------------- new game
def place(board, layout_path):
    layout = C(["get", V("L"), V("i")], L={"$lit": LAYOUTS}, i=P(*layout_path))
    return FOR(layout, "ship", [
        FOR(P("local", "ship"), "c", [
            SET(G + [board, P("local", "c")], C(["add", V("s"), 1], s=P("local", "s")))
        ])
    ], index="s")

NEW = [
    SET(G, {"$tpl": {"seed": P("input", "seed"), "turn": 0, "status": "playing",
                     "eShots": Z36, "pShots": Z36, "eBoard": Z36, "pBoard": Z36,
                     "eHP": HP0, "pHP": HP0}}),
    SET(["state", "tmp"], {"$tpl": {"el": C(["mod", V("s"), 4], s=P("input", "seed"))}}),
    SET(TP("pl"), C(["mod", ["add", V("e"), 2], 4], e=T("el"))),
    place("eBoard", TP("el")),
    place("pBoard", TP("pl")),
    APPEND(MSGS, "New game. Sink the enemy fleet before it sinks yours."),
    APPEND(MSGS, "Fire with: x y   (both 0 to 5)"),
]

# ---------------------------------------------------------------- player fires
IDX = T("idx")
SHIP = T("ship")
SHIPIDX = C(["sub", V("s"), 1], s=SHIP)

FIRE_PLAYER = [
    SET(TP("ship"), C(["get", V("b"), V("i")], b=P(*G, "eBoard"), i=IDX)),
    SET(G + ["eShots", IDX], 1),
    IF(C(["gt", V("s"), 0], s=SHIP), [
        SET(G + ["eHP", SHIPIDX], C(["sub", V("h"), 1], h=P(*G, "eHP", SHIPIDX))),
        APPEND(MSGS, C(
            ["concat", "You fire at (", ["to_string", V("x")], ",", ["to_string", V("y")], "): ",
             ["select", ["eq", V("h"), 0],
              ["concat", "HIT! You sank the ", ["get", V("N"), ["sub", V("s"), 1]], "!"],
              "HIT!"]],
            x=T("x"), y=T("y"), h=P(*G, "eHP", SHIPIDX), s=SHIP, N={"$lit": NAMES})),
    ], [
        APPEND(MSGS, C(["concat", "You fire at (", ["to_string", V("x")], ",", ["to_string", V("y")], "): MISS."],
                       x=T("x"), y=T("y"))),
    ]),
    SET(TP("eDead"), C(["eq", SUM3("h"), 0], h=P(*G, "eHP"))),
]

# ---------------------------------------------------------------- enemy fires
CIDX = T("c")
ESHIP = T("es")
ESIDX = C(["sub", V("s"), 1], s=ESHIP)

ENEMY_TURN = [
    SET(G + ["seed"], C(["mod", ["add", ["mul", V("s"), 1103515245], 12345], 2147483648], s=P(*G, "seed"))),
    SET(TP("start"), C(["mod", V("s"), 36], s=P(*G, "seed"))),
    SET(TP("done"), False),
    FOR(K36, "k", [
        SET(TP("c"), C(["mod", ["add", V("s"), V("k")], 36], s=T("start"), k=P("local", "k"))),
        IF(C(["and", ["not", V("d")], ["eq", V("sh"), 0]], d=T("done"), sh=P(*G, "pShots", CIDX)), [
            SET(TP("done"), True),
            SET(G + ["pShots", CIDX], 1),
            SET(TP("es"), C(["get", V("b"), V("i")], b=P(*G, "pBoard"), i=CIDX)),
            SET(TP("ex"), C(["mod", V("c"), 6], c=CIDX)),
            SET(TP("ey"), C(["div", V("c"), 6, 0, "down"], c=CIDX)),
            IF(C(["gt", V("s"), 0], s=ESHIP), [
                SET(G + ["pHP", ESIDX], C(["sub", V("h"), 1], h=P(*G, "pHP", ESIDX))),
                APPEND(MSGS, C(
                    ["concat", "Enemy fires at (", ["to_string", V("x")], ",", ["to_string", V("y")], "): HIT! Your ",
                     ["get", V("N"), ["sub", V("s"), 1]],
                     ["select", ["eq", V("h"), 0], " was sunk!", " was hit!"]],
                    x=T("ex"), y=T("ey"), s=ESHIP, h=P(*G, "pHP", ESIDX), N={"$lit": NAMES})),
            ], [
                APPEND(MSGS, C(["concat", "Enemy fires at (", ["to_string", V("x")], ",", ["to_string", V("y")], "): MISS."],
                               x=T("ex"), y=T("ey"))),
            ]),
        ]),
    ]),
    SET(TP("pDead"), C(["eq", SUM3("h"), 0], h=P(*G, "pHP"))),
    IF(T("pDead"), [
        SET(G + ["status"], "lost"),
        APPEND(MSGS, "DEFEAT! Your fleet has been sunk."),
    ]),
]

FIRE = [
    SET(G, P("input", "game")),
    ASSERT(C(["eq", V("s"), "playing"], s=P(*G, "status")), "The game is over."),
    SET(["state", "tmp"], {"$tpl": {"x": P("input", "x"), "y": P("input", "y")}}),
    ASSERT(C(["and", ["ge", V("x"), 0], ["le", V("x"), 5], ["ge", V("y"), 0], ["le", V("y"), 5]],
             x=T("x"), y=T("y")), "Off the board: x and y must be between 0 and 5."),
    SET(TP("idx"), C(["add", ["mul", V("y"), 6], V("x")], x=T("x"), y=T("y"))),
    ASSERT(C(["eq", V("s"), 0], s=P(*G, "eShots", IDX)), "You already fired there."),
] + FIRE_PLAYER + [
    IF(T("eDead"), [
        SET(G + ["status"], "won"),
        APPEND(MSGS, "VICTORY! You sank the entire enemy fleet."),
    ], ENEMY_TURN),
    SET(G + ["turn"], C(["add", V("t"), 1], t=P(*G, "turn"))),
]

# ---------------------------------------------------------------- rendering
TI = T("i")
LEFT, RIGHT = TP("left"), TP("right")
def cell(field, playing_reveal):
    shot = ["eq", V("sh"), 0]
    ship = ["gt", V("sp"), 0]
    if playing_reveal:
        unshot = ["select", ["and", ["ne", V("st"), "playing"], ship], "#", "~"]
    else:
        unshot = ["select", ship, "#", "~"]
    return ["select", shot, unshot, ["select", ship, "X", "o"]]

RENDER = [
    APPEND(LINES, "      ENEMY WATERS              YOUR FLEET"),
    APPEND(LINES, "    0 1 2 3 4 5          0 1 2 3 4 5"),
    FOR(ROWS, "r", [
        SET(LEFT, ""),
        SET(RIGHT, ""),
        FOR(ROWS, "c", [
            SET(TP("i"), C(["add", ["mul", V("r"), 6], V("c")], r=P("local", "r"), c=P("local", "c"))),
            SET(LEFT, C(["concat", V("l"), cell("e", True), " "],
                        l=P(*LEFT), sh=P(*G, "eShots", TI), sp=P(*G, "eBoard", TI), st=P(*G, "status"))),
            SET(RIGHT, C(["concat", V("l"), cell("p", False), " "],
                         l=P(*RIGHT), sh=P(*G, "pShots", TI), sp=P(*G, "pBoard", TI))),
        ]),
        APPEND(LINES, C(["concat", ["to_string", V("r")], "   ", V("l"), "        ", ["to_string", V("r")], "   ", V("rt")],
                        r=P("local", "r"), l=P(*LEFT), rt=P(*RIGHT))),
    ]),
    APPEND(LINES, C(["concat", "Ships afloat - enemy: ", ["to_string", AFLOAT("e")],
                     "   you: ", ["to_string", AFLOAT("p")], "   turn: ", ["to_string", V("t")]],
                    e=P(*G, "eHP"), p=P(*G, "pHP"), t=P(*G, "turn"))),
]

PROGRAM = [
    SET(MSGS, []),
    SET(LINES, []),
    IF(C(["eq", V("a"), "new"], a=P("input", "action")), NEW, FIRE),
] + RENDER + [
    SET(["output"], {"$tpl": {"game": P(*G), "view": P(*LINES), "messages": P(*MSGS)}}),
]

# ---------------------------------------------------------------- contracts
def cells(lo, hi):
    return {"type": "array", "items": {"type": "number", "int": True, "min": lo, "max": hi},
            "minItems": 36, "maxItems": 36}
HP = {"type": "array", "items": {"type": "number", "int": True, "min": 0, "max": 3},
      "minItems": 3, "maxItems": 3}
GAME = {"type": "object", "fields": {
    "seed": {"type": "number", "int": True, "min": 0, "max": 2147483647},
    "turn": {"type": "number", "int": True, "min": 0},
    "status": {"type": "string", "enum": ["playing", "won", "lost"]},
    "eShots": cells(0, 1), "pShots": cells(0, 1),
    "eBoard": cells(0, 3), "pBoard": cells(0, 3),
    "eHP": HP, "pHP": HP},
    "required": ["seed", "turn", "status", "eShots", "pShots", "eBoard", "pBoard", "eHP", "pHP"]}
INPUT_SCHEMA = {"type": "object", "fields": {
    "action": {"type": "string", "enum": ["new", "fire"]},
    "seed": {"type": "number", "int": True, "min": 0, "max": 2147483647},
    "x": {"type": "number", "int": True}, "y": {"type": "number", "int": True},
    "game": GAME}, "required": ["action"]}
OUTPUT_SCHEMA = {"type": "object", "fields": {
    "game": GAME,
    "view": {"type": "array", "items": {"type": "string"}},
    "messages": {"type": "array", "items": {"type": "string"}}},
    "required": ["game", "view", "messages"]}

pkg = {"jaxson": "1.0", "limits": {"steps": 20000},
       "input": {"action": "new", "seed": 0},
       "inputSchema": INPUT_SCHEMA, "program": PROGRAM, "outputSchema": OUTPUT_SCHEMA}
with open("navywars.json", "w") as f:
    json.dump(pkg, f, indent=1)
    f.write("\n")
print("navywars.json written")
