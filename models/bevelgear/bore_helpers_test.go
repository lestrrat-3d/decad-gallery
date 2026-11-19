package bevelgear_test

import (
	"github.com/lestrrat-3d/decad"
)

func meshEuler(mesh *decad.Mesh) int {
	edges := make(map[[2]int]struct{})
	triangles := mesh.Triangles()
	for _, triangle := range triangles {
		for i := range 3 {
			a, b := triangle[i], triangle[(i+1)%3]
			if a > b {
				a, b = b, a
			}
			edges[[2]int{a, b}] = struct{}{}
		}
	}
	return len(mesh.Vertices()) - len(edges) + len(triangles)
}

func meshComponents(mesh *decad.Mesh) int {
	parent := make([]int, len(mesh.Vertices()))
	for i := range parent {
		parent[i] = i
	}
	root := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for _, triangle := range mesh.Triangles() {
		first := root(triangle[0])
		parent[root(triangle[1])] = first
		parent[root(triangle[2])] = first
	}
	components := make(map[int]struct{})
	for i := range parent {
		components[root(i)] = struct{}{}
	}
	return len(components)
}
