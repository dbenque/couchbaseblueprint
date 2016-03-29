package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type DCInjector struct {
	Topos map[string][]string
	XDCRs map[string][]string
}

func main() {

	if len(os.Args) == 1 {
		gen_sample1()
		gen_RBox1()
		return
	}

	// From DC File
	if len(os.Args) == 2 {
		FromDCFile(os.Args[1])
		return

	}

	// From folder

	if (len(os.Args) != 3 && len(os.Args) != 4) || (os.Args[1] != "yaml" && os.Args[1] != "json") {
		fmt.Println("First parameter must be input format [yaml|json] and the second parameter must be the folder containing the files (couchbase.yaml and XDCR.yaml). Optional last param, Datacenter count")
	}
	dcCount := 1
	if len(os.Args) == 4 {
		var err error
		dcCount, err = strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Println("Error with datacenter counter. Last parameter should be a number ")
		}
	}
	DCs := []Datacenter{}
	for i := 0; i < dcCount; i++ {
		DCs = append(DCs, NewDatacenter(fmt.Sprintf("DC%d", i+1)))
	}
	FromFolder(os.Args[2], os.Args[1], DCs)

}

func ToFile(v interface{}, filePath string) {
	//json
	{
		b, _ := json.Marshal(v)
		var out bytes.Buffer
		json.Indent(&out, b, " ", "\t")
		ioutil.WriteFile(filePath+".json", out.Bytes(), 0777)
	}
	//yaml
	{
		y, _ := yaml.Marshal(v)
		ioutil.WriteFile(filePath+".yaml", y, 0777)

	}
}

func TopoFromFile(file string, DCs []Datacenter, w io.Writer) (error, []Datacenter) {
	format := strings.Split(file, ".")[1]
	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return err, nil
	}
	var cgdefBlueprint ClusterGroupDefBluePrint
	switch format {
	case "json":
		err = json.Unmarshal(b, &cgdefBlueprint)
	case "yaml":
		err = yaml.Unmarshal(b, &cgdefBlueprint)
	}
	if err != nil {
		fmt.Println(err)
		return err, nil
	}

	for _, d := range cgdefBlueprint.ClusterGroups {
		for i := range DCs {
			DCs[i].AddClusterGroupDef(d)
		}
	}

	for i := range DCs {
		DCs[i].Dot(w)
	}

	return nil, DCs
}

func XDCRFromFile(file string, DCs []Datacenter, w io.Writer) {

	format := strings.Split(file, ".")[1]
	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return
	}
	var xdcrdefBlueprint XDCRDefBluePrint
	switch format {
	case "json":
		err = json.Unmarshal(b, &xdcrdefBlueprint)
	case "yaml":
		err = yaml.Unmarshal(b, &xdcrdefBlueprint)
	}
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, xdcr := range xdcrdefBlueprint.XDCRDefs {
		for _, x := range NewXDCR(xdcr, DCs) {
			x.Dot(w)
		}
	}
}

func FromFolder(folder, format string, DCs []Datacenter) {
	var buf bytes.Buffer
	err, _ := TopoFromFile(folder+"/couchbase."+format, DCs, &buf)
	if err != nil {
		return
	}
	XDCRFromFile(folder+"/XDCR."+format, DCs, &buf)
	fmt.Printf("digraph { \n%s\n}\n", buf.String())
}

func FromDCFile(file string) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return
	}
	var dcinjector DCInjector
	err = yaml.Unmarshal(b, &dcinjector)
	if err != nil {
		fmt.Println(err)
		return
	}

	datacenters := map[string]Datacenter{}
	for _, dcs := range dcinjector.Topos {
		for _, d := range dcs {
			if _, ok := datacenters[d]; ok {
				continue
			}
			datacenters[d] = NewDatacenter(d)
		}
	}

	var buf bytes.Buffer

	for f, dcs := range dcinjector.Topos {
		for _, d := range dcs {
			aDc := datacenters[d]
			if err, s := TopoFromFile(f, []Datacenter{aDc}, &buf); err == nil {
				datacenters[d] = s[0]
			}

		}
	}

	for f, dcs := range dcinjector.XDCRs {
		DCS := []Datacenter{}
		for _, d := range dcs {
			DCS = append(DCS, datacenters[d])
		}
		XDCRFromFile(f, DCS, &buf)
	}
	fmt.Printf("digraph { \n%s\n}\n", buf.String())
}
