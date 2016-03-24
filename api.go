package main

import "strings"

type Labels map[string]string
type Selector map[string]string

type LabelMatcher interface {
	Match(s Selector) bool
}

type Datacenter struct {
	Name          string
	ClusterGroups []ClusterGroup
}

func (dc *Datacenter) AddClusterGroup(cg []ClusterGroup) {
	dc.ClusterGroups = append(dc.ClusterGroups, cg...)
}

func (dc *Datacenter) AddClusterGroupDef(cgdef ClusterGroupDef) {
	cg := NewClusterGroups(dc.Name, cgdef)
	dc.ClusterGroups = append(dc.ClusterGroups, cg...)
}

func (dc *Datacenter) GetBuckets() []Bucket {
	result := []Bucket{}
	for _, cg := range dc.ClusterGroups {
		for _, c := range cg.Clusters {
			for _, b := range c.Buckets {
				result = append(result, b)
			}
		}
	}
	return result
}

type ClusterGroupDef struct {
	Name        string
	PeakTokens  []string
	Labels      Labels
	ClusterDefs []ClusterDef
}

type ClusterGroup struct {
	Name      string
	PeakToken string
	Labels    Labels
	Clusters  []Cluster
}

type ClusterDef struct {
	Name      string
	Instances []string
	Labels    Labels
	Buckets   []Bucket
}

type Cluster struct {
	Name     string
	Instance string
	Labels   Labels
	Buckets  []Bucket
}

type Bucket struct {
	Name              string
	RamQuota          int
	CBReplicateNumber int
	Labels            Labels
}

type XDCRRule string

const (
	TreeRule  XDCRRule = "tree"
	RingRule  XDCRRule = "ring"
	ChainRule XDCRRule = "chain"
)

type XDCRDef struct {
	Rule               XDCRRule
	Bidirectional      bool
	Source             Selector
	SourceExclude      Selector
	Destination        Selector
	DestinationExclude Selector
	GroupOn            []string
	Args               []string
	Color              string
}

type XDCR struct {
	Source      Bucket
	Destination Bucket
	Color       string
}

func (b *Bucket) Match(s Selector) bool {
	if s == nil {
		return false
	}
	for k, v := range s {
		vv, ok := b.Labels[k]
		if !ok || vv != v {
			return false
		}
	}
	return true
}

func (b *Bucket) GroupHash(groupOn []string) string {
	f := "#"
	if groupOn == nil {
		return f
	}

	for _, k := range groupOn {
		v := b.Labels[k]
		f = f + k + ":" + v + "#"
	}
	return f
}

func (l Labels) Copy() Labels {
	result := Labels{}
	for k, v := range l {
		result[k] = v
	}
	return result
}

type BucketByPath []Bucket

func (a BucketByPath) Len() int           { return len(a) }
func (a BucketByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BucketByPath) Less(i, j int) bool { return strings.Compare(a[i].Path(), a[j].Path()) < 0 }
