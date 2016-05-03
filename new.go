package main

import "sort"

func NewDatacenter(name string) Datacenter {
	return Datacenter{Name: name, ClusterGroups: []ClusterGroup{}}
}

func NewClusterGroups(dc string, def ClusterGroupDef) []ClusterGroup {
	results := []ClusterGroup{}
	if def.PeakTokens == nil {
		return results
	}

	if def.Labels == nil {
		def.Labels = Labels{}
	}
	def.Labels["Datacenter"] = dc

	for _, p := range def.PeakTokens {
		cg := ClusterGroup{Name: def.Name, PeakToken: p, Labels: def.Labels.Copy()}
		cg.Clusters = []Cluster{}
		lb := cg.Labels.Copy()
		lb["ClusterGroup"] = cg.Name + "_" + cg.PeakToken
		for _, cdef := range def.ClusterDefs {
			cg.Clusters = append(cg.Clusters, NewClusters(lb, cdef)...)
		}
		results = append(results, cg)
	}
	return results
}

func NewClusters(lb Labels, def ClusterDef) []Cluster {
	results := []Cluster{}
	if def.Instances == nil {
		return results
	}

	for _, i := range def.Instances {
		if def.Labels == nil {
			def.Labels = Labels{}
		}

		c := Cluster{Name: def.Name, Instance: i, Labels: def.Labels.Copy()}
		c.Labels["Datacenter"], _ = lb["Datacenter"]
		c.Labels["ClusterGroup"], _ = lb["ClusterGroup"]

		for _, bdef := range def.Buckets {
			bdef.Labels = bdef.Labels.Copy()
			bdef.Labels["Cluster"] = c.Name + "_" + i
			bdef.Labels["Datacenter"], _ = lb["Datacenter"]
			bdef.Labels["ClusterGroup"], _ = lb["ClusterGroup"]
			bdef.Labels["name"] = bdef.Name
			c.Buckets = append(c.Buckets, bdef)
		}
		results = append(results, c)
	}
	return results
}

func NewXDCR(def XDCRDef, dcs []Datacenter) []XDCR {

	allBuckets := []Bucket{}
	for _, dc := range dcs {
		allBuckets = append(allBuckets, dc.GetBuckets()...)
	}

	filteredBuckets := map[string][2]BucketByPath{}
	for _, b := range allBuckets {
		g := b.GroupHash(def.GroupOn)
		if b.Match(def.Source) && !b.Match(def.SourceExclude) {
			sl := filteredBuckets[g]
			sources := sl[0]
			if sources == nil {
				sources = BucketByPath{}
			}
			sl[0] = append(sources, b)
			filteredBuckets[g] = sl
		}
		if b.Match(def.Destination) && !b.Match(def.DestinationExclude) {
			sl := filteredBuckets[g]
			destinations := sl[1]
			if destinations == nil {
				destinations = BucketByPath{}
			}
			sl[1] = append(destinations, b)
			filteredBuckets[g] = sl
		}
	}

	switch def.Rule {
	case RingRule:
		{
			result := []XDCR{}
			// loop over each group
			for _, fb := range filteredBuckets {
				sources := fb[0]
				sort.Sort(sources)
				result = append(result, buildRing(sources, def)...)
			}
			return result
		}
	case CustomRule:
		{
			result := []XDCR{}
			// loop over each group
			for _, fb := range filteredBuckets {
				sources := fb[0]
				destinations := fb[1]
				sort.Sort(sources)
				sort.Sort(destinations)
				result = append(result, buildCustom(sources, destinations, def)...)
			}
			return result
		}
	case UptreeRule, TreeRule:
		{
			result := []XDCR{}
			for _, fb := range filteredBuckets {
				sources := fb[0]
				result = append(result, buildTree(sources, def, def.Rule == UptreeRule)...)
			}
			return result
		}
	}
	return []XDCR{}
}

func buildTree(buckets BucketByPath, def XDCRDef, up bool) []XDCR {
	result := []XDCR{}

	byLevel := map[string]BucketByPath{}

	// create indexing by level
	for _, b := range buckets {
		level := b.Labels["Level"]
		bucketsOfLevel, ok := byLevel[level]
		if !ok {
			bucketsOfLevel = BucketByPath{}
		}
		bucketsOfLevel = append(bucketsOfLevel, b)
		byLevel[level] = bucketsOfLevel
	}

	// retrieve all level
	levels := []string{}
	for l := range byLevel {
		levels = append(levels, l)
	}
	sort.Strings(levels)

	// build tree
	for i := 0; i < len(levels)-1; i++ {
		sources := byLevel[levels[i]]
		destinations := byLevel[levels[i+1]]

		for j, d := range destinations {
			s := sources[j%len(sources)]

			if up {
				s, d = d, s
			}

			result = append(result, XDCR{Source: s, Destination: d, Color: def.Color})
			if def.Bidirectional {
				result = append(result, XDCR{Source: d, Destination: s, Color: def.Color})
			}
		}
	}

	return result
}

func buildCustom(sources, destinations BucketByPath, def XDCRDef) []XDCR {
	result := []XDCR{}
	if len(sources) < 1 || len(destinations) < 1 {
		return result
	}

	for _, s := range sources {
		for _, d := range destinations {
			result = append(result, XDCR{Source: s, Destination: d, Color: def.Color})
			if def.Bidirectional {
				result = append(result, XDCR{Source: d, Destination: s, Color: def.Color})
			}
		}
	}
	return result
}

func buildRing(buckets BucketByPath, def XDCRDef) []XDCR {
	result := []XDCR{}
	if len(buckets) < 2 {
		return result
	}

	for i := 0; i < len(buckets)-1; i++ {
		result = append(result, XDCR{Source: buckets[i], Destination: buckets[i+1], Color: def.Color})
		if def.Bidirectional && len(buckets) > 2 {
			result = append(result, XDCR{Destination: buckets[i], Source: buckets[i+1], Color: def.Color})
		}
	}
	result = append(result, XDCR{Source: buckets[len(buckets)-1], Destination: buckets[0], Color: def.Color})
	if def.Bidirectional && len(buckets) > 2 {
		result = append(result, XDCR{Destination: buckets[len(buckets)-1], Source: buckets[0], Color: def.Color})
	}
	return result
}
