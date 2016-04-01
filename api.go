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

type ClusterGroupDefBluePrint struct {
	ClusterGroups []ClusterGroupDef
}

type ClusterGroupDef struct {
	Name        string       `yaml:"name" json:"name"`
	PeakTokens  []string     `yaml:"peakToken" json:"peakToken"`
	Labels      Labels       `yaml:"labels,omitempty" json:"labels,omitempty"`
	ClusterDefs []ClusterDef `yaml:"clusters" json:"clusters"`
}

type ClusterGroup struct {
	Name      string
	PeakToken string
	Labels    Labels
	Clusters  []Cluster
}

type ClusterDef struct {
	Name      string   `yaml:"name" json:"name"`
	Instances []string `yaml:"instances" json:"instances"`
	Labels    Labels   `yaml:"labels,omitempty" json:"labels,omitempty"`
	Buckets   []Bucket `yaml:"buckets" json:"buckets"`
}

type Cluster struct {
	Name     string
	Instance string
	Labels   Labels
	Buckets  []Bucket
}

type Bucket struct {
	Name              string `yaml:"name" json:"name"`
	RamQuota          int    `yaml:"ramQuota" json:"ramQuota"`
	CBReplicateNumber int    `yaml:"cbReplicatNumber" json:"cbReplicatNumber"`
	Labels            Labels `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type XDCRRule string

const (
	UptreeRule XDCRRule = "uptree"
	TreeRule   XDCRRule = "tree"
	RingRule   XDCRRule = "ring"
	ChainRule  XDCRRule = "chain"
	CustomRule XDCRRule = "custom"
)

type XDCRDefBluePrint struct {
	XDCRDefs []XDCRDef
}

type XDCRDef struct {
	Rule               XDCRRule `yaml:"rule" json:"rule"`
	Bidirectional      bool     `yaml:"bidirectional" json:"bidirectional"`
	Source             Selector `yaml:"source" json:"source"`
	SourceExclude      Selector `yaml:"sourceExclude,omitempty" json:"sourceExclude,omitempty"`
	Destination        Selector `yaml:"destination,omitempty" json:"destination,omitempty"`
	DestinationExclude Selector `yaml:"destinationExclude,omitempty" json:"destinationExclude,omitempty"`
	GroupOn            []string `yaml:"groupOn,omitempty" json:"groupOn,omitempty"`
	Args               []string `yaml:"args" json:"args"`
	Color              string   `yaml:"color" json:"color"`
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
		if !ok {
			return false
		}
		found := false
		for _, vvor := range strings.Split(v, "|") {
			found = (found || (vvor == vv))
		}
		if !found {
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
