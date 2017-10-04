// Copyright ©2016 The tilegram Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tilegram

import (
	"github.com/ctessum/geom"
)

// Grouper is a holder for an individual spatial unit that is to be
// assigned to one of the tile groups in a tilegram.
type Grouper interface {
	geom.Polygonal

	// Weight is the value of the characteristic attribute
	// of the receiver. Population count is a typical weight value.
	Weight() float64

	// Group returns the group that the receiver belongs to.
	// All values in a given tile are required to be of the same group.
	// Typical groups would be political units such as states or provinces.
	Group() string
}

// Data is an implementation of the Grouper interface.
type Data struct {
	geom.Polygonal
	W float64 // Weight
	G string  // Group
}

// Weight implements the Grouper interface.
func (d *Data) Weight() float64 {
	return d.W
}

// Group implements the Grouper interface.
func (d *Data) Group() string {
	return d.G
}
