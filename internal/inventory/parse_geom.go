// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type node struct {
	name, id, ref, text string
	children            []*node
}

// xmlTree bounds structure before creating nodes; directives/entities and mixed
// content are unnecessary for GEOM. Unknown bounded fields remain forward-compatible.
func xmlTree(data []byte, limits Limits) (*node, error) {
	if err := checkInput(data, limits); err != nil {
		return nil, err
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var root *node
	var stack []*node
	elements := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, ErrMalformed
		}
		switch v := token.(type) {
		case xml.StartElement:
			elements++
			if elements > limits.MaxXMLElements || len(stack) >= limits.MaxXMLDepth || len(v.Attr) > 32 {
				return nil, ErrLimit
			}
			if v.Name.Space != "" {
				return nil, ErrMalformed
			}
			if len(v.Name.Local) > limits.MaxFieldBytes {
				return nil, ErrLimit
			}
			n := &node{name: v.Name.Local}
			attrs := make(map[string]bool)
			for _, attr := range v.Attr {
				if attr.Name.Space != "" || attrs[attr.Name.Local] {
					return nil, ErrMalformed
				}
				attrs[attr.Name.Local] = true
				if err := safeText(attr.Value, limits); err != nil {
					return nil, err
				}
				switch attr.Name.Local {
				case "id":
					n.id = attr.Value
				case "ref":
					n.ref = attr.Value
				}
			}
			if len(stack) == 0 {
				if root != nil || n.name != "mesh" {
					return nil, ErrMalformed
				}
				root = n
			} else {
				parent := stack[len(stack)-1]
				if strings.TrimSpace(parent.text) != "" {
					return nil, ErrMalformed
				}
				parent.text = ""
				parent.children = append(parent.children, n)
			}
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, ErrMalformed
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 {
				if strings.TrimSpace(string(v)) != "" {
					return nil, ErrMalformed
				}
				continue
			}
			n := stack[len(stack)-1]
			if len(n.children) != 0 {
				if strings.TrimSpace(string(v)) != "" {
					return nil, ErrMalformed
				}
				continue
			}
			if len(v) > limits.MaxFieldBytes-len(n.text) {
				return nil, ErrLimit
			}
			n.text += string(v)
		case xml.Directive:
			return nil, ErrMalformed
		case xml.ProcInst:
			if v.Target != "xml" || root != nil {
				return nil, ErrMalformed
			}
		}
	}
	if root == nil || len(stack) != 0 {
		return nil, ErrMalformed
	}
	return root, nil
}

func (n *node) childrenNamed(name string) []*node {
	var found []*node
	for _, child := range n.children {
		if child.name == name {
			found = append(found, child)
		}
	}
	return found
}

