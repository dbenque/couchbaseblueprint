package web

import (
	"couchbasebp/api"
	"couchbasebp/utils"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

func prepareNextVersion(baseDir string) (string, string, error) {
	var perm os.FileMode = 0777
	dirbase := baseDir
	os.MkdirAll(dirbase, perm)
	version, err := nextVersion(dirbase)
	if err != nil {
		fmt.Printf("%#v", err)
		return "", "", err
	}
	dir := filepath.Join(dirbase, version)
	os.MkdirAll(dir, perm)
	return dir, version, nil
}

func datacenterURI(user, datacenterName string) string {
	return filepath.Join("/data", user, "dc", datacenterName)
}

func xdcrURI(user string) string {
	return filepath.Join("/data", user, "xdcr")
}

func experimentURI(user string) string {
	return filepath.Join("/data", user, "experiment")
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

func experimentDirectory(user string) string {
	return filepath.Join(userDirectory(user), "experiment")
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
	xdcr, _ := listVersions(xdcrDirectory(user))
	return xdcr
}

func listDatacenter(user string) []string {
	dcs := []string{}
	files, _ := ioutil.ReadDir(filepath.Join(userDirectory(user), "dc"))
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

func log(r *http.Request, f string, a ...interface{}) {
	t := time.Now()
	u := getuser(r)
	if a != nil {
		fmt.Printf(fmt.Sprintf("%s %s: %s\n", t.Format(time.RFC3339), u, f), a...)
	} else {
		fmt.Printf("%s %s: %s\n", t.Format(time.RFC3339), u, f)
	}
}

func getDatacenter(user, datacenterName, version string) (*api.Datacenter, error) {
	_, err := strconv.Atoi(version)
	if err != nil {
		return nil, fmt.Errorf("Version string must represent an int: %v", err)
	}
	return utils.DatacenterFromFile(filepath.Join(datacenterDirectory(user, datacenterName), "v"+version, "topo.yaml"))
}
