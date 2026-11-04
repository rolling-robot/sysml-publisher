package main

import (
	"fmt"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

func main() {
	c, err := opensysml.New()
	fmt.Println(c != nil, err)
	a, err := asciidoctor.New()
	fmt.Println(a != nil, err)
}
