package tilegram

import (
	"image/color"
	"log"
	"os"
	"testing"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/carto"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/tilegram/internal/cmpimg"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"github.com/gonum/plot/vg/vgimg"
)

// This example creates a hexagonal tilegram from census block-group
// level population data for the US state of Washington.
func Example() {
	// Open the shapefile with the census data.
	d, err := shp.NewDecoder("testdata/WA_Population_2010.shp")
	if err != nil {
		log.Panic(err)
	}
	defer d.Close()

	type censusData struct {
		geom.Polygonal         // This is the geometry of the census block group.
		Population     float64 // This is the population in the block group.
		County         string  // This is the name of the county it is in.
	}
	var records []Grouper
	var blckgrps []geom.Polygonal // This will hold the census block group geometry.
	var population []float64
	// Read in the census data.
	for {
		var rec censusData
		if more := d.DecodeRow(&rec); !more {
			break
		}
		records = append(records, NewData(rec, rec.Population, rec.County))
		population = append(population, rec.Population)
		blckgrps = append(blckgrps, rec.Polygonal)
	}
	if err := d.Error(); err != nil {
		log.Panic(err)
	}

	// Create two new Hexagrams with our census data.
	const (
		r          = 20000.0 // This is the hexagon radius in meters.
		rangeRatio = 1.3     // This is the acceptable level of variability in population.
	)
	h1, err := NewHexagram(records, r)
	if err != nil {
		log.Panic(err)
	}

	h2, err := NewHexagram(records, r)
	if err != nil {
		log.Panic(err)
	}
	h2.Distribute(rangeRatio) // We only allocate the population in one tilegram.

	// Make plots of the results.
	// Everything below here is unrelated to the creation
	// of the tilegrams themselves.

	const (
		// These are the figure dimensions.
		width        = 20 * vg.Centimeter
		height       = 6 * vg.Centimeter
		legendHeight = 0.9 * vg.Centimeter
	)

	// Create a canvas to put our plot on.
	img := vgimg.NewWith(vgimg.UseWH(width, height), vgimg.UseDPI(300))
	dc := draw.New(img)
	legendc := draw.Crop(dc, 0, 0, 0, legendHeight-dc.Max.Y+dc.Min.Y)
	dc = draw.Crop(dc, 0, 0, legendHeight, 0)
	tiles := draw.Tiles{
		Cols: 3,
		Rows: 1,
	}
	lineStyle := draw.LineStyle{Width: 0.1 * vg.Millimeter}

	// First, make a map of our unaltered census data.
	cmap := carto.NewColorMap(carto.Linear)
	cmap.AddArray(population)
	panelA := tiles.At(dc, 0, 0)
	cmap.Set()
	lc := tiles.At(legendc, 0, 0)
	cmap.Legend(&lc, "Population")
	b := h1.Bounds()
	m := carto.NewCanvas(b.Max.Y, b.Min.Y, b.Max.X, b.Min.X, panelA)
	for i, rec := range records {
		color := cmap.GetColor(rec.Weight())
		lineStyle.Color = color
		m.DrawVector(blckgrps[i], color, lineStyle, draw.GlyphStyle{})
	}
	// Now, draw the county boundaries.
	d2, err := shp.NewDecoder("testdata/WA_Counties_2010.shp")
	if err != nil {
		log.Panic(err)
	}
	defer d2.Close()

	type countyData struct {
		geom.Polygonal
	}
	for {
		var rec countyData
		if more := d2.DecodeRow(&rec); !more {
			break
		}
		m.DrawVector(rec.Polygonal, color.NRGBA{}, draw.LineStyle{
			Width: 0.25 * vg.Millimeter,
			Color: color.Black,
		}, draw.GlyphStyle{})
	}
	if err := d2.Error(); err != nil {
		log.Panic(err)
	}

	// Next, make a maps of our tilegrams.
	for i, h := range []*Hexagram{h1, h2} {
		weights := make([]float64, h.Len())
		for i := 0; i < h.Len(); i++ {
			weights[i] = h.Weight(i)
		}
		cmap = carto.NewColorMap(carto.Linear)
		cmap.AddArray(weights)
		panelB := tiles.At(dc, i+1, 0)
		cmap.Set()
		lc = tiles.At(legendc, i+1, 0)
		cmap.Legend(&lc, "Population")
		m = carto.NewCanvas(b.Max.Y, b.Min.Y, b.Max.X, b.Min.X, panelB)
		for _, hex := range h.Hexes() {
			c := cmap.GetColor(hex.Weight())
			lineStyle.Color = c
			m.DrawVector(hex.Geom(), c, lineStyle, draw.GlyphStyle{})
		}
		// Now we add our county boundaries.
		for _, g := range h.GroupGeom(r / 2) {
			m.DrawVector(g, color.NRGBA{}, draw.LineStyle{
				Width: 0.25 * vg.Millimeter,
				Color: color.Black,
			}, draw.GlyphStyle{})
		}
	}

	// Save the file
	f, err := os.Create("testdata/hex.png")
	if err != nil {
		panic(err)
	}
	pngc := vgimg.PngCanvas{Canvas: img}
	if _, err := pngc.WriteTo(f); err != nil {
		panic(err)
	}
}

func TestExample(t *testing.T) {
	cmpimg.CheckPlot(Example, t, "hex.png")
}
