package example

import (
	"bytes"
	"couchbasebp/api"
	"couchbasebp/utils"
	"fmt"
	"os"
)

func Gen_RBox1() {

	os.Remove("RBox1")
	os.Mkdir("RBox1", 0777)

	ADP := api.NewDatacenter("ADP")

	RBLH1 := api.NewDatacenter("RB_LH_1")
	RBLH2 := api.NewDatacenter("RB_LH_2")
	RBLH3 := api.NewDatacenter("RB_LH_3")
	RBLH4 := api.NewDatacenter("RB_LH_4")
	RBLHB := api.NewDatacenter("RB_LH_B")

	RBAF1 := api.NewDatacenter("RB_AF_1")
	RBAF2 := api.NewDatacenter("RB_AF_2")
	RBAFB := api.NewDatacenter("RB_AF_B")

	def_LH := defRBox("LH")
	def_AF := defRBox("AF")
	defB_LH := defRBoxB("LH")
	defB_AF := defRBoxB("AF")
	defM := defRBoxM()

	defs := []api.ClusterGroupDef{}
	defs = append(defs, def_LH)
	defs = append(defs, def_AF)
	defs = append(defs, defB_LH)
	defs = append(defs, defB_AF)
	defs = append(defs, defM)

	utils.ToFile(api.ClusterGroupDefBluePrint{defs}, "RBox1/couchbase")

	ADP.AddClusterGroupDef(defM)

	RBLHB.AddClusterGroupDef(defB_LH)
	RBAFB.AddClusterGroupDef(defB_AF)

	RBLH1.AddClusterGroupDef(def_LH)
	RBLH2.AddClusterGroupDef(def_LH)
	RBLH3.AddClusterGroupDef(def_LH)
	RBLH4.AddClusterGroupDef(def_LH)

	RBAF1.AddClusterGroupDef(def_AF)
	RBAF2.AddClusterGroupDef(def_AF)

	DCs := []api.Datacenter{ADP, RBAFB, RBAF1, RBAF2, RBLHB, RBLH1, RBLH2, RBLH3, RBLH4}

	var buf bytes.Buffer

	for _, d := range DCs {
		d.Dot(&buf)
	}

	xdcrdefs := []api.XDCRDef{}
	xdcrdefs = append(xdcrdefs, defXDCR_M())
	xdcrdefs = append(xdcrdefs, defXDCR_B())

	utils.ToFile(api.XDCRDefBluePrint{xdcrdefs}, "RBox1/XDCR")

	for _, xdcr := range xdcrdefs {
		for _, x := range api.NewXDCR(xdcr, DCs) {
			x.Dot(&buf)
		}
	}

	fmt.Printf("digraph { \n%s\n}\n", buf.String())
}

func defRBox(pk string) api.ClusterGroupDef {
	return api.ClusterGroupDef{
		Name:       "CG",
		PeakTokens: []string{pk},
		ClusterDefs: []api.ClusterDef{
			{Name: "CBBOX", Instances: []string{""},
				Buckets: []api.Bucket{
					{Name: "Rbox", Labels: api.Labels{"Role": "Rbox", "Type": "Child"}},
					{Name: "SBox", Labels: api.Labels{"Role": "SBox", "Type": "Child"}},
					{Name: "Stat", Labels: api.Labels{"Role": "Stat", "Type": "Child"}},
				},
			}},
	}
}

func defRBoxB(pk string) api.ClusterGroupDef {
	return api.ClusterGroupDef{
		Name:       "CG",
		PeakTokens: []string{pk},
		ClusterDefs: []api.ClusterDef{
			{Name: "CBBOX", Instances: []string{""},
				Buckets: []api.Bucket{
					{Name: "Rbox", Labels: api.Labels{"Role": "Rbox", "Type": "BCast"}},
					{Name: "SBox", Labels: api.Labels{"Role": "SBox", "Type": "BCast"}},
					{Name: "Stat", Labels: api.Labels{"Role": "Stat", "Type": "BCast"}},
				},
			}},
	}
}

func defRBoxM() api.ClusterGroupDef {
	return api.ClusterGroupDef{
		Name:       "CG",
		PeakTokens: []string{""},
		ClusterDefs: []api.ClusterDef{
			{Name: "CBBOX", Instances: []string{""},
				Buckets: []api.Bucket{
					{Name: "Rbox", Labels: api.Labels{"Role": "Rbox", "Type": "MCast"}},
					{Name: "SBox", Labels: api.Labels{"Role": "SBox", "Type": "MCast"}},
					{Name: "Stat", Labels: api.Labels{"Role": "Stat", "Type": "MCast"}},
				},
			}},
	}
}

func defXDCR_M() api.XDCRDef {
	return api.XDCRDef{
		Rule:          "custom",
		Bidirectional: false,
		Source:        api.Selector{"Type": "MCast"},
		Destination:   api.Selector{"Type": "BCast"},
		GroupOn:       []string{"Role"},
		Args:          []string{},
		Color:         "red",
	}
}

func defXDCR_B() api.XDCRDef {
	return api.XDCRDef{
		Rule:          "custom",
		Bidirectional: false,
		Source:        api.Selector{"Type": "BCast"},
		Destination:   api.Selector{"Type": "Child"},
		GroupOn:       []string{"Role", "ClusterGroup"},
		Args:          []string{},
		Color:         "blue",
	}
}
