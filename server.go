package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"

	"net/http"
)

var templates *template.Template

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		User           string
		DatacenterName string
	}{
		User:           "David",
		DatacenterName: "test",
	}

	renderTemplate(w, "mainPage", data)
	w.WriteHeader(http.StatusOK)
}

func dcTopoPageForm(w http.ResponseWriter, r *http.Request) {
	u := mux.Vars(r)["user"]
	d := mux.Vars(r)["datacenterName"]
	fmt.Printf("USer %s, datacenter %s\n", u, d)

	http.Redirect(w, r, "/topo/"+u+"/datacenter/"+d, http.StatusMovedPermanently)
}

func dcTopoPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		User           string
		DatacenterName string
	}{
		User:           mux.Vars(r)["user"],
		DatacenterName: mux.Vars(r)["datacenterName"],
	}
	renderTemplate(w, "uploadTopoDC", data)
	w.WriteHeader(http.StatusOK)
}

func dcUploadTopo(w http.ResponseWriter, r *http.Request) {
	user := mux.Vars(r)["user"]
	datacenterName := mux.Vars(r)["dcname"]

	// Prepare Folder for next topo
	var perm os.FileMode = 0777
	dirDc := filepath.Join("public", "data", user, "dc", datacenterName)
	os.MkdirAll(dirDc, perm)
	version, err := nextVersion(dirDc)
	if err != nil {
		fmt.Printf("%#v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	dir := filepath.Join(dirDc, version)
	os.MkdirAll(dir, perm)

	// Source
	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Destination
	dstPath := filepath.Join(dir, "topodef.yaml")
	dst, err := os.Create(dstPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, file); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// creation of VDatacenter topology target
	var buf bytes.Buffer
	errDc, s := TopoFromFile(dstPath, []Datacenter{NewDatacenter(datacenterName)}, &buf)
	if errDc != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//write json and yaml topo files
	ToFile(s[0], filepath.Join(dir, "topo"))

	//write dot topo file
	ioutil.WriteFile(filepath.Join(dir, "topo.dot"), []byte(fmt.Sprintf("digraph { \n%s\n}\n", buf.String())), 0777)

	//process dot file to build image
	cmd := exec.Command("dot", "-Tpng", "-o"+filepath.Join(dir, "topo.png"), filepath.Join(dir, "topo.dot"))
	var out bytes.Buffer
	var outerr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &outerr
	if err := cmd.Run(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := struct {
		User              string
		DatacenterName    string
		DatacenterVersion string
	}{
		User:              user,
		DatacenterName:    datacenterName,
		DatacenterVersion: version,
	}

	renderTemplate(w, "topoDC", data)
	w.WriteHeader(http.StatusOK)

}
func latestVersionInt(folderPath string) (int, error) {
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return 0, err
	}
	max := 0
	for _, file := range files {
		if file.Name()[0] == 'v' && len(file.Name()) > 1 {
			i, err := strconv.Atoi(file.Name()[1:])
			if err == nil && i > max {
				max = i
			}
		}
	}
	if max == 0 {
		return 0, fmt.Errorf("No version found in folderPath")
	}
	return max, nil
}

func nextVersion(folderPath string) (string, error) {
	v, _ := latestVersionInt(folderPath)
	v++
	return fmt.Sprintf("v%d", v), nil
}
func latestVersion(folderPath string) (string, error) {
	v, _ := latestVersionInt(folderPath)
	return fmt.Sprintf("v%d", v), nil
}
