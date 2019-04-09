package subdivision

import (
	"errors"
	"fmt"

	"github.com/gdey/quad-edge/debugger"
	"github.com/gdey/quad-edge/geometry"
	"github.com/gdey/quad-edge/quadedge"
	"github.com/go-spatial/geom"
)

const debug = true

func ErrAssumptionFailed() error {
	str := fmt.Sprintf("Assumption failed at: %v ", debugger.FFL(0))
	if debug {
		return errors.New(str)
	}
	panic(str)
}

func DumpSubdivision(sd *Subdivision) {
		fmt.Printf("Frame: %#v\n", sd.frame)

		_ = sd.WalkAllEdges(func(e *quadedge.Edge) error {
			org := *e.Orig()
			dst := *e.Dest()

			fmt.Printf("%#v", geom.Line{
				geometry.UnwrapPoint(org),
				geometry.UnwrapPoint(dst),
			},
			)

			return nil
		})
}
