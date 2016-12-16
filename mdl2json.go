/*
 * mdl2json
 *
 * Copyright (C) 2016 Florian Zwoch <fzwoch@gmail.com>
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

type Vec3 struct {
	X float32
	Y float32
	Z float32
}

type MdlHeader struct {
	Id           uint32
	Version      uint32
	Scale        Vec3
	Origin       Vec3
	Radius       float32
	Offsets      Vec3
	NumSkins     uint32
	SkinWidth    uint32
	SkinHeight   uint32
	NumVerts     uint32
	NumTriangles uint32
	NumFrames    uint32
	SyncType     uint32
	Flags        uint32
	Size         float32
}

type Skin struct {
	Type uint32
}

type SkinGroup struct {
	NumSkins uint32
	Time     float32
}

type STVert struct {
	OnSeam uint32
	S      uint32
	T      uint32
}

type Triangle struct {
	Front  uint32
	Vertex [3]uint32
}

type Vert struct {
	V      [3]uint8
	Normal uint8
}

type Frame struct {
	Type uint32
	Min  Vert
	Max  Vert
	Name [16]uint8
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %v <model.mdl>", os.Args[0])
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	mdl_name := filepath.Base(strings.TrimSuffix(os.Args[1], filepath.Ext(os.Args[1])))

	var mdl MdlHeader

	err = binary.Read(file, binary.LittleEndian, &mdl)
	if err != nil {
		log.Fatal(err)
	}

	if mdl.Id != 1330660425 {
		log.Fatalf("MDL magic %v != \"IDPO\"", mdl.Version)
	}

	if mdl.Version != 6 {
		log.Fatalf("MDL version %v != 6", mdl.Version)
	}

	for i := 0; i < int(mdl.NumSkins); i++ {
		var skin Skin
		var skin_group SkinGroup

		skin_group.NumSkins = 1

		_ = binary.Read(file, binary.LittleEndian, &skin)

		if skin.Type != 0 {
			_ = binary.Read(file, binary.LittleEndian, &skin_group)
		}

		_, _ = file.Seek(int64(skin_group.NumSkins*mdl.SkinWidth*mdl.SkinHeight), io.SeekCurrent)
	}

	stverts := make([]STVert, mdl.NumVerts)
	triangles := make([]Triangle, mdl.NumTriangles)

	err = binary.Read(file, binary.LittleEndian, &stverts)
	err = binary.Read(file, binary.LittleEndian, &triangles)

	verts := [][]Vert{}
	frame_names := make([]string, mdl.NumFrames)

	for k := 0; k < int(mdl.NumFrames); k++ {
		var frame Frame

		_ = binary.Read(file, binary.LittleEndian, &frame)

		if frame.Type != 0 {
			log.Fatal("FIXME")
		}

		frame_verts := make([]Vert, mdl.NumVerts)
		_ = binary.Read(file, binary.LittleEndian, &frame_verts)
		verts = append(verts, frame_verts)

		name := strings.Split(string(frame.Name[:]), "\x00")

		if name[0] == "" {
			name[0] = "Frame " + strconv.Itoa(k)
		}

		frame_names[k] = name[0]
	}

	out, err := os.Create(mdl_name + ".json")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	out.WriteString("{\n")
	out.WriteString("\t\"materials\": [")
	out.WriteString("\n\t{\n")
	out.WriteString("\t\t\"mapDiffuse\": \"textures/" + mdl_name + ".jpg\",\n")
	out.WriteString("\t\t\"mapDiffuseWrap\": [\"repeat\", \"repeat\"]\n")
	out.WriteString("\t} ],\n")
	out.WriteString("\t\"vertices\": [")

	for _, vertice := range verts[0] {
		out.WriteString(strconv.FormatFloat(float64(mdl.Scale.X*float32(vertice.V[0])+mdl.Origin.X), 'f', -1, 32) + ",")
		out.WriteString(strconv.FormatFloat(float64(mdl.Scale.Y*float32(vertice.V[1])+mdl.Origin.Y), 'f', -1, 32) + ",")
		out.WriteString(strconv.FormatFloat(float64(mdl.Scale.Z*float32(vertice.V[2])+mdl.Origin.Z), 'f', -1, 32) + ",")
	}
	out.Seek(-1, io.SeekCurrent)

	out.WriteString("],\n")

	if len(verts) > 1 {
		out.WriteString("\t\"morphTargets\": [\n")

		for i, frame := range verts {
			out.WriteString("\t\t{ \"name\": \"" + frame_names[i] + "\", \"vertices\": [")
			for _, vertice := range frame {
				out.WriteString(strconv.FormatFloat(float64(mdl.Scale.X*float32(vertice.V[0])+mdl.Origin.X), 'f', -1, 32) + ",")
				out.WriteString(strconv.FormatFloat(float64(mdl.Scale.Y*float32(vertice.V[1])+mdl.Origin.Y), 'f', -1, 32) + ",")
				out.WriteString(strconv.FormatFloat(float64(mdl.Scale.Z*float32(vertice.V[2])+mdl.Origin.Z), 'f', -1, 32) + ",")
			}
			out.Seek(-1, io.SeekCurrent)
			out.WriteString("] },\n")
		}
		out.Seek(-2, io.SeekCurrent)

		out.WriteString("\n\t],\n")
	}

	out.WriteString("\t\"uvs\": [[")

	for _, stvertice := range stverts {
		out.WriteString(strconv.FormatFloat(float64(stvertice.S)/float64(mdl.SkinWidth), 'f', -1, 32) + ",")
		out.WriteString(strconv.FormatFloat(1-float64(stvertice.T)/float64(mdl.SkinHeight), 'f', -1, 32) + ",")
	}

	var needs_hack bool = false

	for _, stvertice := range stverts {
		if stvertice.OnSeam != 0 {
			needs_hack = true
			break
		}
	}

	if needs_hack == true {
		for _, stvertice := range stverts {
			if stvertice.OnSeam == 0 {
				out.WriteString("0,0,")
			} else {
				out.WriteString(strconv.FormatFloat((float64(stvertice.S)+float64(mdl.SkinWidth/2))/float64(mdl.SkinWidth), 'f', -1, 32) + ",")
				out.WriteString(strconv.FormatFloat(1-float64(stvertice.T)/float64(mdl.SkinHeight), 'f', -1, 32) + ",")
			}
		}
	}
	out.Seek(-1, io.SeekCurrent)

	out.WriteString("]],\n")
	out.WriteString("\t\"faces\": [")

	for _, triangle := range triangles {
		out.WriteString("10,")
		out.WriteString(strconv.Itoa(int(triangle.Vertex[0])) + ",")
		out.WriteString(strconv.Itoa(int(triangle.Vertex[2])) + ",")
		out.WriteString(strconv.Itoa(int(triangle.Vertex[1])) + ",")
		out.WriteString("0,")

		var uvs [3]int

		for i, vertex := range triangle.Vertex {
			uvs[i] = int(vertex)
			if triangle.Front == 0 && stverts[vertex].OnSeam != 0 {
				uvs[i] += int(mdl.NumVerts)
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
