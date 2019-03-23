package subdivision

import (
	"errors"
	"log"

	"github.com/gdey/quad-edge/geometry"
	"github.com/gdey/quad-edge/quadedge"
)

var (
	ErrCancel          = errors.New("canceled walk")
	ErrCoincidentEdges = errors.New("coincident edges")
)

type Subdivision struct {
	startingEdge *quadedge.Edge
	ptcount      int
	frame        [3]geometry.Point
}

// New initialize a subdivision to the triangle defined by the points a,b,c.
func New(a, b, c geometry.Point) *Subdivision {
	ea := quadedge.New()
	ea.EndPoints(&a, &b)
	eb := quadedge.New()
	quadedge.Splice(ea.Sym(), eb)
	eb.EndPoints(&b, &c)
	ec := quadedge.New()
	quadedge.Splice(eb.Sym(), ec)
	ec.EndPoints(&c, &a)
	quadedge.Splice(ec.Sym(), ea)
	return &Subdivision{
		startingEdge: ea,
		ptcount:      3,
		frame:        [3]geometry.Point{a, b, c},
	}
}

func ptEqual(x geometry.Point, a *geometry.Point) bool {
	if a == nil {
		return false
	}
	return geometry.ArePointsEqual(*a, x)
}

func testEdge(x geometry.Point, e *quadedge.Edge) (bool, *quadedge.Edge) {
	switch {
	case ptEqual(x, e.Org()) || ptEqual(x, e.Dest()):
		return true, e
	case quadedge.RightOf(x, e):
		//	log.Printf("Right of e -> Sym: %p :: %p", e, e.Sym())
		return false, e.Sym()
	case !quadedge.RightOf(x, e.ONext()):
		//	log.Printf("!Right of e.ONext  -> ONext: %p :: %p", e, e.ONext())
		return false, e.ONext()
	case !quadedge.RightOf(x, e.DPrev()):
		//	log.Printf("!Right of e.DPrev  -> DPrev: %p :: %p", e, e.DPrev())
		return false, e.DPrev()
	default:
		return true, e
	}
}

// locate returns an edge e, s.t. either x is on e, or e is an edge of
// a triangle containing x. The search starts from startingEdge
// and proceeds in the general direction of x. Based on the
// pseudocode in Guibas and Stolfi (1985) p.121
func (sd *Subdivision) locate(x geometry.Point) (*quadedge.Edge, bool) {
	var (
		e     *quadedge.Edge
		ok    bool
		count int
	)
	for ok, e = testEdge(x, sd.startingEdge); !ok; ok, e = testEdge(x, e) {

		count++
		if e == sd.startingEdge || count > sd.ptcount*2 {
			log.Println("searching all edges for", x)
			e = nil

			WalkAllEdges(sd.startingEdge, func(ee *quadedge.Edge) error {
				if ok, _ = testEdge(x, ee); ok {
					log.Printf("Found the edge %p", ee)
					e = ee
					return ErrCancel
				}
				return nil
			})
			log.Printf(
				"Got back to starting edge after %v iterations, only have %v points ",
				count,
				sd.ptcount,
			)
			return e, false
		}
	}
	return e, true
}

func (sd *Subdivision) locateSegment(startingEdge *quadedge.Edge, end geometry.Point) *quadedge.Edge {
	if startingEdge == nil {
		return nil
	}
	curr := startingEdge
	for backToStart := false; !backToStart; backToStart = curr == startingEdge {
		if geometry.ArePointsEqual(end, *curr.Dest()) {
			return curr
		}
		curr = curr.ONext()
	}
	// Did not find and edge with vertex start, end
	return nil
}
func (sd *Subdivision) LocateSegment(start, end geometry.Point) *quadedge.Edge {
	startingEdge, _ := sd.locate(start)
	return sd.locateSegment(startingEdge, end)
}

