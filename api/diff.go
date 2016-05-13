package api

import (
	"fmt"
	"reflect"
)

type diffValues struct {
	Current  interface{}
	Proposed interface{}
}

type diffComposition struct {
	Modified []diff
	Deleted  []interface{}
	New      []interface{}
}

func (d *diff) Empty() bool {
	return len(d.Composition) == 0 && len(d.Param) == 0
}

type diff struct {
	Path        string
	Param       map[string]diffValues
	Composition map[string]diffComposition
}

func newDiffComposition() diffComposition {
	return diffComposition{[]diff{}, []interface{}{}, []interface{}{}}
}

func GetDiff(current, proposed PathIdentifier) (*diff, error) {
	if current == nil || proposed == nil {
		return nil, fmt.Errorf("Nil bucket as input")
	}

	if current.Path() != proposed.Path() {
		return nil, fmt.Errorf("diff on bucket not under same path")
	}

	d := diff{Path: current.Path(), Param: map[string]diffValues{}, Composition: map[string]diffComposition{}}

	vc := reflect.ValueOf(current)
	vp := reflect.ValueOf(proposed)
	for i := 0; i < vc.NumField(); i++ {

		vcurrent := vc.Field(i).Interface()
		vproposed := vp.Field(i).Interface()

		tag := vc.Type().Field(i).Tag
		fieldName := vc.Type().Field(i).Name

		switch tag.Get("diff") {
		case "value":
			if !reflect.DeepEqual(vcurrent, vproposed) {
				d.Param[fieldName] = diffValues{Current: vcurrent, Proposed: vproposed}
			}
		case "composition":
			same, added, deleted, err := checkDiffInComposition(vcurrent, vproposed)
			d.Composition[fieldName] = newDiffComposition()
			if err != nil {
				return nil, err
			}
			for _, n := range added {
				dc := d.Composition[fieldName]
				dc.New = append(dc.New, n)
				d.Composition[fieldName] = dc
			}
			for _, n := range deleted {
				dc := d.Composition[fieldName]
				dc.Deleted = append(dc.Deleted, n)
				d.Composition[fieldName] = dc
			}
			for _, n := range same {
				fmt.Printf("Checking compo under same path %s\n", n[0].Path())
				md, err := GetDiff(n[0], n[1])
				if err != nil {
					return nil, err
				}
				if !md.Empty() {
					fmt.Println("Diff detected in composition")
					dc := d.Composition[fieldName]
					dc.Modified = append(dc.Modified, *md)
					d.Composition[fieldName] = dc
				}
			}
		}

	}
	return &d, nil
}

func checkDiffInComposition(current, proposed interface{}) (samePath [][2]PathIdentifier, newPath, deletedPath []PathIdentifier, err error) {
	samePath = [][2]PathIdentifier{}
	newPath = []PathIdentifier{}
	deletedPath = []PathIdentifier{}

	// index all path in current composition
	currentMap := map[string]PathIdentifier{}
	s := reflect.ValueOf(current)
	for i := 0; i < s.Len(); i++ {
		item := s.Index(i)
		p, ok := item.Interface().(PathIdentifier)
		if !ok {
			err = fmt.Errorf("Compisition of non-PathIdentifier in current: %T", item.Interface())
			return
		}
		currentMap[p.Path()] = p
	}

	// index all path in proposed composition
	proposedMap := map[string]PathIdentifier{}
	s = reflect.ValueOf(proposed)
	for i := 0; i < s.Len(); i++ {
		item := s.Index(i)
		p, ok := item.Interface().(PathIdentifier)
		if !ok {
			err = fmt.Errorf("Compisition of non-PathIdentifier in proposed: %T", item.Interface())
			return
		}
		proposedMap[p.Path()] = p
	}

	//Deleted and Same
	for k := range currentMap {
		if p, ok := proposedMap[k]; ok {
			samePath = append(samePath, ([2]PathIdentifier{currentMap[k], p}))
		} else {
			deletedPath = append(deletedPath, currentMap[k])
		}
	}

	//New
	for k := range proposedMap {
		if _, ok := currentMap[k]; !ok {
			newPath = append(newPath, proposedMap[k])
		}
	}

	return samePath, newPath, deletedPath, nil
}

func diffReport(d *diff, report []string) []string {
	for k, v := range d.Param {
		line := fmt.Sprintf("%s.%s:%v->%v", d.Path, k, v.Current, v.Proposed)
		report = append(report, line)
	}

	for k, v := range d.Composition {
		if v.New != nil {
			for _, vv := range v.New {
				p := vv.(PathIdentifier)
				line := fmt.Sprintf("%s.%s:New=%s", d.Path, k, p.Path())
				report = append(report, line)
			}
		}
		if v.Deleted != nil {
			for _, vv := range v.Deleted {
				p := vv.(PathIdentifier)
				line := fmt.Sprintf("%s.%s:Deleted=%s", d.Path, k, p.Path())
				report = append(report, line)
			}
		}
		if v.Modified != nil {
			for _, dd := range v.Modified {
				report = diffReport(&dd, report)
			}
		}
	}
	return report
}

func GetDiffReport(current, proposed PathIdentifier) ([]string, error) {
	d, err := GetDiff(current, proposed)
	if err != nil {
		return nil, err
	}
	report := []string{}
	return diffReport(d, report), nil
}
