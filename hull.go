// Copyright ©2016 The tilegram Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tilegram

import (
	"fmt"
	"math"

	"github.com/ctessum/geom"
)

type empty struct{}

// hull represents the hull of a set of polygons
type hull struct {
	// graph holds the points of a polygon(s) in a graph.
	// The index of the first map is the starting point of each segment
	// in the polygon and the index of the second map is the ending point
	// of each segment.
	graph map[geom.Point]map[geom.Point]empty

	tolerance float64
}

// newHull creates a new hull from polygons, where tolerance
// specifies the maximum distance between two points where they are
// assumed to be equivalent.
func newHull(tolerance float64, p ...geom.Polygonal) geom.Polygon {
	h := hull{
		graph:     make(map[geom.Point]map[geom.Point]empty),
		tolerance: tolerance,
	}
	for _, polys := range p {
		for _, poly := range polys.Polygons() {
			for _, r := range poly {
				for i := 0; i < len(r)-1; i++ {
					h.addToGraph(segment{start: r[i], end: r[i+1]})
				}
				if r[0] != r[len(r)-1] {
					// close the ring
					h.addToGraph(segment{start: r[len(r)-1], end: r[0]})
				}
			}
		}
	}
	return h.Polygon()
}

// addToGraph adds the segments of the polygon to the graph in a
// way that ensures the same segment is not included twice in the
// polygon.
func (h *hull) addToGraph(seg segment) {
	if seg.start.Equals(seg.end) {
		// The starting and ending points are the same, so this is
		// not in fact a segment.
		return
	}

	// Replace the points with any existing points within tolerance.
	for p1, x := range h.graph {
		if seg.start != p1 && seg.end != p1 {
			if math.Hypot(p1.X-seg.start.X, p1.Y-seg.start.Y) < h.tolerance {
				seg.start = p1
			}
			if math.Hypot(p1.X-seg.end.X, p1.Y-seg.end.Y) < h.tolerance {
				seg.end = p1
			}
		}
		for p2 := range x {
			if seg.start == p2 || seg.end == p2 {
				break
			}
			if math.Hypot(p2.X-seg.start.X, p2.Y-seg.start.Y) < h.tolerance {
				seg.start = p2
			}
			if math.Hypot(p2.X-seg.end.X, p2.Y-seg.end.Y) < h.tolerance {
				seg.end = p2
			}
		}
	}

	if _, ok := h.graph[seg.end][seg.start]; ok {
		// This polygonGraph already has a segment end -> start, adding
		// start -> end would make the polygon degenerate, so we delete both.
		delete(h.graph[seg.end], seg.start)
		if len(h.graph[seg.end]) == 0 {
			delete(h.graph, seg.end)
		}
		return
	}
	if _, ok := h.graph[seg.start][seg.end]; ok {
		// This polygonGraph already has this segment, so adding
		// start -> end would make the polygon degenerate, so we delete both.
		delete(h.graph[seg.start], seg.end)
		if len(h.graph[seg.start]) == 0 {
			delete(h.graph, seg.start)
		}
		return
	}

	if _, ok := h.graph[seg.start]; !ok {
		h.graph[seg.start] = make(map[geom.Point]empty)
	}

	// Add the segment.
	h.graph[seg.start][seg.end] = empty{}
}

// Used to represent an edge of a polygon.
type segment struct {
	start, end geom.Point
}

func (h *hull) Polygon() geom.Polygon {
	var p geom.Polygon
	for len(h.graph) > 0 {
		p = append(p, h.ring())
	}
	return p
}

func (h *hull) ring() []geom.Point {
	var p geom.Point
	for p = range h.graph { // get first point
		break
	}
	r := []geom.Point{p}
	for {
		if len(h.graph[p]) != 1 {
			panic("problem here")
		}
		for pp := range h.graph[p] {
			r = append(r, pp)
			delete(h.graph[p], pp)
			if len(h.graph[p]) == 0 {
				delete(h.graph, p)
			}
			p = pp
			break
		}
		if r[0] == r[len(r)-1] {
			break
		}
	}
	return r
}

func (h *hull) String() string {
	s := "*hull{\n"
	for p1, d := range h.graph {
		for p2 := range d {
			s += fmt.Sprintf("\t%v -> %v\n", p1, p2)
		}
	}
	return s + "}\n"
}
