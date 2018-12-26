/*
 * mdl2json
 *
 * Copyright (C) 2016-2018 Florian Zwoch <fzwoch@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type vec3 struct {
	x float32
	y float32
	z float32
}

type mdlHeader struct {
	id           uint32
	version      uint32
	scale        vec3
	origin       vec3
	radius       float32
	offsets      vec3
	numSkins     uint32
	skinWidth    uint32
	skinHeight   uint32
	numVerts     uint32
	numTriangles uint32
	numFrames    uint32
	syncType     uint32
	flags        uint32
	size         float32
}

type skin struct {
	skinType uint32
}

type skinGroup struct {
	numSkins uint32
	time     float32
}

type stVert struct {
	onSeam uint32
	s      uint32
	t      uint32
}

type triangle struct {
	front  uint32
	vertex [3]uint32
}

type vert struct {
	v      [3]uint8
	normal uint8
}

type frame struct {
	frameType uint32
	min       vert
	max       vert
	name      [16]uint8
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %v <input.mdl> <output.json>", os.Args[0])
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	mdlName := filepath.Base(strings.TrimSuffix(os.Args[1], filepath.Ext(os.Args[1])))

	var mdl mdlHeader

	err = binary.Read(file, binary.LittleEndian, &mdl)
	if err != nil {
		log.Fatal(err)
	}

	if mdl.id != 1330660425 {
		log.Fatalf("MDL magic %v != \"IDPO\"", mdl.version)
	}

	if mdl.version != 6 {
		log.Fatalf("MDL version %v != 6", mdl.version)
	}

	for i := 0; i < int(mdl.numSkins); i++ {
		var skin skin
		var skinGroup skinGroup

		skinGroup.numSkins = 1

		_ = binary.Read(file, binary.LittleEndian, &skin)

		if skin.skinType != 0 {
			_ = binary.Read(file, binary.LittleEndian, &skinGroup)
		}

		_, _ = file.Seek(int64(skinGroup.numSkins*mdl.skinWidth*mdl.skinHeight), io.SeekCurrent)
	}

	stverts := make([]stVert, mdl.numVerts)
	triangles := make([]triangle, mdl.numTriangles)

	err = binary.Read(file, binary.LittleEndian, &stverts)
	err = binary.Read(file, binary.LittleEndian, &triangles)

	verts := [][]vert{}
	frameNames := make([]string, mdl.numFrames)

	for k := 0; k < int(mdl.numFrames); k++ {
		var frame frame

		_ = binary.Read(file, binary.LittleEndian, &frame)

		if frame.frameType != 0 {
			panic("FIXME")
		}

		frameVerts := make([]vert, mdl.numVerts)
		_ = binary.Read(file, binary.LittleEndian, &frameVerts)
		verts = append(verts, frameVerts)

		name := strings.Split(string(frame.name[:]), "\x00")

		if name[0] == "" {
			name[0] = "Frame " + strconv.Itoa(k)
		}

		frameNames[k] = name[0]
	}

	out, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	out.WriteString("{\n")
	out.WriteString("\t\"materials\": [")
	out.WriteString("\n\t{\n")
	out.WriteString("\t\t\"mapDiffuse\": \"textures/" + mdlName + ".jpg\",\n")
	out.WriteString("\t\t\"mapDiffuseWrap\": [\"repeat\", \"repeat\"]\n")
	out.WriteString("\t} ],\n")
	out.WriteString("\t\"vertices\": [")

	for _, vertice := range verts[0] {
		out.WriteString(strconv.FormatFloat(float64(mdl.scale.x*float32(vertice.v[0])+mdl.origin.x), 'f', -1, 32) + ",")
		out.WriteString(strconv.FormatFloat(float64(mdl.scale.y*float32(vertice.v[1])+mdl.origin.y), 'f', -1, 32) + ",")
		out.WriteString(strconv.FormatFloat(float64(mdl.scale.z*float32(vertice.v[2])+mdl.origin.z), 'f', -1, 32) + ",")
	}
	out.Seek(-1, io.SeekCurrent)

	out.WriteString("],\n")

	if len(verts) > 1 {
		out.WriteString("\t\"morphTargets\": [\n")

		for i, frame := range verts {
			out.WriteString("\t\t{ \"name\": \"" + frameNames[i] + "\", \"vertices\": [")
			for _, vertice := range frame {
				out.WriteString(strconv.FormatFloat(float64(mdl.scale.x*float32(vertice.v[0])+mdl.origin.x), 'f', -1, 32) + ",")
				out.WriteString(strconv.FormatFloat(float64(mdl.scale.y*float32(vertice.v[1])+mdl.origin.y), 'f', -1, 32) + ",")
				out.WriteString(strconv.FormatFloat(float64(mdl.scale.z*float32(vertice.v[2])+mdl.origin.z), 'f', -1, 32) + ",")
			}
			out.Seek(-1, io.SeekCurrent)
			out.WriteString("] },\n")
		}
		out.Seek(-2, io.SeekCurrent)

		out.WriteString("\n\t],\n")
	}

	out.WriteString("\t\"uvs\": [[")

	for _, stvertice := range stverts {
		out.WriteString(strconv.FormatFloat(float64(stvertice.s)/float64(mdl.skinWidth), 'f', -1, 32) + ",")
		out.WriteString(strconv.FormatFloat(1-float64(stvertice.t)/float64(mdl.skinHeight), 'f', -1, 32) + ",")
	}

	var needsHack bool

	for _, stvertice := range stverts {
		if stvertice.onSeam != 0 {
			needsHack = true
			break
		}
	}

	if needsHack == true {
		for _, stvertice := range stverts {
			if stvertice.onSeam == 0 {
				out.WriteString("0,0,")
			} else {
				out.WriteString(strconv.FormatFloat((float64(stvertice.s)+float64(mdl.skinWidth/2))/float64(mdl.skinWidth), 'f', -1, 32) + ",")
				out.WriteString(strconv.FormatFloat(1-float64(stvertice.t)/float64(mdl.skinHeight), 'f', -1, 32) + ",")
			}
		}
	}
	out.Seek(-1, io.SeekCurrent)

	out.WriteString("]],\n")
	out.WriteString("\t\"faces\": [")

	for _, triangle := range triangles {
		out.WriteString("10,")
		out.WriteString(strconv.Itoa(int(triangle.vertex[0])) + ",")
		out.WriteString(strconv.Itoa(int(triangle.vertex[2])) + ",")
		out.WriteString(strconv.Itoa(int(triangle.vertex[1])) + ",")
		out.WriteString("0,")

		var uvs [3]int

		for i, vertex := range triangle.vertex {
			uvs[i] = int(vertex)
			if triangle.front == 0 && stverts[vertex].onSeam != 0 {
				uvs[i] += int(mdl.numVerts)
			}
		}

		out.WriteString(strconv.Itoa(uvs[0]) + ",")
		out.WriteString(strconv.Itoa(uvs[2]) + ",")
		out.WriteString(strconv.Itoa(uvs[1]) + ",")
	}
	out.Seek(-1, io.SeekCurrent)

	out.WriteString("]\n")
	out.WriteString("}\n")
}
