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

	"net/http"

	"github.com/labstack/echo"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func dcTopoPageForm(c echo.Context) error {
	u := c.QueryParam("name")
	d := c.QueryParam("datacenterName")
	fmt.Printf("USer %s, datacenter %s\n", u, d)
	return c.Redirect(http.StatusMovedPermanently, "/topo/"+u+"/datacenter/"+d)
}

func dcTopoPage(c echo.Context) error {
	data := struct {
		User           string
		DatacenterName string
	}{
		User:           c.Param("user"),
		DatacenterName: c.Param("datacenterName"),
	}
	return c.Render(http.StatusOK, "uploadTopoDC", data)
}

func dcUploadTopo(c echo.Context) error {
	user := c.FormValue("user")
	datacenterName := c.FormValue("datacenterName")

	// Prepare Folder for next topo
	var perm os.FileMode = 0777
	dirDc := filepath.Join("data", user, "dc", datacenterName)
	os.MkdirAll(dirDc, perm)
	v, err := nextVersion(dirDc)
	if err != nil {
		fmt.Printf("%#v", err)
		return err
	}
	dir := filepath.Join(dirDc, v)
	os.MkdirAll(dir, perm)

	// Source
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// Destination
	dstPath := filepath.Join(dir, "topodef.yaml")
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	// creation of VDatacenter topology target
	var buf bytes.Buffer
	errDc, s := TopoFromFile(dstPath, []Datacenter{NewDatacenter(datacenterName)}, &buf)
	if errDc != nil {
		return errDc
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
		return err
	}

	return c.HTML(http.StatusOK, fmt.Sprintf("<p>File %s uploaded successfully with fields user=%s and datacenter=%s.</p>", file.Filename, user, datacenterName))
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