func (n *node) one(name string, required bool) (*node, error) {
	items := n.childrenNamed(name)
	if len(items) > 1 || required && len(items) != 1 {
		return nil, ErrMalformed
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (n *node) scalar(name string, limits Limits) (string, error) {
	child, err := n.one(name, true)
	if err != nil || len(child.children) != 0 {
		return "", ErrMalformed
	}
	value := strings.TrimSpace(child.text)
	if err := safeText(value, limits); err != nil {
		return "", err
	}
	return value, nil
}

func parentRef(n *node, name, expected string) error {
	child, err := n.one(name, false)
	if err != nil {
		return err
	}
	if child != nil && child.ref != expected {
		return ErrMalformed
	}
	return nil
}

func identifiers(n *node, source string, limits Limits) ([]Identifier, error) {
	config, err := n.one("config", false)
	if err != nil || config == nil {
		return nil, err
	}
	var result []Identifier
	for _, key := range []string{"ident", "lunid"} {
		child, err := config.one(key, false)
		if err != nil {
			return nil, err
		}
		if child == nil {
			continue
		}
		if len(child.children) != 0 {
			return nil, ErrMalformed
		}
		value := strings.TrimSpace(child.text)
		if value == "" {
			continue
		}
		if err := safeText(value, limits); err != nil {
			return nil, err
		}
		kind := "serial"
		if key == "lunid" {
			kind = "lunid"
		}
		result = append(result, Identifier{Kind: kind, Value: value, Source: source + ".config." + key})
	}
	return result, nil
}

func ParseGEOM(data []byte, limits Limits) (result GEOM, parseError error) {
	stage := "xml"
	defer func() {
		if parseError != nil {
			parseError = fmt.Errorf("%w: GEOM %s", parseError, stage)
		}
	}()
	root, err := xmlTree(data, limits)
	if err != nil {
		return GEOM{}, err
	}
	graph := GEOM{Classes: []Class{}, Geoms: []Geom{}, Providers: []Provider{}}
	ids := make(map[string]bool)
	register := func(id string) error {
		if len(ids) >= limits.MaxRecords {
			return ErrLimit
		}
		if err := safeText(id, limits); err != nil {
			return err
		}
		if ids[id] {
			return ErrMalformed
		}
		ids[id] = true
		return nil
	}
	for _, c := range root.childrenNamed("class") {
		stage = "class-id"
		if err := register(c.id); err != nil {
			return GEOM{}, err
		}
		stage = "class-name"
		class, err := c.scalar("name", limits)
		if err != nil {
			return GEOM{}, err
		}
		graph.Classes = append(graph.Classes, Class{ID: c.id, Name: class})
		for _, g := range c.childrenNamed("geom") {
			stage = "geom-id"
			if err := register(g.id); err != nil {
				return GEOM{}, err
			}
			stage = "geom-class"
			if err := parentRef(g, "class", c.id); err != nil {
				return GEOM{}, err
			}
			stage = "geom-name"
			name, err := g.scalar("name", limits)
			if err != nil {
				return GEOM{}, err
			}
			geom := Geom{ID: g.id, ClassID: c.id, Name: name, Consumers: []Consumer{}}
			for _, consumer := range g.childrenNamed("consumer") {
				stage = "consumer-id"
				if err := register(consumer.id); err != nil {
					return GEOM{}, err
				}
				stage = "consumer-geom"
				if err := parentRef(consumer, "geom", g.id); err != nil {
					return GEOM{}, err
				}
				stage = "consumer-provider"
				parent, err := consumer.one("provider", false)
				if err != nil {
					return GEOM{}, err
				}
				ref := ""
				if parent != nil {
					ref = parent.ref
				}
				geom.Consumers = append(geom.Consumers, Consumer{ID: consumer.id, ProviderID: ref})
			}
			graph.Geoms = append(graph.Geoms, geom)
			stage = "geom-identifiers"
			geomIDs, err := identifiers(g, "geom", limits)
			if err != nil {
				return GEOM{}, err
			}
			for _, p := range g.childrenNamed("provider") {
				stage = "provider-id"
				if err := register(p.id); err != nil {
					return GEOM{}, err
				}
				stage = "provider-geom"
				if err := parentRef(p, "geom", g.id); err != nil {
					return GEOM{}, err
				}
				stage = "provider-name"
				name, err := p.scalar("name", limits)
				if err != nil {
					return GEOM{}, err
				}
				stage = "provider-mediasize"
				media, err := p.scalar("mediasize", limits)
				if err != nil {
					return GEOM{}, err
				}
				stage = "provider-sectorsize"
				sector, err := p.scalar("sectorsize", limits)
				if err != nil {
					return GEOM{}, err
				}
				stage = "provider-mediasize-value"
				mediaSize, err := number(media)
				if err != nil {
					return GEOM{}, err
				}
				stage = "provider-sectorsize-value"
				sectorSize, err := number(sector)
				if err != nil || sectorSize == 0 {
					return GEOM{}, ErrMalformed
				}
				stage = "provider-identifiers"
				providerIDs, err := identifiers(p, "provider", limits)
				if err != nil {
					return GEOM{}, err
				}
				allIDs := append(append([]Identifier{}, geomIDs...), providerIDs...)
				kind := Logical
				switch class {
				case "DISK":
					kind = DiskCandidate
				case "PART":
					kind = Partition
				case "MD":
					kind = Memory
				}
				graph.Providers = append(graph.Providers, Provider{ID: p.id, GeomID: g.id, Name: name, Class: class, Kind: kind, MediaSize: mediaSize, SectorSize: sectorSize, Identifiers: allIDs})
			}
		}
	}
	providers := make(map[string]bool)
	for _, p := range graph.Providers {
		providers[p.ID] = true
	}
	for _, g := range graph.Geoms {
		for _, c := range g.Consumers {
			stage = "consumer-target"
			if c.ProviderID != "" && !providers[c.ProviderID] {
				return GEOM{}, ErrMalformed
			}
		}
	}
	markIdentity(graph.Providers)
	return graph, nil
}

func markIdentity(providers []Provider) {
	owners := make(map[string]map[int]bool)
	for i, p := range providers {
		if p.Kind != DiskCandidate {
			continue
		}
		for _, id := range p.Identifiers {
			key := id.Kind + "\x00" + id.Value
			if owners[key] == nil {
				owners[key] = make(map[int]bool)
			}
			owners[key][i] = true
		}
	}
	for i := range providers {
		p := &providers[i]
		p.Identity = IdentityNotApplicable
		if p.Kind != DiskCandidate {
			continue
		}
		p.Identity = IdentityMissing
		if len(p.Identifiers) == 0 {
			continue
		}
		p.Identity = IdentityUnique
		values := make(map[string]string)
		for _, id := range p.Identifiers {
			if len(owners[id.Kind+"\x00"+id.Value]) > 1 || values[id.Kind] != "" && values[id.Kind] != id.Value {
				p.Identity = IdentityAmbiguous
			}
			values[id.Kind] = id.Value
		}
	}
}
