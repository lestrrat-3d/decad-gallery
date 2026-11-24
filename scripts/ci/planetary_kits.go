//go:build ignore

package main

import (
	"fmt"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
)

func main() {
	kits := planetarygearbox.SupportedCombinations()
	for _, kit := range kits {
		if kit.Name == "planet605-shaft605" {
			continue
		}
		fmt.Println(kit.Name)
	}
}
