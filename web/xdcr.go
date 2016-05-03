package web

import (
	"bytes"
	"couchbasebp/api"
	"couchbasebp/utils"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

func deleteXDCRPage(w http.ResponseWriter, r *http.Request, user string) {
	version := mux.Vars(r)["version"]
	os.RemoveAll(filepath.Join(xdcrDirectory(user), version))
	http.Redirect(w, r, "/xdcr", http.StatusTemporaryRedirect)
}

func uploadxdcr(w http.ResponseWriter, r *http.Request, user string) {
	log(r, "uploadxdcr")

	// Prepare Folder for xdcr
	dir, _, err := prepareNextVersion(xdcrDirectory(user))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log(r, "new xdrc folder %s", dir)

	dstPath := filepath.Join(dir, "xdcrdef.yaml")
	if err := uploadFile(r, "xdcrfile", dstPath); err != nil {
		fmt.Printf("%#v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log(r, "new xdrc definition %s", dstPath)

	dstEnvPath := filepath.Join(dir, "xdcrenv.yaml")
	if err := uploadFile(r, "envfile", dstEnvPath); err == nil {
		dstPath = dstPath + "+" + dstEnvPath
		log(r, "new xdrc environment %s", dstEnvPath)
	} else {
		log(r, "no xdrc environment file")
	}

	// Datacenter list
	r.ParseForm()
	dcFiles := []string{}
	dcnameList := r.Form["datacenters"]
	log(r, "xdrc on datacenters %v", dcnameList)
	for _, dcname := range dcnameList {
		dcp := datacenterDirectory(user, dcname)
		if version, err := latestVersion(dcp); err == nil {
			log(r, "xdrc on datacenter '%s' for version '%s'", dcname, version)
			dcFiles = append(dcFiles, filepath.Join(dcp, version, "topo.yaml"))
		} else {
			log(r, "Error no version for datacenter '%s'", version)
		}
	}

	//Read the datacenters
	datacenters := []api.Datacenter{}
	log(r, "Retrieving datacenters")
	for _, dc := range dcFiles {
		datacenter, err := utils.DatacenterFromFile(dc)
		if err != nil {
			fmt.Printf("%#v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		datacenters = append(datacenters, *datacenter)
	}

	log(r, "Datacenters for XDRC:\n%#v", datacenters)

	if err := createXDCRFiles(user, dstPath, dir, datacenters); err != nil {
		fmt.Printf("Error Dot: %#v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	version, _ := latestVersionInt(xdcrDirectory(user))
	uri := fmt.Sprintf("/xdcr?v=%d", version)
	http.Redirect(w, r, uri, http.StatusTemporaryRedirect)
}

func createXDCRFiles(user, defFiles, dir string, datacenters []api.Datacenter) error {
	//Build the xdcrs
	var buf bytes.Buffer
	if errXDCR := utils.XDCRFromFile(defFiles, datacenters, &buf); errXDCR != nil {
		fmt.Printf("%#v", errXDCR)
		return errXDCR
	}

	var bufDC bytes.Buffer
	dctxt := []string{}
	for i := range datacenters {
		datacenters[i].Dot(&bufDC)
		// for back compatibility when there was no version
		if len(datacenters[i].Version) < 2 {
			datacenters[i].Version = "v0"
		}
		dctxt = append(dctxt, datacenters[i].Name+":"+datacenters[i].Version[1:])
	}
	ioutil.WriteFile(filepath.Join(dir, "datacenters.txt"), []byte(strings.Join(dctxt, ",")), 0644)

	//write dot topo file
	ioutil.WriteFile(filepath.Join(dir, "xdcr.dot"), []byte(fmt.Sprintf("digraph { \n%s\n%s\n}\n", bufDC.String(), buf.String())), 0777)

	//process dot file to build image
	cmd := exec.Command("dot", "-Tpng", "-o"+filepath.Join(dir, "xdcr.png"), filepath.Join(dir, "xdcr.dot"))
	var out bytes.Buffer
	var outerr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &outerr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Error Dot Processor: %s\n", string(outerr.Bytes()))
	}
	return nil
}

func xdcrPage(w http.ResponseWriter, r *http.Request, user string) {
	r.ParseForm()
	version := r.Form.Get("v")

	data := struct {
		User            string
		Versions        []int
		Version         string
		DatacenterLinks []linkdc
	}{
		User:            user,
		Versions:        []int{},
		Version:         version,
		DatacenterLinks: []linkdc{},
	}
	renderTemplate(w, "xdcr", data)
}

var xdcrFileList = []string{"xdcr.png", "xdcrdef.yaml", "xdcrenv.yaml", "datacenters.txt"}

func experimentXDCR(w http.ResponseWriter, r *http.Request, user string) {
	// build directory in case we start from scratch
	var perm os.FileMode = 0777
	dirExp := experimentDirectory(user)
	os.MkdirAll(dirExp, perm)

	//clean previous input
	for _, f := range xdcrFileList {
		os.Remove(filepath.Join(dirExp, f))
	}

	r.ParseForm()

	data := struct {
		User            string
		XDCR            string
		ErrXDCR         string
		Env             string
		ErrEnv          string
		Error           string
		DatacenterLinks []linkdc
	}{
		User:            user,
		XDCR:            r.Form.Get("xdcrArea"),
		ErrXDCR:         "",
		Env:             r.Form.Get("envArea"),
		ErrEnv:          "",
		Error:           "",
		DatacenterLinks: []linkdc{},
	}

	defFiles := ""
	// validate and write topodef
	var xdcrdef api.XDCRDefBluePrint
	if err := yaml.Unmarshal([]byte(data.XDCR), &xdcrdef); err != nil {
		data.Error = err.Error()
		renderTemplate(w, "expXdcr", data)
		return

	}
	defFiles = filepath.Join(dirExp, "xdcrdef.yaml")
	ioutil.WriteFile(defFiles, []byte(data.XDCR), 0644)

	// validate and write environment
	environment := len(data.Env) > 0
	if environment {
		var envdata api.EnvData
		if err := yaml.Unmarshal([]byte(data.Env), &envdata); err != nil {
			data.Error = err.Error()
			renderTemplate(w, "expXdcr", data)
			return
		}
		envDefFile := filepath.Join(dirExp, "topoenv.yaml")
		ioutil.WriteFile(envDefFile, []byte(data.Env), 0644)
		defFiles = defFiles + "+" + envDefFile
	}

	//Load datacenter
	datacenters := []api.Datacenter{}
	log(r, "Retrieving datacenters")
	listDatacenterNames := r.Form["datacenters"]
	for _, dcname := range listDatacenterNames {
		version := r.Form.Get(dcname + "_version")
		datacenter, err := utils.DatacenterFromFile(filepath.Join(datacenterDirectory(user, dcname), "v"+version, "topo.yaml"))
		if err != nil {
			fmt.Printf("%#v", err)
			data.Error = err.Error()
			renderTemplate(w, "expXdcr", data)
			return
		}
		datacenters = append(datacenters, *datacenter)
	}

	//create the files
	if err := createXDCRFiles(user, defFiles, dirExp, datacenters); err != nil {
		log(r, "%#v", err)
		data.Error = err.Error()
		renderTemplate(w, "expXdcr", data)
		return
	}

	//	xdcrDatacenterLinks(user,filepath.Join(xdcrDirectory(user), xdcrversion, "datacenters.txt"))
	//data.DatacenterLinks = xdcrDatacenterLinks(user, filepath.Join(experimentDirectory(user), "datacenters.txt"))

	renderTemplate(w, "expXdcr", data)
}

func experimentXDCRsave(w http.ResponseWriter, r *http.Request, user string) {
	dir, version, err := prepareNextVersion(xdcrDirectory(user))
	if err != nil {
		log(r, "%#v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, f := range xdcrFileList {
		err := copyFile(filepath.Join(experimentDirectory(user), f), filepath.Join(dir, f))
		if err != nil {
			log(r, "%#v", err)
		}
	}
	http.Redirect(w, r, "/xdcr?v="+version[1:], http.StatusTemporaryRedirect)
}
