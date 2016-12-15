// Copyright ©2016 The tilegram Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tilegram

import (
	"fmt"
	"sort"
)

// Distribute spatially distributes the data items among hexagons
// so that the ratio of the range of total data weights in different
// hexagons to their mean among all hexagons is no greater than range ratio.
func (h *Hexagram) Distribute(rangeRatio float64) {
	for {
		ratio, mean := h.rangeRatio()
		fmt.Println(ratio, rangeRatio)
		if ratio <= rangeRatio {
			break
		}
		for _, hh := range h.hexes {
			if hh.Weight() > mean {
				for _, n := range h.neighbors(hh) {
					wdiff := hh.Weight() - n.Weight()
					if wdiff > 0 {
						hh.transfer(n, wdiff/2)
					}
				}
			}
		}
	}
}

// transfer transfers data items with weights approximately
// equalling the specified amount to neighbor n.
func (h *Hex) transfer(n *Hex, amount float64) {
	h.sort(n)
	var i int
	var sum float64
	for _, d := range h.Data {
		// Add data items to the neighbor.
		sum += d.Weight()
		n.Data = append(n.Data, d)
		i++
		if sum >= amount {
			break
		}
	}
	for j := 0; j < i; j++ {
		// Delete data items from the receiver.
		copy(h.Data[0:], h.Data[1:])
		h.Data[len(h.Data)-1] = nil // or the zero value of T
		h.Data = h.Data[:len(h.Data)-1]
	}
}

// sort sorts the data items in h in order of desirability for
// transfer to n.
func (h *Hex) sort(n *Hex) {
	sort.Sort(hexSort{h: h, n: n})
}

type hexSort struct {
	h, n *Hex
}

func (h hexSort) Len() int      { return len(h.h.Data) }
func (h hexSort) Swap(i, j int) { h.h.Data[j], h.h.Data[i] = h.h.Data[i], h.h.Data[j] }

// Less sorts the data items so that the items with highest priority
// for transfer are at the beginning of the list.
func (h hexSort) Less(i, j int) bool {
	if h.h.Data[i].Group() == h.n.Group() && h.h.Data[j].Group() != h.n.Group() {
		// Prioritize transfering items that are of the same group as the neighbor.
		return true
	}
	// Prioritize transfering items that are lower weight so as to keep
	// higher weight items closer to their original location.
	return h.h.Data[i].Weight() < h.h.Data[j].Weight()
}
