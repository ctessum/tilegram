// Copyright ©2016 The tilegram Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tilegram

import (
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
)

// Grouper is a holder for an individual spatial unit that is to be
// assigned to one of the tile groups in a tilegram.
type Grouper interface {
	// Centroid returns the geomtric center of the reciever.
	Centroid() geom.Point

	// Bounds returns the bounding box of the receiver geometry
	Bounds() *geom.Bounds

	// Weight is the value of the characteristic attribute
	// of the receiver. Population count is a typical weight value.
	Weight() float64

	// Group returns the group that the receiver belongs to.
	// All values in a given tile are required to be of the same group.
	// Typical groups would be political units such as states or provinces.
	Group() string

	// Similar implements the geom.Geom interface.
	Similar(geom.Geom, float64) bool

	// Transform implements the geom.Geom interface.
	Transform(proj.Transformer) (geom.Geom, error)
}

// Data is an implementation of the Grouper interface.
type Data struct {
	centroid geom.Point
	bounds   *geom.Bounds
	weight   float64
	group    string
}

// NewData creates a new data value.
func NewData(g geom.Polygonal, weight float64, group string) *Data {
	return &Data{
		centroid: g.Centroid(),
		bounds:   g.Bounds(),
		weight:   weight,
		group:    group,
	}
}

// Centroid implements the Grouper interface.
func (d *Data) Centroid() geom.Point {
	return d.centroid
}

// Weight implements the Grouper interface.
func (d *Data) Weight() float64 {
	return d.weight
}

// Group implements the Grouper interface.
func (d *Data) Group() string {
	return d.group
}

// Bounds implements the Grouper interface.
func (d *Data) Bounds() *geom.Bounds {
	return d.bounds
}

// Similar implements the geom.Geom interface.
func (d *Data) Similar(_ geom.Geom, _ float64) bool {
	return false
}

// Transform implements the geom.Geom interface.
func (d *Data) Transform(_ proj.Transformer) (geom.Geom, error) {
	return nil, nil
}
