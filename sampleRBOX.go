package main

import (
	"bytes"
	"fmt"
	"os"
)

func gen_RBox1() {

	os.Remove("RBox1")
	os.Mkdir("RBox1", 0777)

	ADP := NewDatacenter("ADP")

	RBLH1 := NewDatacenter("RB_LH_1")
	RBLH2 := NewDatacenter("RB_LH_2")
	RBLH3 := NewDatacenter("RB_LH_3")
	RBLH4 := NewDatacenter("RB_LH_4")
	RBLHB := NewDatacenter("RB_LH_B")

	RBAF1 := NewDatacenter("RB_AF_1")
	RBAF2 := NewDatacenter("RB_AF_2")
	RBAFB := NewDatacenter("RB_AF_B")

	def_LH := DefRBox("LH")
	def_AF := DefRBox("AF")
	defB_LH := DefRBoxB("LH")
	defB_AF := DefRBoxB("AF")
	defM := DefRBoxM()

	defs := []ClusterGroupDef{}
	defs = append(defs, def_LH)
	defs = append(defs, def_AF)
	defs = append(defs, defB_LH)
	defs = append(defs, defB_AF)
	defs = append(defs, defM)

	ToFile(ClusterGroupDefBluePrint{defs}, "RBox1/couchbase")

	ADP.AddClusterGroupDef(defM)

	RBLHB.AddClusterGroupDef(defB_LH)
	RBAFB.AddClusterGroupDef(defB_AF)

	RBLH1.AddClusterGroupDef(def_LH)
	RBLH2.AddClusterGroupDef(def_LH)
	RBLH3.AddClusterGroupDef(def_LH)
	RBLH4.AddClusterGroupDef(def_LH)

	RBAF1.AddClusterGroupDef(def_AF)
	RBAF2.AddClusterGroupDef(def_AF)

	DCs := []Datacenter{ADP, RBAFB, RBAF1, RBAF2, RBLHB, RBLH1, RBLH2, RBLH3, RBLH4}

	var buf bytes.Buffer

	for _, d := range DCs {
		d.Dot(&buf)
	}

	xdcrdefs := []XDCRDef{}
	xdcrdefs = append(xdcrdefs, DefXDCR_M())
	xdcrdefs = append(xdcrdefs, DefXDCR_B())

	ToFile(XDCRDefBluePrint{xdcrdefs}, "RBox1/XDCR")

	for _, xdcr := range xdcrdefs {
		for _, x := range NewXDCR(xdcr, DCs) {
			x.Dot(&buf)
		}
	}

	fmt.Printf("digraph { \n%s\n}\n", buf.String())
}

func DefRBox(pk string) ClusterGroupDef {
	return ClusterGroupDef{
		Name:       "CG",
		PeakTokens: []string{pk},
		ClusterDefs: []ClusterDef{
			{Name: "CBBOX", Instances: []string{""},
				Buckets: []Bucket{
					{Name: "Rbox", Labels: Labels{"Role": "Rbox", "Type": "Child"}},
					{Name: "SBox", Labels: Labels{"Role": "SBox", "Type": "Child"}},
					{Name: "Stat", Labels: Labels{"Role": "Stat", "Type": "Child"}},
				},
			}},
	}
}

func DefRBoxB(pk string) ClusterGroupDef {
	return ClusterGroupDef{
		Name:       "CG",
		PeakTokens: []string{pk},
		ClusterDefs: []ClusterDef{
			{Name: "CBBOX", Instances: []string{""},
				Buckets: []Bucket{
					{Name: "Rbox", Labels: Labels{"Role": "Rbox", "Type": "BCast"}},
					{Name: "SBox", Labels: Labels{"Role": "SBox", "Type": "BCast"}},
					{Name: "Stat", Labels: Labels{"Role": "Stat", "Type": "BCast"}},
				},
			}},
	}
}

func DefRBoxM() ClusterGroupDef {
	return ClusterGroupDef{
		Name:       "CG",
		PeakTokens: []string{""},
		ClusterDefs: []ClusterDef{
			{Name: "CBBOX", Instances: []string{""},
				Buckets: []Bucket{
					{Name: "Rbox", Labels: Labels{"Role": "Rbox", "Type": "MCast"}},
					{Name: "SBox", Labels: Labels{"Role": "SBox", "Type": "MCast"}},
					{Name: "Stat", Labels: Labels{"Role": "Stat", "Type": "MCast"}},
				},
			}},
	}
}

func DefXDCR_M() XDCRDef {
	return XDCRDef{
		Rule:          "custom",
		Bidirectional: false,
		Source:        Selector{"Type": "MCast"},
		Destination:   Selector{"Type": "BCast"},
		GroupOn:       []string{"Role"},
		Args:          []string{},
		Color:         "red",
	}
}

func DefXDCR_B() XDCRDef {
	return XDCRDef{
		Rule:          "custom",
		Bidirectional: false,
		Source:        Selector{"Type": "BCast"},
		Destination:   Selector{"Type": "Child"},
		GroupOn:       []string{"Role", "ClusterGroup"},
		Args:          []string{},
		Color:         "blue",
	}
}
