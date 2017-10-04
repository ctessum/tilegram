// Copyright ©2016 The tilegram Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

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
	"github.com/gonum/floats"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

type censusData struct {
	geom.Polygon         // This is the geometry of the census block group.
	Population   float64 // This is the population in the block group.
	County       string  // This is the name of the county it is in.
}

type censusDensity []censusData

func (c censusDensity) Len() int                     { return len(c) }
func (c censusDensity) Polygon(i int) geom.Polygonal { return c[i].Polygon }
func (c censusDensity) Density(i int) float64        { return c[i].Population / c[i].Polygon.Area() }

// This example creates a hexagonal tilegram from census block-group
// level population data for the US state of Washington.
// The output from this example can be seen at:
// https://github.com/ctessum/tilegram/blob/master/testdata/hex_golden.png
func Example() {
	// Open the shapefile with the census data.
	d, err := shp.NewDecoder("testdata/WA_Population_2010.shp")
	if err != nil {
		log.Panic(err)
	}
	defer d.Close()

	var data censusDensity
	var blckgrps []geom.Polygon // This will hold the census block group geometry.
	var population []float64
	// Read in the census data.
	for {
		var rec censusData
		if more := d.DecodeRow(&rec); !more {
			break
		}
		data = append(data, rec)
		population = append(population, rec.Population)
		blckgrps = append(blckgrps, rec.Polygon)
	}
	if err = d.Error(); err != nil {
		log.Panic(err)
	}

	const (
		margin = 500000.0 // meters
		rows   = 512      //256
		cols   = 1024     //512
	)
	cartogram := NewCartogram(data, margin, rows, cols)
	cartogram.Blur = 3
	blckgrps2 := cartogram.TransformPolygons(blckgrps)

	h := plotter.NewHeatMap(cartogram, palette.Heat(12, 1))
	plt, err := plot.New()
	if err != nil {
		log.Panic(err)
	}
	plt.Add(h)
	if err = plt.Save(3*vg.Inch, 2*vg.Inch, "density.png"); err != nil {
		log.Panic(err)
	}

	var records []Grouper
	for i, b := range blckgrps2 {
		records = append(records, &Data{
			Polygonal: b,
			W:         data[i].Population,
			G:         data[i].County,
		})
	}

	const r = 20000.0 // This is the hexagon radius in meters.
	hex, err := NewHexagram(records, r)
	if err != nil {
		log.Panic(err)
	}

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
	panelB := tiles.At(dc, 1, 0)
	cmap.Set()
	lc := tiles.At(legendc, 0, 0)
	cmap.Legend(&lc, "Population")
	b := hex.Bounds()
	m := carto.NewCanvas(b.Max.Y, b.Min.Y, b.Max.X, b.Min.X, panelA)
	for i, rec := range records {
		color := cmap.GetColor(rec.Weight())
		lineStyle.Color = color
		m.DrawVector(blckgrps[i], color, lineStyle, draw.GlyphStyle{})
	}

	cartoPopDensity := make([]float64, len(blckgrps2))
	for i, g := range blckgrps2 {
		cartoPopDensity[i] = population[i] / g.Area() * 1e6
	}
	avg := stat.Mean(cartoPopDensity, nil)
	floats.AddConst(-avg, cartoPopDensity)
	floats.Scale(1/avg, cartoPopDensity)
	cmap2 := carto.NewColorMap(carto.LinCutoff)
	cmap2.AddArray(cartoPopDensity)
	cmap2.NumDivisions = 7
	cmap2.Set()
	lc2 := tiles.At(legendc, 1, 0)
	cmap2.Legend(&lc2, "(Population/km² - mean) / mean")

	m2 := carto.NewCanvas(b.Max.Y, b.Min.Y, b.Max.X, b.Min.X, panelB)
	for i, blckgrp := range blckgrps2 {
		color := cmap2.GetColor(cartoPopDensity[i])
		lineStyle.Color = color
		m2.DrawVector(blckgrp, color, lineStyle, draw.GlyphStyle{})
	}

	// Now, draw the county boundaries.
	d2, err := shp.NewDecoder("testdata/WA_Counties_2010.shp")
	if err != nil {
		log.Panic(err)
	}
	defer d2.Close()

	type countyData struct {
		geom.Polygon
	}
	var counties []geom.Polygon
	for {
		var rec countyData
		if more := d2.DecodeRow(&rec); !more {
			break
		}
		counties = append(counties, rec.Polygon)
		m.DrawVector(rec.Polygon, color.NRGBA{}, draw.LineStyle{
			Width: 0.25 * vg.Millimeter,
			Color: color.Black,
		}, draw.GlyphStyle{})
	}
	if err = d2.Error(); err != nil {
		log.Panic(err)
	}

	counties2 := cartogram.TransformPolygons(counties)
	for _, c := range counties2 {
		m2.DrawVector(c, color.NRGBA{}, draw.LineStyle{
			Width: 0.25 * vg.Millimeter,
			Color: color.Black,
		}, draw.GlyphStyle{})
	}

	// Next, make a maps of our tilegrams.
	weights := make([]float64, hex.Len())
	for i := 0; i < hex.Len(); i++ {
		weights[i] = hex.Weight(i)
	}
	avg = stat.Mean(weights, nil)
	floats.AddConst(-avg, weights)
	floats.Scale(1/avg, weights)

	cmap = carto.NewColorMap(carto.Linear)
	cmap.AddArray(weights)
	panelC := tiles.At(dc, 2, 0)
	cmap.Set()
	lc = tiles.At(legendc, 2, 0)
	cmap.Legend(&lc, "(Population-mean)/mean")
	m = carto.NewCanvas(b.Max.Y, b.Min.Y, b.Max.X, b.Min.X, panelC)
	for i, hex := range hex.Hexes() {
		c := cmap.GetColor(weights[i])
		lineStyle.Color = c
		m.DrawVector(hex.Geom(), c, lineStyle, draw.GlyphStyle{})
	}
	// Now we add our county boundaries.
	for _, g := range hex.GroupGeom(r / 2) {
		m.DrawVector(g, color.NRGBA{}, draw.LineStyle{
			Width: 0.25 * vg.Millimeter,
			Color: color.Black,
		}, draw.GlyphStyle{})
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
