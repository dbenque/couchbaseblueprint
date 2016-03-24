package main

import (
	"bytes"
	"fmt"
)

func main() {

	DC1 := NewDatacenter("DC1")
	DC2 := NewDatacenter("DC2")
	DC1.AddClusterGroupDef(Def1())
	DC2.AddClusterGroupDef(Def1())

	var buf bytes.Buffer
	DC1.Dot(&buf)
	DC2.Dot(&buf)

	for _, x := range NewXDCR(Def1XDCR_Hyatt(), []Datacenter{DC1, DC2}) {
		x.Dot(&buf)
	}
	for _, x := range NewXDCR(Def1XDCR_HyattR(), []Datacenter{DC1, DC2}) {
		x.Dot(&buf)
	}

	for _, x := range NewXDCR(Def1XDCR_Campanile(), []Datacenter{DC1, DC2}) {
		x.Dot(&buf)
	}

	fmt.Printf("digraph { \n%s\n}\n", buf.String())

	//j, _ := json.Marshal(DC1)
	//fmt.Printf("\n\n%s\n", string(j))
}
