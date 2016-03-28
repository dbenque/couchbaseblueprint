package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"gopkg.in/yaml.v2"
)

func main() {

	if len(os.Args) == 1 {
		gen_sample1()
		gen_RBox1()
		return
	}

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
	FromFile(os.Args[2], os.Args[1], dcCount)
	//FromFile(os.Args[2], "json")
	//FromFile(os.Args[2], "yaml")

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

func FromFile(folder, format string, dcCount int) {

	DCs := []Datacenter{}
	for i := 0; i < dcCount; i++ {
		DCs = append(DCs, NewDatacenter(fmt.Sprintf("DC%d", i+1)))
	}

	switch format {
	case "json":
		b, err := ioutil.ReadFile(folder + "/couchbase.json")
		if err != nil {
			fmt.Println(err)
			return
		}
		var cgdefBlueprint ClusterGroupDefBluePrint
		err = json.Unmarshal(b, &cgdefBlueprint)
		if err != nil {
			fmt.Println(err)
			return
		}
		b, err = ioutil.ReadFile(folder + "/XDCR.json")
		if err != nil {
			fmt.Println(err)
			return
		}
		var xdcrdefBlueprint XDCRDefBluePrint
		err = json.Unmarshal(b, &xdcrdefBlueprint)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, d := range cgdefBlueprint.ClusterGroups {
			for i := range DCs {
				DCs[i].AddClusterGroupDef(d)
			}
		}

		var buf bytes.Buffer
		for i := range DCs {
			DCs[i].Dot(&buf)
		}

		//fmt.Printf("%#v\n", xdcrdefBlueprint)

		for _, xdcr := range xdcrdefBlueprint.XDCRDefs {
			for _, x := range NewXDCR(xdcr, DCs) {
				x.Dot(&buf)
			}
		}
		fmt.Printf("digraph { \n%s\n}\n", buf.String())

	case "yaml":
		b, err := ioutil.ReadFile(folder + "/couchbase.yaml")
		if err != nil {
			fmt.Println(err)
			return
		}
		var cgdefBlueprint ClusterGroupDefBluePrint
		err = yaml.Unmarshal(b, &cgdefBlueprint)
		if err != nil {
			fmt.Println(err)
			return
		}
		b, err = ioutil.ReadFile(folder + "/XDCR.yaml")
		if err != nil {
			fmt.Println(err)
			return
		}
		var xdcrdefBlueprint XDCRDefBluePrint
		err = yaml.Unmarshal(b, &xdcrdefBlueprint)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, d := range cgdefBlueprint.ClusterGroups {
			for i := range DCs {
				DCs[i].AddClusterGroupDef(d)
			}
		}

		var buf bytes.Buffer
		for i := range DCs {
			DCs[i].Dot(&buf)
		}

		//fmt.Printf("%#v\n", xdcrdefBlueprint)

		for _, xdcr := range xdcrdefBlueprint.XDCRDefs {
			for _, x := range NewXDCR(xdcr, DCs) {
				x.Dot(&buf)
			}
		}
		fmt.Printf("digraph { \n%s\n}\n", buf.String())

	}

}
