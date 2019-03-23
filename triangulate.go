package qetriangulate

import (
	"log"
	"sort"

	"github.com/gdey/quad-edge/geometry"
	"github.com/gdey/quad-edge/subdivision"
	"github.com/go-spatial/geom/cmp"
)

type Triangulator struct {
	points [][2]float64

	sd  *subdivision.Subdivision
	tri [3]geometry.Point
}

func New(pts ...[2]float64) *Triangulator {

	return &Triangulator{
		points: pts,
	}
}
func (t *Triangulator) InitSubdivision() {
	sort.Sort(cmp.ByXY(t.points))
	tri := geometry.TriangleContaining(t.points...)
	t.tri = [3]geometry.Point{geometry.NewPoint(tri[0][0], tri[0][1]), geometry.NewPoint(tri[1][0], tri[1][1]), geometry.NewPoint(tri[2][0], tri[2][1])}
	t.sd = subdivision.New(t.tri[0], t.tri[1], t.tri[2])
	var oldPt geometry.Point
	for i, pt := range t.points {
		bfpt := geometry.NewPoint(pt[0], pt[1])
		if i != 0 && geometry.ArePointsEqual(oldPt, bfpt) {
			continue
		}
		oldPt = bfpt
		if !t.sd.InsertSite(bfpt) {
			log.Printf("Failed to insert point %v", bfpt)
		}
	}
}
