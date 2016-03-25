package main

import (
	"bytes"
	"fmt"
	"os"
)

func gen_sample1() {

	os.Remove("sample1")
	os.Mkdir("sample1", 0777)

	DC1 := NewDatacenter("DC1")
	DC2 := NewDatacenter("DC2")
	def1 := Def1()

	ToFile(def1, "sample1/couchbase.json")

	DC1.AddClusterGroupDef(def1)
	DC2.AddClusterGroupDef(def1)

	var buf bytes.Buffer
	DC1.Dot(&buf)
	DC2.Dot(&buf)

	xdcrdefs := []XDCRDef{}
	xdcrdefs = append(xdcrdefs, Def1XDCR_Hyatt())
	xdcrdefs = append(xdcrdefs, Def1XDCR_HyattR())
	xdcrdefs = append(xdcrdefs, Def1XDCR_Campanile())

	ToFile(xdcrdefs, "sample1/XDCR.json")

	for _, xdcr := range xdcrdefs {
		for _, x := range NewXDCR(xdcr, []Datacenter{DC1, DC2}) {
			x.Dot(&buf)
		}
	}

	fmt.Printf("digraph { \n%s\n}\n", buf.String())
}

func Def1() ClusterGroupDef {
	return ClusterGroupDef{
		Name:       "CG",
		PeakTokens: []string{"PK1", "PK2"},
		ClusterDefs: []ClusterDef{
			{Name: "Booking", Instances: []string{"A", "B"},
				Buckets: []Bucket{
					{Name: "Hyatt", Labels: Labels{"Company": "Hyatt"}},
					{Name: "HyattR", Labels: Labels{"Company": "Hyatt", "ReadOnly": "true"}},
					{Name: "Campanile", Labels: Labels{"Company": "Campanile"}},
				},
			}},
	}
}

func Def1XDCR_Hyatt() XDCRDef {
	return XDCRDef{
		Rule:          "ring",
		Bidirectional: false,
		Source:        Selector{"Company": "Hyatt"},
		SourceExclude: Selector{"ReadOnly": "true"},
		GroupOn:       []string{},
		Args:          []string{},
		Color:         "red",
	}
}

func Def1XDCR_HyattR() XDCRDef {
	return XDCRDef{
		Rule:          "custom",
		Bidirectional: true,
		Source:        Selector{"Company": "Hyatt"},
		SourceExclude: Selector{"ReadOnly": "true"},
		Destination:   Selector{"Company": "Hyatt", "ReadOnly": "true"},
		GroupOn:       []string{"Cluster", "ClusterGroup", "Datacenter"},
		Args:          []string{},
		Color:         "blue",
	}
}

func Def1XDCR_Campanile() XDCRDef {
	return XDCRDef{
		Rule:          "ring",
		Bidirectional: false,
		Source:        Selector{"Company": "Campanile"},
		GroupOn:       []string{"Datacenter", "ClusterGroup"},
		Args:          []string{},
		Color:         "green",
	}
}
