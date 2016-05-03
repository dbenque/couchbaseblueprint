package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"gopkg.in/yaml.v2"
)

func main() {

	if len(os.Args) == 1 {
		gen_hos1()
		gen_RBox1()
		return
	}

	// From DC File
	if len(os.Args) == 2 {
		if os.Args[1] == "server" {
			templates = template.Must(template.New("abc").Funcs(fns).ParseGlob("public/template/*.html"))
			r := mux.NewRouter()
			r.HandleFunc("/main", mainPage)
			r.HandleFunc("/", mainPage)
			r.HandleFunc("/users", usersPage)
			r.HandleFunc("/deleteuser/{user}", deleteUserPage)
			r.HandleFunc("/topo", dcTopoPageForm)
			r.HandleFunc("/datacenters", datacentersPage)
			r.HandleFunc("/deletedatacenter/{datacenterName}/{version}", deleteDatacenterPage)
			r.HandleFunc("/deletedatacenter/{datacenterName}", deleteDatacenterPage)
			r.HandleFunc("/datacenter/{datacenterName}", dcPage)
			r.HandleFunc("/newdatacenter", newDatacenterPage)
			r.HandleFunc("/topo/{user}/datacenter/{datacenterName}", dcTopoPage)
			r.HandleFunc("/uploadTopo/{user}/datacenter/{dcname}", dcUploadTopo)
			r.HandleFunc("/xdcr", xdcrPage)
			r.HandleFunc("/deletexdcr/{version}", deleteXDCRPage)
			r.HandleFunc("/uploadxdcr", uploadxdcr)
			r.HandleFunc("/experiment/topo", experimentTopo)
			r.HandleFunc("/experiment/topopush", experimentTopopush)
			r.HandleFunc("/experiment/xdcr", experimentXDCR)
			r.HandleFunc("/experiment/xdcrsave", experimentXDCRsave)

			r.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
			http.Handle("/", r)
			http.ListenAndServe(":1323", nil)
		} else {
			FromDCFile(os.Args[1])
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

func DatacenterFromFile(file string) (*Datacenter, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	format := strings.Split(file, ".")[1]
	var datacenter Datacenter

	switch format {
	case "json":
		err = json.Unmarshal(b, &datacenter)
	case "yaml":
		err = yaml.Unmarshal(b, &datacenter)
	}
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &datacenter, nil

}
func ProcessEnv(file, envfile string) (error, string) {

	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return err, ""
	}

	format := strings.Split(envfile, ".")[1]
	benv, err := ioutil.ReadFile(envfile)
	if err != nil {
		fmt.Println(err)
		return err, ""
	}

	var envData EnvData
	switch format {
	case "json":
		err = json.Unmarshal(benv, &envData)
	case "yaml":
		err = yaml.Unmarshal(benv, &envData)
	}
	if err != nil {
		fmt.Println(err)
		return err, ""
	}

	str := string(b)

	for k, v := range envData.Replacements {
		str = strings.Replace(str, k, v, -1)
	}

	tfile, err := ioutil.TempFile("./", "tmp_")
	defer tfile.Close()
	if err != nil {
		fmt.Println(err)
		return err, ""
	}

	_, err = tfile.WriteString(str)
	if err != nil {
		fmt.Println(err)
		return err, ""
	}

	return nil, tfile.Name()

}

func TopoFromFile(files string, DCs []Datacenter, w io.Writer) (error, []Datacenter) {

	splitted := strings.Split(files, "+")
	file := splitted[0]
	format := strings.Split(file, ".")[1]
	if len(splitted) == 2 {
		envFile := splitted[1]
		var err error
		err, file = ProcessEnv(file, envFile)
		if err != nil {
			fmt.Println(err)
			return err, nil
		}
		tmpFile := file
		defer os.Remove(tmpFile)
	}

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

func XDCRFromFile(files string, DCs []Datacenter, w io.Writer) error {
	splitted := strings.Split(files, "+")
	file := splitted[0]
	format := strings.Split(file, ".")[1]
	if len(splitted) == 2 {
		envFile := splitted[1]
		var err error
		err, file = ProcessEnv(file, envFile)
		if err != nil {
			fmt.Println(err)
			return err
		}
		tmpFile := file
		defer os.Remove(tmpFile)
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return err
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
		return err
	}

	for _, xdcr := range xdcrdefBlueprint.XDCRDefs {
		for _, x := range NewXDCR(xdcr, DCs) {
			x.Dot(w)
		}
	}

	return nil
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
