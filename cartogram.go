package tilegram

// #cgo LDFLAGS: -lfftw3 -lm
// #cgo CFLAGS: -O7
// #include <cart.h>
import "C"
import (
	"sync"
	"unsafe"

	"gonum.org/v1/gonum/mat"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
)

var lock sync.Mutex

// Cartogram holds information for cartogram creation.
type Cartogram struct {
	density    **C.double
	dens       *mat.Dense
	rows, cols int
	index      *rtree.Rtree
	b          *geom.Bounds
	dx, dy     float64

	// Blur is the radius (in pixels) for Gaussian blurring.
	Blur float64
}

func (c *Cartogram) Dims() (cols, rows int) { return c.cols, c.rows }
func (c *Cartogram) Z(col, row int) float64 { return c.dens.At(row, col) }
func (c *Cartogram) X(col int) float64      { return c.b.Min.X + float64(col)*c.dx }
func (c *Cartogram) Y(row int) float64      { return c.b.Min.Y + float64(row)*c.dy }

// PolygonDensity defines the inputs for cartogram creation.
type PolygonDensity interface {
	Len() int
	Polygon(int) geom.Polygonal
	Density(int) float64
}

type gridCell struct {
	geom.Polygon
	i, j int
}

// NewCartogram creates a cartogram with the given margin
// added to each border of the matrix and the given numbers of rows and columns.
//
// It applies the cartogram creation algorithm described in the
// article below:
//
// Gastner, M. T., & Newman, M. E. J. (2004). Diffusion-based method for
// producing density-equalizing maps. Proc. Nat. Acad. of Sci., 101(20),
// 7499–7504. http://doi.org/10.1073/pnas.0400280101
func NewCartogram(shapes PolygonDensity, margin float64, rows, cols int) *Cartogram {
	b := geom.NewBounds()
	var avgDens float64
	var totalArea float64
	for i := 0; i < shapes.Len(); i++ {
		p := shapes.Polygon(i)
		ap := p.Area()
		b.Extend(p.Bounds())
		avgDens += shapes.Density(i) * ap
		totalArea += ap
	}
	avgDens /= totalArea

	b.Min.X -= margin
	b.Min.Y -= margin
	b.Max.X += margin
	b.Max.Y += margin

	o := Cartogram{
		b:     b,
		dx:    (b.Max.X - b.Min.X) / float64(cols),
		dy:    (b.Max.Y - b.Min.Y) / float64(rows),
		index: rtree.NewTree(25, 50),
		rows:  rows,
		cols:  cols,
	}

	for j := 0; j < rows; j++ {
		for i := 0; i < cols; i++ {
			x := float64(i)*o.dx + o.b.Min.X
			y := float64(j)*o.dy + o.b.Min.Y
			gc := gridCell{
				Polygon: geom.Polygon{{
					geom.Point{X: x, Y: y},
					geom.Point{X: x + o.dx, Y: y},
					geom.Point{X: x + o.dx, Y: y + o.dy},
					geom.Point{X: x, Y: y + o.dy},
					geom.Point{X: x, Y: y},
				}},
				i: i,
				j: j,
			}
			o.index.Insert(&gc)
		}
	}

	// Allocate C memory
	lock.Lock()
	C.cart_makews(C.int(cols), C.int(rows))
	o.density = C.cart_dmalloc(C.int(cols), C.int(rows))

	m := mat.NewDense(rows, cols, nil)
	for j := 0; j < rows; j++ {
		for i := 0; i < cols; i++ {
			m.Set(j, i, avgDens) // Set average density.
		}
	}

	for i := 0; i < shapes.Len(); i++ {
		p := shapes.Polygon(i)
		v := shapes.Density(i)
		for _, cI := range o.index.SearchIntersect(p.Bounds()) {
			c := cI.(*gridCell)
			a := c.Intersection(p).Area()
			if a <= 0 {
				continue
			}
			ac := c.Area()
			m.Set(c.j, c.i, m.At(c.j, c.i)+(v-avgDens)*a/ac)
		}
	}
	o.dens = m

	for j := 0; j < rows; j++ {
		for i := 0; i < cols; i++ {
			C.cart_setrho(o.density, C.int(i), C.int(j), C.double(m.At(j, i)))
		}
	}
	C.cart_transform(o.density, C.int(o.cols), C.int(o.rows))
	C.cart_dfree(o.density)
	return &o
}

// TransformPoint moves a point to match a cartogram.
func (c *Cartogram) TransformPoint(p geom.Point) geom.Point {
	x, y := c.pointToGrid(p)
	C.cart_makecartnooptions((*C.double)(unsafe.Pointer(&x[0])), (*C.double)(unsafe.Pointer(&y[0])), C.int(1), C.int(c.cols), C.int(c.rows), C.double(c.Blur))
	return c.pointFromGrid(x, y)
}

func (c *Cartogram) TransformPath(p geom.Path) geom.Path {
	x, y := c.pathToGrid(p)
	C.cart_makecartnooptions((*C.double)(unsafe.Pointer(&x[0])), (*C.double)(unsafe.Pointer(&y[0])), C.int(len(x)), C.int(c.cols), C.int(c.rows), C.double(c.Blur))
	return c.pathFromGrid(x, y)
}

func (c *Cartogram) TransformPolygons(p []geom.Polygon) []geom.Polygon {
	o := make([]geom.Polygon, len(p))
	var path geom.Path
	cuts := make([][][2]int, len(p))
	var k int
	for i, poly := range p {
		cuts[i] = make([][2]int, len(poly))
		for j, ring := range poly {
			path = append(path, ring...)
			cuts[i][j] = [2]int{k, k + len(ring)}
			k += len(ring)
		}
	}
	path = c.TransformPath(path)
	for i, cut := range cuts {
		o[i] = make(geom.Polygon, len(cut))
		for j, c := range cut {
			o[i][j] = path[c[0]:c[1]]
		}
	}
	return o
}

func (c *Cartogram) pointToGrid(p geom.Point) (x, y []float64) {
	x = []float64{(p.X - c.b.Min.X) / c.dx}
	y = []float64{(p.Y - c.b.Min.Y) / c.dy}
	return x, y
}

func (c *Cartogram) pathToGrid(p geom.Path) (x, y []float64) {
	x = make([]float64, len(p))
	y = make([]float64, len(p))
	for i, pp := range p {
		x[i] = (pp.X - c.b.Min.X) / c.dx
		y[i] = (pp.Y - c.b.Min.Y) / c.dy
	}
	return x, y
}

func (c *Cartogram) pointFromGrid(x, y []float64) geom.Point {
	return geom.Point{X: c.b.Min.X + c.dx*x[0], Y: c.b.Min.Y + c.dy*y[0]}
}

func (c *Cartogram) pathFromGrid(x, y []float64) geom.Path {
	p := make(geom.Path, len(x))
	for i, xx := range x {
		yy := y[i]
		p[i] = geom.Point{X: c.b.Min.X + c.dx*xx, Y: c.b.Min.Y + c.dy*yy}
	}
	return p
}

// Destroy frees the memory associated with the receiver. No other cartogram
// can be created before this happens.
func (c *Cartogram) Destroy() {
	C.cart_freews(C.int(c.cols), C.int(c.rows))
	lock.Unlock()
}
