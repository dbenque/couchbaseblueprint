package main

import (
	"fmt"
	"io"
)

var DotLevels = map[string]Bucket{}

func (c *Cluster) Dot(w io.Writer) {

	fmt.Fprintf(w, "subgraph cluster_%s {\n", c.Path())
	fmt.Fprintf(w, "label=\"%s %s\";\n", c.Name, c.Instance)

	for _, b := range c.Buckets {
		fmt.Fprintf(w, "%s[label=%s];\n", b.Path(), b.Name)
		if l, ok := b.Labels["Level"]; ok {
			if other, found := DotLevels[l]; found {
				fmt.Fprintf(w, "{rank=same; %s %s}\n", b.Path(), other.Path())
			} else {
				DotLevels[l] = b
			}
		}
	}
	fmt.Fprintf(w, "}\n")
}

func (cg *ClusterGroup) Dot(w io.Writer) {

	fmt.Fprintf(w, "subgraph cluster_%s {\n", cg.Path())
	fmt.Fprintf(w, "label=\"%s %s\";\n", cg.Name, cg.PeakToken)

	for _, c := range cg.Clusters {
		c.Dot(w)
	}

	fmt.Fprintf(w, "}\n")

}

func (dc *Datacenter) Dot(w io.Writer) {

	fmt.Fprintf(w, "subgraph cluster_%s {\n", dc.Name)
	fmt.Fprintf(w, "label=\"%s\";\n", dc.Name)

	for _, cg := range dc.ClusterGroups {
		cg.Dot(w)
	}

	fmt.Fprintf(w, "}\n")

}

func (x *XDCR) Dot(w io.Writer) {
	fmt.Fprintf(w, "%s -> %s [color=%s];\n", x.Source.Path(), x.Destination.Path(), x.Color)
}

func (c *Cluster) Path() string {
	return c.Labels["Datacenter"] + "_" + c.Labels["ClusterGroup"] + "_" + c.Name + "_" + c.Instance
}

func (cg *ClusterGroup) Path() string {
	return cg.Labels["Datacenter"] + "_" + cg.Name + "_" + cg.PeakToken
}

func (b *Bucket) Path() string {
	return b.Labels["Datacenter"] + "_" + b.Labels["ClusterGroup"] + "_" + b.Labels["Cluster"] + "_" + b.Name
}
