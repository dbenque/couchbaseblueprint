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
	"strings"
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
	"ImgXDCRPath": func(user, version string) string {
		if version != "" {
			return filepath.Join(xdcrURI(user), "v"+version, "xdcr.png")
		}
		if lv, err := latestVersion(xdcrDirectory(user)); err == nil {
			return filepath.Join(xdcrURI(user), lv, "xdcr.png")
		}
		return ""
	},
	"ListUsers": func() []string {
		return listUsers()
	},
	"listDatacenter": func(user string) []string {
		return listDatacenter(user)
	},
	"listXDCR": func(user string) []int {
		return listXDCR(user)
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

func deleteXDCRPage(w http.ResponseWriter, r *http.Request) {
	user := getuser(r)
	if user == "" {
		http.Redirect(w, r, "/main", http.StatusTemporaryRedirect)
		return
	}
	version := mux.Vars(r)["version"]
	os.RemoveAll(filepath.Join(xdcrDirectory(user),version))
	http.Redirect(w, r, "/xdcr", http.StatusTemporaryRedirect)
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
	user := getuser(r)
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
	dir,err:=prepareNextVersion(datacenterDirectory(user,datacenterName))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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
func uploadxdcr(w http.ResponseWriter, r *http.Request) {
	user := getuser(r)
	if user == "" {
		http.Redirect(w, r, "/main", http.StatusTemporaryRedirect)
		log(r,"no user")
		return
	}
	
	log(r,"uploadxdcr")
	
	// Prepare Folder for xdcr
	dir,err:=prepareNextVersion(xdcrDirectory(user))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	
	log(r,"new xdrc folder %s",dir)
	
	dstPath := filepath.Join(dir, "xdcrdef.yaml")
	if err := uploadFile(r,"xdcrfile",dstPath); err!=nil {
		fmt.Printf("%#v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	
	log(r,"new xdrc definition %s",dstPath)
	
	dstEnvPath := filepath.Join(dir, "xdcrenv.yaml")	
	if err := uploadFile(r,"envfile",dstEnvPath); err==nil {
			dstPath = dstPath+"+"+dstEnvPath
			log(r,"new xdrc environment %s",dstEnvPath)
	}else{
		log(r,"no xdrc environment file")
	}
	
	// Datacenter list
	r.ParseForm()
	dcFiles := []string{}
	dcUrls := []string{}
	dcnameList:=r.Form["datacenters"]
	log(r,"xdrc on datacenters %v",dcnameList)
	for _,dcname := range dcnameList {
		dcp:=datacenterDirectory(user,dcname)
		if version,err:=latestVersion(dcp); err==nil {
			log(r,"xdrc on datacenter '%s' for version '%s'",dcname,version)
			dcFiles = append(dcFiles,filepath.Join(dcp,version,"topo.yaml"))
			dcUrls = append(dcUrls,datacenterURI(user,dcname)+"?v="+version[1:])
		}else{
			log(r,"Error no version for datacenter '%s'",version)
		}
	}
	ioutil.WriteFile(filepath.Join(dir, "datacenters.urls"),[]byte(strings.Join(dcUrls,",")),0644)
	
	//Read the datacenters
	datacenters := []Datacenter{}
	log(r,"Retrieving datacenters")
	for _,dc := range dcFiles {
		datacenter,err := DatacenterFromFile(dc)
		if err!=nil {
			fmt.Printf("%#v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		
		datacenters = append(datacenters,*datacenter)
	}
	
	log(r,"Datacenters for XDRC:\n%#v",datacenters)
	
	//Build the xdcrs
	var buf bytes.Buffer
	if  errXDCR := XDCRFromFile(dstPath,datacenters,&buf); errXDCR != nil {
		fmt.Printf("%#v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	

	//write json and yaml topo files
	//ToFile(s[0], filepath.Join(dir, "xdcr"))
	
	//write dot topo file
	ioutil.WriteFile(filepath.Join(dir, "xdcr.dot"), []byte(fmt.Sprintf("digraph { \n%s\n}\n", buf.String())), 0777)

	//process dot file to build image
	cmd := exec.Command("dot", "-Tpng", "-o"+filepath.Join(dir, "xdcr.png"), filepath.Join(dir, "xdcr.dot"))
	var out bytes.Buffer
	var outerr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &outerr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error Dot: %#v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	version,_ := latestVersionInt(xdcrDirectory(user))
	uri := fmt.Sprintf("/xdcr?v=%d",version)
	http.Redirect(w, r, uri , http.StatusTemporaryRedirect)
}
func xdcrPage(w http.ResponseWriter, r *http.Request) {
	user:=getuser(r)
	r.ParseForm()
	version := r.Form.Get("v")

		data := struct {
		User           string
		Versions       []int
		Version        string
	}{
		User:           user,
		Versions:       []int{},
		Version:        version,
	}
	renderTemplate(w, "xdcr", data)
}

func experimentTopo(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	data := struct {
		User           string
		Topo		string
		Env string
	}{
		User:           getuser(r),
		Topo: r.Form.Get("topoArea"),
		Env: r.Form.Get("envArea"),				
	}

	
	
	renderTemplate(w, "expTopo", data)
}

func experimentXDCR(w http.ResponseWriter, r *http.Request) {
	data := struct {
		User           string
	}{
		User:           getuser(r),		
	}
	renderTemplate(w, "expXdcr", data)
}

func prepareNextVersion(baseDir string) (string,error) {
	var perm os.FileMode = 0777
	dirbase := baseDir
	os.MkdirAll(dirbase, perm)
	version, err := nextVersion(dirbase)
	if err != nil {
		fmt.Printf("%#v", err)
		return "",err
	}
	dir := filepath.Join(dirbase, version)
	os.MkdirAll(dir, perm)
	return dir,nil
}

func datacenterURI(user, datacenterName string) string {
	return filepath.Join("/data", user, "dc", datacenterName)
}

func xdcrURI(user string) string {
	return filepath.Join("/data", user, "xdcr")
}

func userDirectory(user string) string {
	return filepath.Join("public", "data", user)
}

func datacenterDirectory(user, datacenterName string) string {
	return filepath.Join(userDirectory(user), "dc", datacenterName)
}

func xdcrDirectory(user string) string {
	return filepath.Join(userDirectory(user), "xdcr")
}

func expDirectory(user string) string {
	return filepath.Join(userDirectory(user), "exp")
}

func listUsers() []string {
	users := []string{}
	files, _ := ioutil.ReadDir(filepath.Join("public", "data"))
	for _, file := range files {
		users = append(users, file.Name())
	}
	return users
}
func listXDCR(user string) []int {
	xdcr,_ :=listVersions(xdcrDirectory(user))
	return xdcr
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

func log(r * http.Request,f string, a ...interface{} ) {
	t:= time.Now()
	u:=getuser(r)
	if a!=nil {
		fmt.Printf(fmt.Sprintf("%s %s: %s\n",t.Format(time.RFC3339),u,f),a...)
	}else{
		fmt.Printf("%s %s: %s\n",t.Format(time.RFC3339),u,f)
	}
}