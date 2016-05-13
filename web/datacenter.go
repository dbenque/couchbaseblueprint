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
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/gorilla/mux"
)

var topoFileList = []string{"topo.png", "topo.yaml", "topo.json", "topo.dot", "topodef.yaml", "topoenv.yaml"}

func datacentersPage(w http.ResponseWriter, r *http.Request, user string) {
	folderPath := filepath.Join("public", "data", user, "dc")
	files, _ := ioutil.ReadDir(folderPath)
	type dc struct {
		User string
		Name string
	}

	data := struct {
		User        string
		Datacenters []dc
	}{
		User:        user,
		Datacenters: []dc{},
	}
	if files != nil {
		for _, file := range files {
			data.Datacenters = append(data.Datacenters, dc{Name: file.Name()})
		}
	}
	renderTemplate(w, "datacenters", data)
}

func deleteDatacenterPage(w http.ResponseWriter, r *http.Request, user string) {
	dcname := mux.Vars(r)["datacenterName"]
	version := mux.Vars(r)["version"]

	folder := filepath.Join(datacenterDirectory(user, dcname))
	if version != "" {
		folder = filepath.Join(folder, version)
	}
	os.RemoveAll(folder)
	http.Redirect(w, r, "/datacenters", http.StatusTemporaryRedirect)
}
func newDatacenterPage(w http.ResponseWriter, r *http.Request, user string) {
	r.ParseForm()
	d := r.Form.Get("datacenterName")
	var perm os.FileMode = 0777
	os.MkdirAll(datacenterDirectory(user, d), perm)
	http.Redirect(w, r, "/datacenter/"+d, http.StatusTemporaryRedirect)
}

func dcPage(w http.ResponseWriter, r *http.Request, user string) {
	r.ParseForm()
	version := r.Form.Get("v")
	datacenterName := mux.Vars(r)["datacenterName"]
	versions, _ := listVersions(datacenterDirectory(user, datacenterName))

	if version == "" && versions != nil && len(version) > 0 {
		version = strconv.Itoa(versions[len(versions)-1])
	}

	//diff part
	diffCurrent := r.Form.Get("current_verdion")
	diffProposed := r.Form.Get("proposed_verdion")
	diffReport := []string{}
	if diffCurrent != "" && diffProposed != "" {
		if r, err := computeDiff(user, datacenterName, diffCurrent, diffProposed); err == nil && r != nil {
			diffReport = r
		}
	}

	data := struct {
		User           string
		DatacenterName string
		Versions       []int
		Version        string
		DiffReport     []string
	}{
		User:           user,
		DatacenterName: datacenterName,
		Versions:       versions,
		Version:        version,
		DiffReport:     diffReport,
	}
	renderTemplate(w, "topoDC", data)
}

func computeDiff(user, datacenterName, currentVersion, proposedVersion string) ([]string, error) {
	// retrieve the 2 datacenters
	dcCurrent, err := getDatacenter(user, datacenterName, currentVersion)
	if err != nil {
		return nil, fmt.Errorf("Can't read datatacenter %s version %s. Error: %v", datacenterName, currentVersion, err)
	}
	dcProposed, err := getDatacenter(user, datacenterName, proposedVersion)
	if err != nil {
		return nil, fmt.Errorf("Can't read datatacenter %s version %s. Error: %v", datacenterName, proposedVersion, err)
	}

	return api.GetDiffReport(*dcCurrent, *dcProposed)
}