// InsertSite will insert a new point into a subdivision representing a Delaunay
// triangulation, and fixes the affected edges so that the result
// is  still a Delaunay triangulation. This is based on the pseudocode
// from Guibas and Stolfi (1985) p.120, with slight modificatons and a bug fix.
func (sd *Subdivision) InsertSite(x geometry.Point) bool {
	sd.ptcount++
	e, got := sd.locate(x)
	if !got {
		// Did not find the edge using normal walk
		return false
	}
	if ptEqual(x, e.Org()) || ptEqual(x, e.Dest()) {
		// Point is already in subdivision
		return true
	}
	if quadedge.OnEdge(x, e) {
		e = e.OPrev()
		quadedge.Delete(e.ONext())
	}

	// Connect the new point to the vertices of the containing
	// triangle (or quadrilaterial, if the new point fell on an
	// existing edge.)
	base := quadedge.NewWithEndPoints(e.Org(), &x)
	quadedge.Splice(base, e)
	sd.startingEdge = base

	base = quadedge.Connect(e, base.Sym())
	e = base.OPrev()
	for e.LNext() != sd.startingEdge {
		base = quadedge.Connect(e, base.Sym())
		e = base.OPrev()
	}

	// Examine suspect edges to ensure that the Delaunay condition
	// is satisfied.
	for {
		t := e.OPrev()
		switch {
		case quadedge.RightOf(*t.Dest(), e) &&
			geometry.InCircle(*e.Org(), *t.Dest(), *e.Dest(), x):
			quadedge.Swap(e)
			e = e.OPrev()

		case e.ONext() == sd.startingEdge: // no more suspect edges
			return true
		default: // pop a suspect edge
			e = e.ONext().LPrev()
		}
	}
	return true
}

func (sd *Subdivision) InsertConstaint(start, end geometry.Point) error {
	var (
		pu []geometry.Point
		pl []geometry.Point
	)

	startingEdge, _ := sd.locate(start)
	if startingEdge == nil {
		// start is not in our subdivision
		return errors.New("Invlid starting vertex.")
	}
	if e := sd.locateSegment(startingEdge, end); e != nil {
		// Nothing to do, edge already in the subdivision.
		return nil
	}
	removalList, err := IntersectingEdges(startingEdge, end)
	if err != nil {
		return err
	}

	pu = append(pu, start)
	pl = append(pl, start)

	for _, e := range removalList {
		if IsHardFrameEdge(sd.frame, e) {
			continue
		}
		for _, spoint := range [2]geometry.Point{*e.Org(), *e.Dest()} {
			switch Classify(spoint, start, end) {
			case LEFT:
				pl = geometry.AppendNonRepeat(pl, spoint)
			case RIGHT:
				pu = geometry.AppendNonRepeat(pu, spoint)
			default:
				// should not come here.

			}
		}
		quadedge.Delete(e)
	}

	pl = geometry.AppendNonRepeat(pl, end)
	pu = geometry.AppendNonRepeat(pu, end)

	for _, pts := range [2][]geometry.Point{pu, pl} {
		if len(pts) == 2 {
			// just a shared line, no points to triangulate.
			continue
		}

		edges, err := triangulatePseudoPolygon(pts)
		if err != nil {
			return err
		}

		for _, edge := range edges {

			// First we need to check that the edge does not intersect other edges, this can happen if
			// the polygon we were wer triangulating happens to be concave. In which case it is possible
			// a triangle outside of the "ok" region, and we should ignore those edges

			// Original code think this is a bug: intersectList, _ := intersectingEdges(startingEdge,end)
			{
				startingEdge, _ := sd.locate(edge[0])
				intersectList, _ := IntersectingEdges(startingEdge, edge[1])
				if len(intersectList) > 0 {
					continue
				}
			}

			if err = sd.insertEdge(edge[0], edge[1]); err != nil {
				return err
			}

		}
	}

	return nil
}

