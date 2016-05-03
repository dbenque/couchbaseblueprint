package main

import (
	"bytes"
	"couchbasebp/api"
	"couchbasebp/example"
	"couchbasebp/utils"
	"couchbasebp/web"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"gopkg.in/yaml.v2"
)

func main() {

	if len(os.Args) == 1 {
		example.Gen_hos1()
		example.Gen_RBox1()
		return
	}

	// From DC File
	if len(os.Args) == 2 {
		if os.Args[1] == "server" {
			web.ServeHTTP()
		} else {
			fromDCFile(os.Args[1])
		}
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
	DCs := []api.Datacenter{}
	for i := 0; i < dcCount; i++ {
		DCs = append(DCs, api.NewDatacenter(fmt.Sprintf("DC%d", i+1)))
	}
	fromFolder(os.Args[2], os.Args[1], DCs)

}

func fromFolder(folder, format string, DCs []api.Datacenter) {
	var buf bytes.Buffer
	_, err := utils.TopoFromFile(folder+"/couchbase."+format, DCs, &buf)
	if err != nil {
		return
	}
	utils.XDCRFromFile(folder+"/XDCR."+format, DCs, &buf)
	fmt.Printf("digraph { \n%s\n}\n", buf.String())
}

func fromDCFile(file string) {
	//DCInjector is a struct to simulate operations to be done at Datacenter level
	type DCInjector struct {
		//Topos key=topoFile[+envFile] value={list of datacenters on which the topology need to be deployed (one after the other)}
		Topos map[string][]string
		//XDCRs key=xdcrFile[+envFile] value={list of datacenters on which the rules need to be applied. Union of buckets of listed DC will be use to compute the XDCR instances}
		XDCRs map[string][]string
	}

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

	datacenters := map[string]api.Datacenter{}
	for _, dcs := range dcinjector.Topos {
		for _, d := range dcs {
			if _, ok := datacenters[d]; ok {
				continue
			}
			datacenters[d] = api.NewDatacenter(d)
		}
	}

	var buf bytes.Buffer

	for f, dcs := range dcinjector.Topos {
		for _, d := range dcs {
			aDc := datacenters[d]
			if s, err := utils.TopoFromFile(f, []api.Datacenter{aDc}, &buf); err == nil {
				datacenters[d] = s[0]
			}

		}
	}

	for f, dcs := range dcinjector.XDCRs {
		DCS := []api.Datacenter{}
		for _, d := range dcs {
			DCS = append(DCS, datacenters[d])
		}
		utils.XDCRFromFile(f, DCS, &buf)
	}
	fmt.Printf("digraph { \n%s\n}\n", buf.String())
}
