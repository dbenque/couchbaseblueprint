package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
)

func main() {

	//gen_sample1()
	gen_RBox1()

}

func ToFile(v interface{}, filePath string) {
	b, _ := json.Marshal(v)
	var out bytes.Buffer
	json.Indent(&out, b, " ", "\t")
	ioutil.WriteFile(filePath, out.Bytes(), 0777)
}