func (sd *Subdivision) insertEdge(start, end geometry.Point) error {
	edge, _ := sd.locate(start)
	if edge == nil {
		// start is not in our subdivision
		return errors.New("Invlid starting vertex.")
	}
	if e := sd.locateSegment(edge, end); e != nil {
		// Nothing to do, edge already in the subdivision.
		return nil
	}
	// Only Error it gives is ErrCoincidentEdges, and we are fine with those.
	ct, _ := FindIntersectingTriangle(edge, end)
	if ct == nil {
		return errors.New("did not find an intersecting trinagle. assumptions broken.")
	}

	from := ct.StartingEdge().Sym()

	symEdge, _ := sd.locate(end)
	if symEdge == nil {
		return errors.New("Invlid ending vertex.")
	}

	ct, _ = FindIntersectingTriangle(symEdge, start)
	if ct == nil {
		return errors.New("did not find an intersecting trinagle. assumptions broken.")
	}

	to := ct.StartingEdge().OPrev()
	_ = quadedge.Connect(from, to)
	return nil
}

// WalkAllEdges will call the provided function for each edge in the subdivision. The walk will
// be terminated if the function returns an error or ErrCancel. ErrCancel will not result in
// an error be returned by main function, otherwise the error will be passed on.
func (sd *Subdivision) WalkAllEdges(fn func(e *quadedge.Edge) error) error {

	if sd == nil || sd.startingEdge == nil {
		return nil
	}
	return WalkAllEdges(sd.startingEdge, fn)
}

func (sd *Subdivision) Triangles(includeFrame bool) (triangles [][3]geometry.Point, err error) {

	err = WalkAllTriangleEdges(
		sd.startingEdge,
		func(edges []*quadedge.Edge) error {
			if len(edges) != 3 {
				return errors.New("Something Strange!")
			}

			pts := [3]geometry.Point{*edges[0].Org(), *edges[1].Org(), *edges[2].Org()}

			// Do we want to skip because the points are part of the frame and
			// we have been requested not to include triangles attached to the frame.
			if IsFramePoint(sd.frame, pts[:]...) && !includeFrame {
				return nil
			}

			triangles = append(triangles, pts)
			return nil
		},
	)
	return triangles, err
}

func WalkAllEdges(se *quadedge.Edge, fn func(e *quadedge.Edge) error) error {
	if se == nil {
		return nil
	}
	var (
		toProcess quadedge.Stack
		visited   = make(map[*quadedge.Edge]bool)
	)
	toProcess.Push(se)
	for toProcess.Length() > 0 {
		e := toProcess.Pop()
		if visited[e] {
			continue
		}

		if err := fn(e); err != nil {
			if err == ErrCancel {
				return nil
			}
			return err
		}

		sym := e.Sym()

		toProcess.Push(e.ONext())
		toProcess.Push(sym.ONext())

		visited[e] = true
		visited[sym] = true
	}
	return nil
}

// IsFrameEdge indicates if the edge is part of the given frame.
func IsFrameEdge(frame [3]geometry.Point, es ...*quadedge.Edge) bool {
	for _, e := range es {
		o, d := *e.Org(), *e.Dest()
		of := geometry.ArePointsEqual(o, frame[0]) || geometry.ArePointsEqual(o, frame[1]) || geometry.ArePointsEqual(o, frame[2])
		df := geometry.ArePointsEqual(d, frame[0]) || geometry.ArePointsEqual(d, frame[1]) || geometry.ArePointsEqual(d, frame[2])
		if of || df {
			return true
		}
	}
	return false
}

// IsFrameEdge indicates if the edge is part of the given frame where both vertexs are part of the frame.
func IsHardFrameEdge(frame [3]geometry.Point, e *quadedge.Edge) bool {
	o, d := *e.Org(), *e.Dest()
	of := geometry.ArePointsEqual(o, frame[0]) || geometry.ArePointsEqual(o, frame[1]) || geometry.ArePointsEqual(o, frame[2])
	df := geometry.ArePointsEqual(d, frame[0]) || geometry.ArePointsEqual(d, frame[1]) || geometry.ArePointsEqual(d, frame[2])
	return of && df
}

