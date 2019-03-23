package qetriangulate_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/gdey/quad-edge/debugger"
	"github.com/gdey/quad-edge/geometry"
	"github.com/gdey/quad-edge/quadedge"
	"github.com/gdey/quad-edge/subdivision"
	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/cmp"
	"github.com/go-spatial/geom/encoding/wkt"
)


func logEdges(sd *subdivision.Subdivision) {
	_ = sd.WalkAllEdges(func(e *quadedge.Edge) error {
		org := *e.Org()
		dst := *e.Dest()

		fmt.Println(wkt.MustEncode(
			geom.Line{
				geometry.UnwrapPoint(org),
				geometry.UnwrapPoint(dst),
			},
		))

		return nil
	})
}

func Draw(rec debugger.Recorder, name string, pts ...[2]float64) {

	sort.Sort(cmp.ByXY(pts))

	ffl := debugger.FFL(0)
	tri := geometry.TriangleContaining(pts...)
	rec.Record(
		geom.Triangle(tri),
		ffl,
		debugger.TestDescription{
			Category:    "frame:triangle",
			Description: "triangle frame.",
			Name:        name,
		},
	)
	ext := geometry.Extent(pts...)
	rec.Record(
		geom.Extent{
			float64(ext[0][0]), float64(ext[0][1]),
			float64(ext[1][0]), float64(ext[1][1]),
		},
		ffl,
		debugger.TestDescription{
			Category:    "frame:extent",
			Description: "extent frame.",
			Name:        name,
		},
	)
	sd := subdivision.New(geometry.NewPoint(tri[0][0], tri[0][1]), geometry.NewPoint(tri[1][0], tri[1][1]), geometry.NewPoint(tri[2][0], tri[2][1]))
	badcount := 0
	for i, pt := range pts {
		if i != 0 && pts[i-1][0] == pt[0] && pts[i-1][1] == pt[1] {
			continue
		}
		bfpt := geometry.NewPoint(pt[0], pt[1])
		if !sd.InsertSite(bfpt) {
			rec.Record(
				geometry.UnwrapPoint(bfpt),
				ffl,
				debugger.TestDescription{
					Category:    "initial:point:failed",
					Description: fmt.Sprintf("point:%v %v:failed", i, geometry.UnwrapPoint(bfpt)),
					Name:        name,
				},
			)
			badcount++
			continue
		}
		rec.Record(
			geometry.UnwrapPoint(bfpt),
			ffl,
			debugger.TestDescription{
				Category:    "initial:point",
				Description: fmt.Sprintf("point:%v %v", i, geometry.UnwrapPoint(bfpt)),
				Name:        name,
			},
		)
	}
	if badcount > 0 {
		log.Printf("Failed to insert %v points\n", badcount)
	}

	count := 0
	_ = sd.WalkAllEdges(func(e *quadedge.Edge) error {
		org := *e.Org()
		dst := *e.Dest()

		rec.Record(
			geom.Line{
				geometry.UnwrapPoint(org),
				geometry.UnwrapPoint(dst),
			},
			ffl,
			debugger.TestDescription{
				Category: fmt.Sprintf("edge:%v", count),
				Description: fmt.Sprintf(
					"edge:%v [%p]( %v %v, %v %v)",
					count, e,
					org[0], org[1],
					dst[0], dst[1],
				),
				Name: name,
			},
		)
		count++
		return nil
	})
	triangles, err := sd.Triangles(true) 
	if err != nil {
		log.Printf("Got an error: %v",err)
	}
	for i, tri := range triangles {
		rec.Record(
			geom.Triangle{
				geometry.UnwrapPoint(tri[0]),
				geometry.UnwrapPoint(tri[1]),
				geometry.UnwrapPoint(tri[2]),
			},
			ffl,
			debugger.TestDescription{
				Category: fmt.Sprintf("triangle:%v", i),
				Description: fmt.Sprintf(
					"triangle:%v (%v)", i,tri,
				),
				Name: name,
			},
		)
	}
}

func cleanup(data []byte) (parts []string) {
	toreplace := []byte(`[]{}(),;`)
	for _, v := range toreplace {
		data = bytes.ReplaceAll(data, []byte{v}, []byte(" "))
	}
	dparts := bytes.Split(data, []byte(` `))
	for _, dpt := range dparts {
		s := bytes.TrimSpace(dpt)
		if len(s) == 0 {
			continue
		}
		parts = append(parts, string(s))
	}
	return parts
}

func gettests(inputdir, mid string, ts map[string][][2]float64) {
	files, err := ioutil.ReadDir(inputdir)
	if err != nil {
		panic(
			fmt.Sprintf("Could not read dir %v: %v", inputdir, err),
		)
	}
	var filename string
	for _, file := range files {
		if mid != "" {
			filename = filepath.Join(mid, file.Name())
		} else {
			filename = file.Name()
		}
		if file.IsDir() {
			gettests(inputdir, filename, ts)
			continue
		}
		idx := strings.LastIndex(filename, ".points")
		if idx == -1 || filename[idx:] != ".points" {
			continue
		}
		data, err := ioutil.ReadFile(filepath.Join(inputdir, filename))
		if err != nil {
			panic(
				fmt.Sprintf("Could not read file %v: %v", filename, err),
			)
		}
		var pts [][2]float64

		// clean up file of { [ ( , ;
		parts := cleanup(data)
		if len(parts)%2 != 0 {
			panic(
				fmt.Sprintf("Badly formatted file %v:\n%v\n%s", filename, parts, data),
			)
		}
		for i := 0; i < len(parts); i += 2 {
			x, err := strconv.ParseFloat(parts[i], 64)
			if err != nil {
				panic(
					fmt.Sprintf("%v::%v: Badly formatted value {{%v}}:%v\n%s", filename, i, parts[i], err, data),
				)
			}
			y, err := strconv.ParseFloat(parts[i+1], 64)
			if err != nil {
				panic(
					fmt.Sprintf("%v::%v: Badly formatted value {{%v}}:%v\n%s", filename, i+1, parts[i], err, data),
				)
			}
			pts = append(pts, [2]float64{x, y})
		}
		ts[filename[:idx]] = pts
	}
}

func TestTriangulation(t *testing.T) {

	debugger.DefaultOutputDir = "output"
	const inputdir = "testdata"

	/*
		tests := map[string][]geometry.Point{
			"First Test": {
				{516, 661}, {369, 793}, {426, 539}, {273, 525}, {204, 694}, {747, 750}, {454, 390},
			},
			"Second test": {
				{382, 302}, {382, 328}, {382, 205}, {623, 175}, {382, 188}, {382, 284}, {623, 87}, {623, 341}, {141, 227},
			},
		}
	*/
	tests := make(map[string][][2]float64)
	gettests(inputdir, "", tests)

	for name, pts := range tests {
		t.Run(name, func(t *testing.T) {

			var rec debugger.Recorder
			rec, _ = debugger.AugmentRecorder(rec, fmt.Sprintf("%v_drawn_%v", geometry.Type, name))
			Draw(rec, name, pts...)
			rec.CloseWait()
		})
	}

}
