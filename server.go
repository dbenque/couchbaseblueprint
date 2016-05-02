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
	"ListUsers": func() []string {
		return listUsers()
	},
	"listDatacenter": func(user string) []string {
		return listDatacenter(user)
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
		User string
	}{
		User: user,
	}
	renderTemplate(w, "mainPage", data)
}

func usersPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "users", nil)
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

func deleteUserPage(w http.ResponseWriter, r *http.Request) {
	os.RemoveAll(userDirectory(mux.Vars(r)["user"]))
	http.Redirect(w, r, "/users", http.StatusTemporaryRedirect)
}

func deleteDatacenterPage(w http.ResponseWriter, r *http.Request) {
	user := getuser(r)
	if user == "" {
		http.Redirect(w, r, "/main", http.StatusTemporaryRedirect)
		return
	}
	dcname := mux.Vars(r)["datacenterName"]
	version := mux.Vars(r)["version"]
	
	folder := filepath.Join(datacenterDirectory(user,dcname))
	if version!="" {
		folder = filepath.Join(folder,version)
	}
	os.RemoveAll(folder)
	http.Redirect(w, r, "/datacenters", http.StatusTemporaryRedirect)	
}
func newDatacenterPage(w http.ResponseWriter, r *http.Request) {
	user := getuser(r)
	if user == "" {
		http.Redirect(w, r, "/main", http.StatusTemporaryRedirect)
		return
	}
	r.ParseForm()
	d := r.Form.Get("datacenterName")
	http.Redirect(w, r, "/topo/"+user+"/datacenter/"+d, http.StatusTemporaryRedirect)
}

func dcPage(w http.ResponseWriter, r *http.Request) {
	user := getuser(r)
	if user == "" {
		http.Redirect(w, r, "/main", http.StatusTemporaryRedirect)
		return
	}
	d := mux.Vars(r)["datacenterName"]
	http.Redirect(w, r, "/topo/"+user+"/datacenter/"+d, http.StatusTemporaryRedirect)
}

func dcTopoPageForm(w http.ResponseWriter, r *http.Request) {
	u := mux.Vars(r)["user"]
	d := mux.Vars(r)["datacenterName"]
	fmt.Printf("USer %s, datacenter %s\n", u, d)

	http.Redirect(w, r, "/topo/"+u+"/datacenter/"+d, http.StatusTemporaryRedirect)
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

func uploadFile(r*http.Request,formField,destinationFilePath string) error {
	// Source
	file, _, err := r.FormFile(formField)
	if err != nil {
		return err
	}
	defer file.Close()

	// Destination
	dst, err := os.Create(destinationFilePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, file); err != nil {		
		return err
	}
	return nil
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

	dstPath := filepath.Join(dir, "topodef.yaml")
	if err := uploadFile(r,"file",dstPath); err!=nil {
		fmt.Printf("%#v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	
	dstEnvPath := filepath.Join(dir, "topoenv.yaml")
	
	if err := uploadFile(r,"envfile",dstEnvPath); err==nil {
			dstPath = dstPath+"+"+dstEnvPath
	}

	// creation of VDatacenter topology target
	var buf bytes.Buffer
	errDc, s := TopoFromFile(dstPath, []Datacenter{NewDatacenter(datacenterName)}, &buf)
	if errDc != nil {
		fmt.Printf("%#v", err)
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
		fmt.Printf("Error Dot: %#v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/topo/"+user+"/datacenter/"+datacenterName, http.StatusTemporaryRedirect)
}

func xdcrPage(w http.ResponseWriter, r *http.Request) {
	user:=getuser(r)

		data := struct {
		User           string
		Versions       []int
		Version        string
	}{
		User:           user,
		Versions:       []int{},
		Version:        "",
	}
	renderTemplate(w, "xdcr", data)
}

func datacenterURI(user, datacenterName string) string {
	return filepath.Join("/data", user, "dc", datacenterName)
}

func userDirectory(user string) string {
	return filepath.Join("public", "data", user)
}

func datacenterDirectory(user, datacenterName string) string {
	return filepath.Join("public", "data", user, "dc", datacenterName)
}

func listUsers() []string {
	users := []string{}
	files, _ := ioutil.ReadDir(filepath.Join("public", "data"))
	for _, file := range files {
		users = append(users, file.Name())
	}
	return users
}
func listDatacenter(user string) []string {
	dcs := []string{}
	files, _ := ioutil.ReadDir(filepath.Join(userDirectory(user),"dc"))
	for _, file := range files {
		dcs = append(dcs, file.Name())
	}
	return dcs
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
