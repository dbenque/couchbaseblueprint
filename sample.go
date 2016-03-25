package main

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
		GroupOn:       []string{"ClusterGroup", "Datacenter"},
		Args:          []string{},
		Color:         "green",
	}
}
