// Copyright ©2016 The tilegram Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tilegram

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/ctessum/geom"
)

func TestHull(t *testing.T) {
	d := []geom.Polygonal{
		geom.Polygon{{{0, 0}, {1, 0}, {1, 1}, {0, 1}}},
		geom.Polygon{{{1, 0}, {2, 0}, {2, 1}, {1, 1.01}}},
		geom.Polygon{{{1, 1}, {2, 1}, {2, 2}, {1, 2}}},
		geom.Polygon{{{0, 1}, {1, 1}, {1, 2}, {0, 2}}},
	}
	want := normalize(geom.Polygon{{{0, 0}, {2, 0}, {2, 2}, {0, 2}, {0, 1}}})
	have := normalize(newHull(0.1, d...))
	have = have.Simplify(0.0001).(geom.Polygon)
	if !reflect.DeepEqual(normalize(want), normalize(have)) {
		t.Errorf("want: %v, have %v", dump(want), dump(have))
	}
}

type sorter geom.Polygon

func (s sorter) Len() int      { return len(s) }
func (s sorter) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sorter) Less(i, j int) bool {
	if len(s[i]) != len(s[j]) {
		return len(s[i]) < len(s[j])
	}
	for k := range s[i] {
		pi, pj := s[i][k], s[j][k]
		if pi.X != pj.X {
			return pi.X < pj.X
		}
		if pi.Y != pj.Y {
			return pi.Y < pj.Y
		}
	}
	return false
}

// basic normalization just for tests; to be improved if needed
func normalize(poly geom.Polygon) geom.Polygon {
	for i, c := range poly {
		if len(c) == 0 {
			continue
		}

		// find bottom-most of leftmost points, to have fixed anchor
		min := 0
		for j, p := range c {
			if p.X < c[min].X || p.X == c[min].X && p.Y < c[min].Y {
				min = j
			}
		}

		// rotate points to make sure min is first
		poly[i] = append(c[min:], c[:min]...)
	}

	sort.Sort(sorter(poly))
	return poly
}

func dump(poly geom.Polygon) string {
	return fmt.Sprintf("%v", normalize(poly))
}