func IsFramePoint(frame [3]geometry.Point, pts ...geometry.Point) bool {
	for _, pt := range pts {
		if geometry.ArePointsEqual(pt, frame[0]) ||
			geometry.ArePointsEqual(pt, frame[1]) ||
			geometry.ArePointsEqual(pt, frame[2]) {
			return true
		}
	}
	return false

}

func constructTriangleEdges(
	e *quadedge.Edge,
	toProcess *quadedge.Stack,
	visited map[*quadedge.Edge]bool,
	fn func(edges []*quadedge.Edge) error,
) error {

	if visited[e] {
		return nil
	}

	curr := e
	var triedges []*quadedge.Edge
	for backToStart := false; !backToStart; backToStart = curr == e {

		// Collect edge
		triedges = append(triedges, curr)

		sym := curr.Sym()
		if !visited[sym] {
			toProcess.Push(sym)
		}

		// mark edge as visted
		visited[curr] = true

		// Move the ccw edge
		curr = curr.LNext()
	}
	return fn(triedges)
}

// WalkAllTriangleEdges will walk the subdivision starting from the starting edge (se) and return
// sets of edges that make make a triangle for each face.
func WalkAllTriangleEdges(se *quadedge.Edge, fn func(edges []*quadedge.Edge) error) error {
	if se == nil {
		return nil
	}
	var (
		toProcess quadedge.Stack
		visited   = make(map[*quadedge.Edge]bool)
	)
	toProcess.Push(se)
	for toProcess.Length() > 0 {
		e := toProcess.Pop()
		if visited[e] {
			continue
		}
		err := constructTriangleEdges(e, &toProcess, visited, fn)
		if err != nil {
			if err == ErrCancel {
				return nil
			}
			return err
		}
	}
	return nil
}

func FindIntersectingTriangle(startingEdge *quadedge.Edge, end geometry.Point) (*Triangle, error) {
	var (
		//start = startingEdge.Org()
		left  = startingEdge
		right *quadedge.Edge
	)

	for {
		right = left.OPrev()

		lc := Classify(end, *left.Org(), *left.Dest())
		rc := Classify(end, *right.Org(), *right.Dest())

		if (lc == RIGHT && rc == LEFT) ||
			lc == BETWEEN ||
			lc == DESTINATION ||
			lc == BEYOND {
			return &Triangle{left}, nil
		}

		if lc != RIGHT && lc != LEFT && rc != RIGHT && rc != LEFT {
			return &Triangle{left}, ErrCoincidentEdges
		}
		left = right
		if left == startingEdge {
			// We have walked all around the vertex.
			break
		}

	}
	return nil, nil
}

func IntersectingEdges(startingEdge *quadedge.Edge, end geometry.Point) (intersected []*quadedge.Edge, err error) {

	var (
		start        = startingEdge.Org()
		tseq         *Triangle
		pseq         geometry.Point
		shared       *quadedge.Edge
		currentPoint = start
	)

	t, err := FindIntersectingTriangle(startingEdge, end)
	if err != nil {
		return nil, err
	}

	for !t.IntersectsPoint(end) {
		if tseq, err = t.OppositeTriangle(*currentPoint); err != nil {
			return nil, err
		}
		shared = t.SharedEdge(*tseq)
		if shared == nil {
			// Should I panic? This is weird.
			return nil, errors.New("did not find shared edge with Opposite Triangle.")
		}
		pseq = *tseq.OppositeVertex(*t)
		switch Classify(pseq, *start, end) {
		case LEFT:
			currentPoint = shared.Org()
		case RIGHT:
			currentPoint = shared.Dest()
		}
		intersected = append(intersected, shared)
		t = tseq
	}
	return intersected, nil

}
