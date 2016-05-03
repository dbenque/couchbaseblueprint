package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"couchbasebp/api"
)

func ToFile(v interface{}, filePath string) {
	//json
	{
		b, _ := json.Marshal(v)
		var out bytes.Buffer
		json.Indent(&out, b, " ", "\t")
		ioutil.WriteFile(filePath+".json", out.Bytes(), 0777)
	}
	//yaml
	{
		y, _ := yaml.Marshal(v)
		ioutil.WriteFile(filePath+".yaml", y, 0777)

	}
}

func DatacenterFromFile(file string) (*api.Datacenter, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	format := strings.Split(file, ".")[1]
	var datacenter api.Datacenter

	switch format {
	case "json":
		err = json.Unmarshal(b, &datacenter)
	case "yaml":
		err = yaml.Unmarshal(b, &datacenter)
	}
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &datacenter, nil

}

func TopoFromFile(files string, DCs []api.Datacenter, w io.Writer) ([]api.Datacenter, error) {

	splitted := strings.Split(files, "+")
	file := splitted[0]
	format := strings.Split(file, ".")[1]
	if len(splitted) == 2 {
		envFile := splitted[1]
		var err error
		file, err = processEnv(file, envFile)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		tmpFile := file
		defer os.Remove(tmpFile)
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var cgdefBlueprint api.ClusterGroupDefBluePrint
	switch format {
	case "json":
		err = json.Unmarshal(b, &cgdefBlueprint)
	case "yaml":
		err = yaml.Unmarshal(b, &cgdefBlueprint)
	}
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	for _, d := range cgdefBlueprint.ClusterGroups {
		for i := range DCs {
			DCs[i].AddClusterGroupDef(d)
		}
	}

	for i := range DCs {
		DCs[i].Dot(w)
	}

	return DCs, nil
}

func XDCRFromFile(files string, DCs []api.Datacenter, w io.Writer) error {
	splitted := strings.Split(files, "+")
	file := splitted[0]
	format := strings.Split(file, ".")[1]
	if len(splitted) == 2 {
		envFile := splitted[1]
		var err error
		file, err = processEnv(file, envFile)
		if err != nil {
			fmt.Println(err)
			return err
		}
		tmpFile := file
		defer os.Remove(tmpFile)
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return err
	}

	var xdcrdefBlueprint api.XDCRDefBluePrint
	switch format {
	case "json":
		err = json.Unmarshal(b, &xdcrdefBlueprint)
	case "yaml":
		err = yaml.Unmarshal(b, &xdcrdefBlueprint)
	}
	if err != nil {
		fmt.Println(err)
		return err
	}

	for _, xdcr := range xdcrdefBlueprint.XDCRDefs {
		for _, x := range api.NewXDCR(xdcr, DCs) {
			x.Dot(w)
		}
	}
	return nil
}

func processEnv(file, envfile string) (string, error) {

	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	format := strings.Split(envfile, ".")[1]
	benv, err := ioutil.ReadFile(envfile)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	var envData api.EnvData
	switch format {
	case "json":
		err = json.Unmarshal(benv, &envData)
	case "yaml":
		err = yaml.Unmarshal(benv, &envData)
	}
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	str := string(b)

	for k, v := range envData.Replacements {
		str = strings.Replace(str, k, v, -1)
	}

	tfile, err := ioutil.TempFile("./", "tmp_")
	defer tfile.Close()
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	_, err = tfile.WriteString(str)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return tfile.Name(), nil

}
