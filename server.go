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
	"time"

	"github.com/gorilla/mux"

	"net/http"
)

var templates *template.Template

var fns = template.FuncMap{
	"ImgPath": func(user, datacenterName, version string) string {
		if version != "" {
			return filepath.Join(datacenterURI(user, datacenterName), "v"+version, "topo.png")
		}
		if lv, err := latestVersion(datacenterDirectory(user, datacenterName)); err == nil {
			return filepath.Join(datacenterURI(user, datacenterName), lv, "topo.png")
		}
		return ""
	},
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func mainPage(w http.ResponseWriter, r *http.Request) {

	user := ""
	//check cookie
	if c, err := r.Cookie("user"); err == nil {
		user = c.Value
	}

	//check uri
	r.ParseForm()
	userURL := r.Form.Get("user")
	if userURL != "" {
		user = userURL
		expiration := time.Now().Add(24 * time.Hour)
		cookie := http.Cookie{Name: "user", Value: user, Expires: expiration}
		http.SetCookie(w, &cookie)
	}

	data := struct {
		User           string
		DatacenterName string
	}{
		User:           user,
		DatacenterName: "test",
	}

	fmt.Printf("data:%#v\n", data)

	renderTemplate(w, "mainPage", data)
}

func getuser(r *http.Request) string {
	user := ""
	//check cookie
	if c, err := r.Cookie("user"); err == nil {
		user = c.Value
	}
	return user
}

func datacentersPage(w http.ResponseWriter, r *http.Request) {
	user := getuser(r)
	if user == "" {
		http.Redirect(w, r, "/main", http.StatusTemporaryRedirect)
		return
	}
	folderPath := filepath.Join("public", "data", user, "dc")
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		http.Redirect(w, r, "/main", http.StatusTemporaryRedirect)
		return
	}

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

	for _, file := range files {
		data.Datacenters = append(data.Datacenters, dc{Name: file.Name()})
	}
	renderTemplate(w, "datacenters", data)
}

func newDatacenterPage(w http.ResponseWriter, r *http.Request) {
	user := getuser(r)
	if user == "" {
		http.Redirect(w, r, "/main", http.StatusTemporaryRedirect)
		return
	}
	r.ParseForm()
	d := r.Form.Get("datacenterName")
	http.Redirect(w, r, "/topo/"+user+"/datacenter/"+d, http.StatusMovedPermanently)
}

func dcPage(w http.ResponseWriter, r *http.Request) {
	user := getuser(r)
	if user == "" {
		http.Redirect(w, r, "/main", http.StatusTemporaryRedirect)
		return
	}
	d := mux.Vars(r)["datacenterName"]
	http.Redirect(w, r, "/topo/"+user+"/datacenter/"+d, http.StatusMovedPermanently)
}

func dcTopoPageForm(w http.ResponseWriter, r *http.Request) {
	u := mux.Vars(r)["user"]
	d := mux.Vars(r)["datacenterName"]
	fmt.Printf("USer %s, datacenter %s\n", u, d)

	http.Redirect(w, r, "/topo/"+u+"/datacenter/"+d, http.StatusMovedPermanently)
}

func datacenterURI(user, datacenterName string) string {
	return filepath.Join("/data", user, "dc", datacenterName)
}

func datacenterDirectory(user, datacenterName string) string {
	return filepath.Join("public", "data", user, "dc", datacenterName)
}

func dcTopoPage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	version := r.Form.Get("v")
	user := mux.Vars(r)["user"]
	datacenterName := mux.Vars(r)["datacenterName"]
	versions, _ := listVersions(datacenterDirectory(user, datacenterName))

	if version == "" && versions != nil && len(version) > 0 {
		version = strconv.Itoa(versions[len(versions)-1])
	}

	data := struct {
		User           string
		DatacenterName string
		Versions       []int
		Version        string
	}{
		User:           user,
		DatacenterName: datacenterName,
		Versions:       versions,
		Version:        version,
	}
	renderTemplate(w, "topoDC", data)
}

func dcUploadTopo(w http.ResponseWriter, r *http.Request) {
	user := mux.Vars(r)["user"]
	datacenterName := mux.Vars(r)["dcname"]

	// Prepare Folder for next topo
	var perm os.FileMode = 0777
	dirDc := datacenterDirectory(user, datacenterName)
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
	http.Redirect(w, r, "/topo/"+user+"/datacenter/"+datacenterName, http.StatusMovedPermanently)
}

func listVersions(folderPath string) ([]int, error) {
	versions := []int{}
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}
	max := 0
	for _, file := range files {
		if file.Name()[0] == 'v' && len(file.Name()) > 1 {
			i, err := strconv.Atoi(file.Name()[1:])
			if err == nil && i > max {
				versions = append(versions, i)
			}
		}
	}
	return versions, nil
}
func latestVersionInt(folderPath string) (int, error) {
	max := 0
	if l, err := listVersions(folderPath); err == nil {
		for _, i := range l {
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
