package quadedge

import (
	"github.com/gdey/quad-edge/geometry"
	"github.com/go-spatial/geom"
)

func (e *Edge) AsGeomLine() *geom.Line {
	if e == nil {
		return nil
	}
	return &geom.Line{
		geometry.UnwrapPoint(*e.Orig()),
		geometry.UnwrapPoint(*e.Dest()),
	}
}