func dcUploadTopo(w http.ResponseWriter, r *http.Request, user string) {
	datacenterName := mux.Vars(r)["dcname"]

	// Prepare Folder for next topo
	dir, version, err := prepareNextVersion(datacenterDirectory(user, datacenterName))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dstPath := filepath.Join(dir, "topodef.yaml")
	if err := uploadFile(r, "file", dstPath); err != nil {
		fmt.Printf("%#v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dstEnvPath := filepath.Join(dir, "topoenv.yaml")
	if err := uploadFile(r, "envfile", dstEnvPath); err == nil {
		dstPath = dstPath + "+" + dstEnvPath
	}

	//write the files
	if err := createTopoFiles(dstPath, dir, datacenterName, version); err != nil {
		log(r, "%v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/datacenter/"+datacenterName, http.StatusTemporaryRedirect)
}

func createTopoFiles(defFiles, dir, datacenterName, version string) error {
	// creation of VDatacenter topology target
	var buf bytes.Buffer
	s, errDc := utils.TopoFromFile(defFiles, []api.Datacenter{api.NewDatacenter(datacenterName)}, &buf)
	if errDc != nil {
		fmt.Printf("%#v", errDc)
		return errDc
	}
	s[0].Version = version

	//write json and yaml topo files
	utils.ToFile(s[0], filepath.Join(dir, "topo"))

	//write dot topo file
	ioutil.WriteFile(filepath.Join(dir, "topo.dot"), []byte(fmt.Sprintf("digraph { \n%s\n}\n", buf.String())), 0777)

	//process dot file to build image
	cmd := exec.Command("dot", "-Tpng", "-o"+filepath.Join(dir, "topo.png"), filepath.Join(dir, "topo.dot"))
	var out bytes.Buffer
	var outerr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &outerr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Error Dot Processor: %s\n", string(outerr.Bytes()))
	}
	return nil
}

func experimentTopo(w http.ResponseWriter, r *http.Request, user string) {
	// build directory in case we start from scratch
	var perm os.FileMode = 0777
	dirExp := experimentDirectory(user)
	os.MkdirAll(dirExp, perm)

	//clean previous input
	for _, f := range topoFileList {
		os.Remove(filepath.Join(dirExp, f))
	}

	r.ParseForm()
	data := struct {
		User    string
		Topo    string
		ErrTopo string
		Env     string
		ErrEnv  string
		Error   string
	}{
		User:    user,
		Topo:    r.Form.Get("topoArea"),
		ErrTopo: "",
		Env:     r.Form.Get("envArea"),
		ErrEnv:  "",
		Error:   "",
	}

	if err := createTopoFilesFromStrings(data.Topo, data.Env, dirExp, "Datacenter_Experiment", "vExp"); err != nil {
		data.ErrTopo = err.Error()
		renderTemplate(w, "expTopo", data)
		return
	}

	renderTemplate(w, "expTopo", data)
}

func createTopoFilesFromStrings(topoStr, envStr, dirExp, datacenterName, version string) error {

	defFiles := ""
	// validate and write topodef
	var topodef api.ClusterGroupDefBluePrint
	if err := yaml.Unmarshal([]byte(topoStr), &topodef); err != nil {
		return err
	}
	defFiles = filepath.Join(dirExp, "topodef.yaml")
	ioutil.WriteFile(defFiles, []byte(topoStr), 0644)

	// validate and write environment
	environment := len(envStr) > 0
	if environment {
		var envdata api.EnvData
		if err := yaml.Unmarshal([]byte(envStr), &envdata); err != nil {
			return err
		}
		envDefFile := filepath.Join(dirExp, "topoenv.yaml")
		ioutil.WriteFile(envDefFile, []byte(envStr), 0644)
		defFiles = defFiles + "+" + envDefFile
	}

	//write the files
	if err := createTopoFiles(defFiles, dirExp, datacenterName, version); err != nil {
		return err
	}

	return nil
}

func experimentTopopush(w http.ResponseWriter, r *http.Request, user string) {
	r.ParseForm()
	data := struct {
		User    string
		Topo    string
		ErrTopo string
		Env     string
		ErrEnv  string
		Error   string
	}{
		User: user,
		Topo: r.Form.Get("topoToPush"),
		Env:  r.Form.Get("envToPush"),
	}

	dcnameList := r.Form["datacenters"]
	// Datacenter list
	for _, dcname := range dcnameList {
		dirDatacenter, version, err := prepareNextVersion(datacenterDirectory(user, dcname))
		if err != nil {
			data.ErrTopo = fmt.Sprintf("Push Failed: %s", err.Error())
			renderTemplate(w, "expTopo", data)
			return
		}

		if err := createTopoFilesFromStrings(data.Topo, data.Env, dirDatacenter, dcname, version); err != nil {
			data.ErrTopo = fmt.Sprintf("Push Failed: %s", err.Error())
			renderTemplate(w, "expTopo", data)
			return
		}
	}

	if len(dcnameList) == 1 {
		v, _ := latestVersion(datacenterDirectory(user, dcnameList[0]))
		http.Redirect(w, r, "/datacenter/"+dcnameList[0]+"?v="+v[1:], http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, "/datacenters", http.StatusTemporaryRedirect)
}

type linkdc struct {
	Uri  string
	Text string
}

func xdcrDatacenterLinks(user, directory string) []linkdc {
	pathToDatacentersTxt := filepath.Join(directory, "datacenters.txt")
	links := []linkdc{}
	b, err := ioutil.ReadFile(pathToDatacentersTxt)
	if err != nil {
		fmt.Printf("Error:%#v", err)
		return links
	}

	if len(b) < 3 {
		return links
	}

	for _, dctxt := range strings.Split(string(b), ",") {
		token := strings.Split(dctxt, ":")
		uri := filepath.Join("/datacenter", token[0]+"?v="+token[1])
		str := token[0] + " " + token[1]
		links = append(links, linkdc{uri, str})
	}
	return links
}

func xdcrDatacenterLinksForVersion(user string, xdcrVersion int) []linkdc {
	path := filepath.Join(xdcrDirectory(user), fmt.Sprintf("v%d", xdcrVersion))
	return xdcrDatacenterLinks(user, path)
}
