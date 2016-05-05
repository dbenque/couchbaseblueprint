package web

import (
	"html/template"
	"time"

	"github.com/gorilla/mux"

	"net/http"
)

func ServeHTTP() {
	templates = template.Must(template.New("abc").Funcs(fns).ParseGlob("web/template/*.html"))
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