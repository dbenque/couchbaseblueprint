package web

import (
	"bytes"
	"couchbasebp/api"
	"couchbasebp/utils"
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

	"gopkg.in/yaml.v2"
)

func ServeHTTP() {
	templates = template.Must(template.New("abc").Funcs(fns).ParseGlob("public/template/*.html"))
	r := mux.NewRouter()
	r.HandleFunc("/main", mainPage)
	r.HandleFunc("/", mainPage)
	r.HandleFunc("/users", UserHandler(usersPage))
	r.HandleFunc("/deleteuser/{user}", deleteUserPage)
	r.HandleFunc("/datacenters", UserHandler(datacentersPage))
	r.HandleFunc("/deletedatacenter/{datacenterName}/{version}", UserHandler(deleteDatacenterPage))
	r.HandleFunc("/deletedatacenter/{datacenterName}", UserHandler(deleteDatacenterPage))
	r.HandleFunc("/datacenter/{datacenterName}", UserHandler(dcPage))
	r.HandleFunc("/newdatacenter", UserHandler(newDatacenterPage))
	r.HandleFunc("/uploadTopo/datacenter/{dcname}", UserHandler(dcUploadTopo))
	r.HandleFunc("/xdcr", UserHandler(xdcrPage))
	r.HandleFunc("/deletexdcr/{version}", UserHandler(deleteXDCRPage))
	r.HandleFunc("/uploadxdcr", UserHandler(uploadxdcr))
	r.HandleFunc("/experiment/topo", UserHandler(experimentTopo))
	r.HandleFunc("/experiment/topopush", UserHandler(experimentTopopush))
	r.HandleFunc("/experiment/xdcr", UserHandler(experimentXDCR))
	r.HandleFunc("/experiment/xdcrsave", UserHandler(experimentXDCRsave))

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
	http.Handle("/", r)
	http.ListenAndServe(":1323", nil)
}

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
	"imgExpTopo": func(user string) string {
		return filepath.Join(experimentURI(user), "topo.png")
	},
	"imgExpXDCR": func(user string) string {
		return filepath.Join(experimentURI(user), "xdcr.png")
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
	"lastVersionDC": func(user, dcname string) int {
		v, _ := latestVersionInt(datacenterDirectory(user, dcname))
		return v
	},
}

type linkdc struct {
	Uri  string
	Text string
}

type userData struct {
	User string
}

func getuser(r *http.Request) string {
	user := ""
	//check cookie
	if c, err := r.Cookie("user"); err == nil {
		user = c.Value
	}
	return user
}

func UserHandler(f func(http.ResponseWriter,
	*http.Request, string)) func(http.ResponseWriter,
	*http.Request) {
	return func(w http.ResponseWriter,
		r *http.Request) {
		user := getuser(r)
		if user == "" {
			http.Redirect(w, r, "/main", http.StatusTemporaryRedirect)
			return
		}
		f(w, r, user)
	}
}

func xdcrDatacenterLinks(user, pathToDatacentersTxt string) []linkdc {

	links := []linkdc{}
	b, err := ioutil.ReadFile(pathToDatacentersTxt)
	if err != nil {
		fmt.Printf("Error:%#v", err)
		return links
	}

	for _, dctxt := range strings.Split(string(b), ",") {
		token := strings.Split(dctxt, ":")
		uri := filepath.Join("/topo", user, "datacenter", token[0]) + "v=?" + token[1]
		str := token[0] + " " + token[1]
		links = append(links, linkdc{uri, str})
	}

	return links
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	user := getuser(r)
	if user == "" {
		//check uri
		r.ParseForm()
		userURL := r.Form.Get("user")
		if userURL != "" {
			user = userURL
			expiration := time.Now().Add(24 * time.Hour)
			cookie := http.Cookie{Name: "user", Value: user, Expires: expiration}
			http.SetCookie(w, &cookie)
		}
	}
	data := struct {
		User string
	}{
		User: user,
	}
	renderTemplate(w, "mainPage", data)
}

func usersPage(w http.ResponseWriter, r *http.Request, user string) {
	data := struct {
		User string
	}{
		User: user,
	}
	renderTemplate(w, "users", data)
}

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

func deleteUserPage(w http.ResponseWriter, r *http.Request) {
	os.RemoveAll(userDirectory(mux.Vars(r)["user"]))
	http.Redirect(w, r, "/users", http.StatusTemporaryRedirect)
}

func deleteXDCRPage(w http.ResponseWriter, r *http.Request, user string) {
	version := mux.Vars(r)["version"]
	os.RemoveAll(filepath.Join(xdcrDirectory(user), version))
	http.Redirect(w, r, "/xdcr", http.StatusTemporaryRedirect)
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

func uploadFile(r *http.Request, formField, destinationFilePath string) error {
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

var topoFileList = []string{"topo.png", "topo.yaml", "topo.json", "topo.dot", "topodef.yaml", "topoenv.yaml"}
var xdcrFileList = []string{"xdcr.png", "xdcrdef.yaml", "xdcrenv.yaml", "datacenters.txt"}

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
