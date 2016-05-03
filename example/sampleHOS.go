package example

import (
	"bytes"
	"couchbasebp/api"
	"couchbasebp/utils"
	"fmt"
	"os"
)

func Gen_hos1() {

	os.Remove("hos1")
	os.Mkdir("hos1", 0777)

	DC1 := api.NewDatacenter("DC1")
	DC2 := api.NewDatacenter("DC2")
	def1 := def1()

	utils.ToFile(api.ClusterGroupDefBluePrint{[]api.ClusterGroupDef{def1}}, "hos1/couchbase")

	DC1.AddClusterGroupDef(def1)
	DC2.AddClusterGroupDef(def1)

	var buf bytes.Buffer
	DC1.Dot(&buf)
	DC2.Dot(&buf)

	xdcrdefs := []api.XDCRDef{}
	xdcrdefs = append(xdcrdefs, def1XDCRhyatt())
	xdcrdefs = append(xdcrdefs, def1XDCRhyattR())
	xdcrdefs = append(xdcrdefs, def1XDCRcampanile())

	utils.ToFile(api.XDCRDefBluePrint{xdcrdefs}, "hos1/XDCR")

	for _, xdcr := range xdcrdefs {
		for _, x := range api.NewXDCR(xdcr, []api.Datacenter{DC1, DC2}) {
			x.Dot(&buf)
		}
	}

	fmt.Printf("digraph { \n%s\n}\n", buf.String())
}

func def1() api.ClusterGroupDef {
	return api.ClusterGroupDef{
		Name:       "CG",
		PeakTokens: []string{"PK1", "PK2"},
		ClusterDefs: []api.ClusterDef{
			{Name: "Booking", Instances: []string{"A", "B"},
				Buckets: []api.Bucket{
					{Name: "Hyatt", Labels: api.Labels{"Company": "Hyatt"}},
					{Name: "HyattR", Labels: api.Labels{"Company": "Hyatt", "ReadOnly": "true"}},
					{Name: "Campanile", Labels: api.Labels{"Company": "Campanile"}},
				},
			}},
	}
}

func def1XDCRhyatt() api.XDCRDef {
	return api.XDCRDef{
		Rule:          "ring",
		Bidirectional: false,
		Source:        api.Selector{"Company": "Hyatt"},
		SourceExclude: api.Selector{"ReadOnly": "true"},
		GroupOn:       []string{},
		Args:          []string{},
		Color:         "red",
	}
}

func def1XDCRhyattR() api.XDCRDef {
	return api.XDCRDef{
		Rule:          "custom",
		Bidirectional: true,
		Source:        api.Selector{"Company": "Hyatt"},
		SourceExclude: api.Selector{"ReadOnly": "true"},
		Destination:   api.Selector{"Company": "Hyatt", "ReadOnly": "true"},
		GroupOn:       []string{"Cluster", "ClusterGroup", "Datacenter"},
		Args:          []string{},
		Color:         "blue",
	}
}

func def1XDCRcampanile() api.XDCRDef {
	return api.XDCRDef{
		Rule:          "ring",
		Bidirectional: false,
		Source:        api.Selector{"Company": "Campanile"},
		GroupOn:       []string{"Datacenter", "ClusterGroup"},
		Args:          []string{},
		Color:         "green",
	}
}
