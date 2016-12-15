package tilegram

import (
	"errors"
	"math"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
)

// Hexagram is a hexagonal tilegram.
type Hexagram struct {
	hexes []*Hex
	index *rtree.Rtree

	// r is the radius of each hexagon.
	r float64

	b *geom.Bounds
}

// Hex represents an individual hexagonol tile in a tilegram.
type Hex struct {
	// Point is the geometric center of this hexagon.
	geom.Point

	// Data holds the data points that are assigned to this Hex.
	Data []Grouper

	// i is the index of this Hex in its containing Hexagram
	i int

	// r is the radius of the hexagon.
	r float64
}

// Weight returns the sum of the weights of the
// data items in the receiver.
func (h *Hex) Weight() float64 {
	var sum float64
	for _, d := range h.Data {
		sum += d.Weight()
	}
	return sum
}

// Group returns the group which has the most weight among
// the data items in the receiver. If the receiver does not
// have any data items, the group will be "".
func (h *Hex) Group() string {
	groupWeights := make(map[string]float64)
	for _, d := range h.Data {
		groupWeights[d.Group()] += d.Weight()
	}
	var maxGroup string
	var max float64
	for g, w := range groupWeights {
		if w > max {
			maxGroup = g
			max = w
		}
	}
	return maxGroup
}

// Bounds returns the bounds of the hexagon.
func (h *Hex) Bounds() *geom.Bounds {
	return &geom.Bounds{
		Max: geom.Point{
			X: h.Point.X + 3/2*h.r,
			Y: h.Point.Y + h.r/2*math.Sqrt(3),
		},
		Min: geom.Point{
			X: h.Point.X - 3/2*h.r,
			Y: h.Point.Y - h.r/2*math.Sqrt(3),
		},
	}
}

// Geom returns the geometry of the receiver when given the radius of
// the hexagon.
func (h *Hex) Geom() geom.Polygon {
	g := make([][]geom.Point, 1)
	g[0] = make([]geom.Point, 6)
	for i := 0; i < 6; i++ {
		g[0][i] = geom.Point{
			X: h.Point.X + h.r*math.Cos(math.Pi*2/6*float64(i)),
			Y: h.Point.Y + h.r*math.Sin(math.Pi*2/6*float64(i)),
		}
	}
	return g
}

// NewHexagram creates a new hexagonal tile map, where bounds is the
// geometric boundary for the tiles, and r is the radius of each
// hexagonal tile.
func NewHexagram(data []Grouper, r float64) (*Hexagram, error) {
	o := Hexagram{
		index: rtree.NewTree(25, 50),
		r:     r,
		b:     geom.NewBounds(),
	}

	dataIndex := rtree.NewTree(25, 50)
	bbox := geom.NewBounds()
	for _, d := range data {
		bbox.Extend(d.Bounds())
		dataIndex.Insert(d)
	}

	dx := 3 * r
	dy := r * math.Sqrt(3)
	var i int
	var haveHexagons bool // have we added any hexagons
	xstart := []float64{bbox.Min.X, bbox.Min.X - 1.5*r}
	ystart := []float64{bbox.Min.Y, bbox.Min.Y - r}
	for j, xmin := range xstart {
		ymin := ystart[j]
		for x := xmin; x <= bbox.Max.X; x += dx {
			for y := ymin; y <= bbox.Max.Y; y += dy {
				p := geom.Point{X: x, Y: y}
				if len(dataIndex.SearchIntersect(p.Bounds())) == 0 {
					continue
				}
				h := &Hex{
					Point: p,
					i:     i,
					r:     r,
				}
				i++
				o.hexes = append(o.hexes, h)
				o.index.Insert(h)
				o.b.Extend(h.Bounds())
				haveHexagons = true
			}
		}
	}
	if !haveHexagons {
		return nil, errors.New("tilegram: no hexagons of given radius fit within given bounds")
	}
	for _, d := range data {
		o.add(d)
	}
	return &o, nil
}

// add adds a new data item to the tilegram, allocating it to the
// nearest hexagonal tile. It returns the index of the tile that
// the data was added to.
func (h *Hexagram) add(d Grouper) (i int) {
	tile := h.index.NearestNeighbor(d.Centroid()).(*Hex)
	tile.Data = append(tile.Data, d)
	return tile.i
}

// Len returns the number of tiles in the receiver.
func (h *Hexagram) Len() int { return len(h.hexes) }

// Weight returns the sum of the weights of the
// data items in tile i.
func (h *Hexagram) Weight(i int) float64 {
	return h.hexes[i].Weight()
}

// Hexes returns the Hex tiles that comprise the receiver.
func (h *Hexagram) Hexes() []*Hex {
	return h.hexes
}

// Bounds returns the bounding box of the receiver.
func (h *Hexagram) Bounds() *geom.Bounds {
	return h.b
}

// GroupGeom returns the combined geomtry of the hexagons
// in each group, where tolerance is the distance two points
// can be apart while still being considered as in the same location.
// tolerance can be used to avoid polygon slivers in the result.
func (h *Hexagram) GroupGeom(tolerance float64) map[string]geom.Polygon {
	polys := make(map[string][]geom.Polygonal)
	for _, hh := range h.hexes {
		g := hh.Group()
		polys[g] = append(polys[g], hh.Geom())
	}
	o := make(map[string]geom.Polygon)
	for g, p := range polys {
		o[g] = newHull(tolerance, p...)
	}
	return o
}

func (h *Hexagram) neighbors(hh *Hex) []*Hex {
	b := hh.Bounds()
	b.Max.X += hh.r / 2
	b.Max.Y += hh.r / 2
	b.Min.X -= hh.r / 2
	b.Min.Y -= hh.r / 2
	temp := h.index.SearchIntersect(b)
	o := make([]*Hex, len(temp))
	for i, t := range temp {
		o[i] = t.(*Hex)
	}
	return o
}

// rangeRatio returns the range in total weights among Hexagrams
// divided by their mean.
func (h *Hexagram) rangeRatio() (ratio, mean float64) {
	var max = math.Inf(-1)
	var min = math.Inf(1)
	for _, hh := range h.hexes {
		w := hh.Weight()
		max = math.Max(w, max)
		min = math.Min(w, min)
		mean += w
	}
	mean /= float64(h.Len())
	return (max - min) / mean, mean
}
