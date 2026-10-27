// Copyright (c) 2026 haitch <h@ual.li>
// SPDX-License-Identifier: Apache-2.0

package main

// Data model of an accessgraph database. The JSON shape is the one produced
// by synth.py and read by oracle.py, so databases can be exchanged freely.

type Grant struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Role    string `json:"role"`
	Scope   string `json:"scope"`
	Effect  string `json:"effect"`
}

type Member struct {
	Member string `json:"member"`
	Group  string `json:"group"`
}

type Role struct {
	ID       string   `json:"id"`
	Perms    []string `json:"perms"`
	Inherits []string `json:"inherits"`
}

type Resource struct {
	ID     string  `json:"id"`
	Parent *string `json:"parent"`
}

type Deleg struct {
	ID    string `json:"id"`
	From  string `json:"from"`
	To    string `json:"to"`
	Role  string `json:"role"`
	Scope string `json:"scope"`
}

type Request struct {
	ID       string `json:"id"`
	User     string `json:"user"`
	Action   string `json:"action"`
	Resource string `json:"resource"`
}

type Meta struct {
	Seed    uint64 `json:"seed"`
	Size    string `json:"size"`
	Cycles  bool   `json:"cycles"`
	MaxHops int    `json:"max_hops"`
}

type DB struct {
	Meta        Meta       `json:"meta"`
	Users       []string   `json:"users"`
	Groups      []string   `json:"groups"`
	Roles       []Role     `json:"roles"`
	Resources   []Resource `json:"resources"`
	Members     []Member   `json:"members"`
	Grants      []Grant    `json:"grants"`
	Delegations []Deleg    `json:"delegations"`
	Requests    []Request  `json:"requests"`
}

// Result is the decision for one request.
type Result struct {
	Allow    bool     `json:"allow"`
	Via      *string  `json:"via"`
	DeniedBy []string `json:"denied_by"`
}
