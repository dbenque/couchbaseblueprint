package web

import (
	"html/template"
	"net/http"
	"path/filepath"
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

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
