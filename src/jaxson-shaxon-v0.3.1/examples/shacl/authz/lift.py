#!/usr/bin/env python3
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the Apache License, Version 2.0.
# https://www.apache.org/licenses/LICENSE-2.0
"""Lift a Shaxon example's JSON input into an RDF graph for SHACL.

SHACL validates an RDF graph, so the JSON trail has to be lifted first. The
rules, which are the same for every example:

* an object becomes a blank node, a member becomes a triple whose predicate is
  ex:<member name>; the root object is the node every other node hangs from;
* an array becomes one triple per element, so document order is NOT kept (an
  RDF graph has no order; the examples that depend on order carry a number
  to order by, as the four-eyes events do with `seq`);
* strings are plain literals, booleans xsd:boolean, null is dropped;
* a number keeps its exact decimal text: an integer is xsd:integer and
  anything with a fraction or exponent is xsd:decimal. (JSON-LD would turn
  such a number into xsd:double, which cannot hold 0.1 exactly, so a plain
  JSON-LD lift is not used.)

An object whose member NAMES are data (chinese-wall's `classOf`) has no
natural RDF form; EXAMPLE_FIXUPS rewrites it into a list of records first.
"""
import json
from decimal import Decimal
from rdflib import BNode, Graph, Literal, Namespace
from rdflib.namespace import XSD

EX = Namespace("https://example.org/authz#")


def _load(text):
    return json.loads(text, parse_float=Decimal, parse_int=int)


def _literal(v):
    if isinstance(v, bool):
        return Literal(v, datatype=XSD.boolean)
    if isinstance(v, int):
        return Literal(str(v), datatype=XSD.integer)
    if isinstance(v, Decimal):
        # 1E+2 and the like are normalised to plain decimal text
        return Literal(format(v, "f"), datatype=XSD.decimal)
    return Literal(v)


def _add(g, subject, key, value):
    if value is None:
        return
    if isinstance(value, list):
        for item in value:
            _add(g, subject, key, item)
    elif isinstance(value, dict):
        node = BNode()
        g.add((subject, EX[key], node))
        for k, v in value.items():
            _add(g, node, k, v)
    else:
        g.add((subject, EX[key], _literal(value)))


def _fix_chinese_wall(inp):
    inp = dict(inp)
    inp["companies"] = [{"id": c, "class": k} for c, k in inp.pop("classOf").items()]
    return inp


EXAMPLE_FIXUPS = {"chinese-wall": _fix_chinese_wall}


def lift(name, inp):
    """inp is the decoded input (dict, numbers as int/Decimal)."""
    inp = EXAMPLE_FIXUPS.get(name, lambda x: x)(inp)
    g = Graph()
    g.bind("ex", EX)
    root = BNode()
    for k, v in inp.items():
        _add(g, root, k, v)
    return g
