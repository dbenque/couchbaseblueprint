package api

import "strings"

//Labels help to decorate entities
type Labels map[string]string

//Selector helps to filter in/out an entity based on its lables
type Selector map[string]string

//LabelMatcher interface to play with Selector. Most probably the entities imilementing this interface will compose Labels.
type LabelMatcher interface {
	Match(s Selector) bool
}

//EnvData simulate Environment file
type EnvData struct {
	Replacements Labels `yaml:"replacements" json:"replacements"`
}

//Datacenter named (unique) Datacenter instance.
type Datacenter struct {
	Name          string
	Version       string
	ClusterGroups []ClusterGroup `diff:"composition"`
}

//AddClusterGroup copy clustergroup instances in the Datacenter
func (dc *Datacenter) AddClusterGroup(cg []ClusterGroup) {
	dc.ClusterGroups = append(dc.ClusterGroups, cg...)
}

//AddClusterGroupDef create clustergroup instances in the Datacenter, based on the definition
func (dc *Datacenter) AddClusterGroupDef(cgdef ClusterGroupDef) {
	cg := NewClusterGroups(dc.Name, cgdef)
	dc.ClusterGroups = append(dc.ClusterGroups, cg...)
}

//GetBuckets retrieve all buckets of the datacenter
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

//ClusterGroupDefBluePrint blueprint listing one or several definitions of topology starting at ClusterGroup level.
type ClusterGroupDefBluePrint struct {
	ClusterGroups []ClusterGroupDef
}

//ClusterGroupDef definition of the topology at ClusterGroup level.
type ClusterGroupDef struct {
	Name        string       `yaml:"name" json:"name"`
	PeakTokens  []string     `yaml:"peakToken" json:"peakToken"`
	Labels      Labels       `yaml:"labels,omitempty" json:"labels,omitempty"`
	ClusterDefs []ClusterDef `yaml:"clusters" json:"clusters"`
}

//ClusterGroup ClusterGroup instance
type ClusterGroup struct {
	Name      string
	PeakToken string
	Labels    Labels    `diff:"value"`
	Clusters  []Cluster `diff:"composition"`
}

//ClusterDef definition of the topology at Cluster level.
type ClusterDef struct {
	Name      string   `yaml:"name" json:"name"`
	Instances []string `yaml:"instances" json:"instances"`
	Labels    Labels   `yaml:"labels,omitempty" json:"labels,omitempty" diff:"value"`
	Buckets   []Bucket `yaml:"buckets" json:"buckets" diff:"composition"`
}

//Cluster Cluster instance
type Cluster struct {
	Name     string
	Instance string
	Labels   Labels   `diff:"value"`
	Buckets  []Bucket `diff:"composition"`
}

//Bucket instance
type Bucket struct {
	Name              string `yaml:"name" json:"name"`
	RAMQuota          int    `yaml:"ramQuota" json:"ramQuota" diff:"value"`
	CBReplicateNumber int    `yaml:"cbReplicatNumber" json:"cbReplicatNumber" diff:"value"`
	Labels            Labels `yaml:"labels,omitempty" json:"labels,omitempty" diff:"value"`
}

//XDCRRule define type of rules for XDCR
type XDCRRule string

//Type of rules
const (
	UptreeRule XDCRRule = "uptree"
	TreeRule   XDCRRule = "tree"
	RingRule   XDCRRule = "ring"
	CustomRule XDCRRule = "custom"
)

//XDCRDefBluePrint blueprint listing one or several XDCR Rule definitions.
type XDCRDefBluePrint struct {
	XDCRDefs []XDCRDef
}

//XDCRDef definition of an XDCR Rule
type XDCRDef struct {
	Rule               XDCRRule `yaml:"rule" json:"rule"`
	Bidirectional      bool     `yaml:"bidirectional" json:"bidirectional"`
	Source             Selector `yaml:"source" json:"source"`
	SourceExclude      Selector `yaml:"sourceExclude,omitempty" json:"sourceExclude,omitempty"`
	Destination        Selector `yaml:"destination,omitempty" json:"destination,omitempty"`
	DestinationExclude Selector `yaml:"destinationExclude,omitempty" json:"destinationExclude,omitempty"`
	GroupOn            []string `yaml:"groupOn,omitempty" json:"groupOn,omitempty"`
	Args               []string `yaml:"args" json:"args"`
	ArgsXCluster       []string `yaml:"argsXCluster,omitempty" json:"argsXCluster,omitempty"`
	ArgsXClusterGroup  []string `yaml:"argsXClusterGroup,omitempty" json:"argsXClusterGroup,omitempty"`
	ArgsXDatacenter    []string `yaml:"argsXDatacenter,omitempty" json:"argsXDatacenter,omitempty"`
	Color              string   `yaml:"color" json:"color"`
}

//XDCR Instance of an XDCR link
type XDCR struct {
	Source      Bucket
	Destination Bucket
	Color       string
}

//Match Checks if the Bucket matches the selector
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

//GroupHash compute the group key for that bucket based on a grounpOn key set.
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

//Copy create a new clone of the Lables
func (l Labels) Copy() Labels {
	result := Labels{}
	for k, v := range l {
		result[k] = v
	}
	return result
}

//BucketByPath helps for sorting bucket by their path
type BucketByPath []Bucket

//Len to implement sort.Sort
func (a BucketByPath) Len() int { return len(a) }

//Swap to implement sort.Sort
func (a BucketByPath) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

//Less to implement sort.Sort
func (a BucketByPath) Less(i, j int) bool { return strings.Compare(a[i].Path(), a[j].Path()) < 0 }

//PathIdentifier object implementing this interface are uniquely indentified by their path
type PathIdentifier interface {
	Path() string
}

//Path return the Path to that Cluster (identifier)
func (c Cluster) Path() string {
	return c.Labels["Datacenter"] + "_" + c.Labels["ClusterGroup"] + "_" + c.Name + "_" + c.Instance
}

//Path return the Path to that ClusterGroup (identifier)
func (cg ClusterGroup) Path() string {
	return cg.Labels["Datacenter"] + "_" + cg.Name + "_" + cg.PeakToken
}

//Path return the Path to that Bucket (identifier)
func (b Bucket) Path() string {
	return b.Labels["Datacenter"] + "_" + b.Labels["ClusterGroup"] + "_" + b.Labels["Cluster"] + "_" + b.Name
}
