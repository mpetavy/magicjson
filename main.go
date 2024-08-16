package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	xj "github.com/basgys/goxml2json"
	"github.com/mpetavy/common"
	"github.com/saintfish/chardet"
	"html/template"
	"io"
	"os"
	"strings"
)

//go:embed go.mod
var resources embed.FS

type Pet struct {
	Name   string
	Sex    string
	Intact bool
	Age    string
	Breed  string
}

var (
	inputFile     = flag.String("i", "", "Input filename. Use . to read from STDIN")
	inputEncoding = flag.String("e", "", "Input encoding")
	outputFile    = flag.String("o", "", "Output filename. Omit to print to STDOUT")
	templateFile  = flag.String("t", "", "Template filename or directory")
	clean         = flag.Bool("c", false, "Clean key names")

	dogs = []Pet{
		{
			Name:   "Jujube",
			Sex:    "Female",
			Intact: false,
			Age:    "10 months",
			Breed:  "German Shepherd/Pitbull",
		},
		{
			Name:   "Zephyr",
			Sex:    "Male",
			Intact: true,
			Age:    "13 years, 3 months",
			Breed:  "German Shepherd/Border Collie",
		},
	}
)

type JsonMap map[string]any

func init() {
	common.Init("", "", "", "", "", "", "", "", &resources, nil, nil, run, 0)
}

func ReadBytes(r io.Reader) ([]byte, error) {
	buf := bytes.Buffer{}
	ba := make([]byte, 8192)

	for {
		n, err := r.Read(ba)
		if err == io.EOF {
			return buf.Bytes(), nil
		}
		if common.Error(err) {
			return nil, err
		}

		_, err = buf.Write(ba[:n])
		if common.Error(err) {
			return nil, err
		}
	}
}

func ReadXML(ba []byte) (JsonMap, error) {
	buf, err := xj.Convert(bytes.NewBuffer(ba))
	if common.Error(err) {
		return nil, err
	}

	var result JsonMap

	err = json.Unmarshal(buf.Bytes(), &result)
	if common.Error(err) {
		return nil, err
	}

	return result, nil
}

func ReadHL7(ba []byte) (JsonMap, error) {
	buf, err := xj.Convert(bytes.NewBuffer(ba))
	if common.Error(err) {
		return nil, err
	}

	var result JsonMap

	err = json.Unmarshal(buf.Bytes(), &result)
	if common.Error(err) {
		return nil, err
	}

	return result, nil
}

func cleanKeys(m map[string]interface{}) map[string]interface{} {
	mnew := make(map[string]interface{})

	for k, v := range m {
		k = strings.ReplaceAll(k, "-", "")

		switch x := v.(type) {
		case map[string]interface{}:
			mnew[k] = cleanKeys(x)
		case []interface{}:
			list := make([]interface{}, 0)

			for _, a := range x {
				if sm, ok := a.(map[string]interface{}); ok {
					list = append(list, cleanKeys(sm))
				} else {
					list = append(list, a)
				}
			}

			mnew[k] = list
		default:
			mnew[k] = v
		}
	}

	return mnew
}

func run() error {
	var ba []byte

	if *inputFile == "." {
		var err error

		ba, err = ReadBytes(os.Stdin)
		if common.Error(err) {
			return err
		}
	} else {
		var err error

		ba, err = os.ReadFile(*inputFile)
		if common.Error(err) {
			return err
		}
	}

	encoding := *inputEncoding

	if encoding == "" {
		detector := chardet.NewTextDetector()
		result, err := detector.DetectBest(ba)
		if !common.WarnError(err) {
			encoding = result.Charset
		}
	}

	common.Info("Input encoding: %s", encoding)

	if strings.ToUpper(encoding) != "UTF-8" {
		var err error

		common.Info("Convert to UTF-8")

		ba, err = common.ToUTF8(bytes.NewReader(ba), encoding)
		if common.Error(err) {
			return err
		}
	}

	var jsonObj JsonMap

	switch {
	case strings.HasSuffix(*inputFile, ".json"):
		err := json.Unmarshal(ba, &jsonObj)
		if common.Error(err) {
			return err
		}
	case strings.HasSuffix(*inputFile, ".hl7"):
		var err error

		jsonObj, err = ReadHL7(ba)
		if common.Error(err) {
			return err
		}
	case strings.HasSuffix(*inputFile, ".xml"):
		var err error

		jsonObj, err = ReadXML(ba)
		if common.Error(err) {
			return err
		}
	default:
		return fmt.Errorf("Unsupport file type. only .hl7 .json and .xml allowed")
	}

	org, _ := json.MarshalIndent(jsonObj, "", "    ")
	err := os.WriteFile("/home/ransom/temp/org.json", org, common.DefaultFileMode)
	if common.Error(err) {
		return err
	}

	if *clean {
		jsonObj = cleanKeys(jsonObj)
	}

	mod, _ := json.MarshalIndent(jsonObj, "", "    ")
	err = os.WriteFile("/home/ransom/temp/mod.json", mod, common.DefaultFileMode)
	if common.Error(err) {
		return err
	}

	formattedJson, err := json.MarshalIndent(jsonObj, "", "    ")
	if common.Error(err) {
		return err
	}

	if !bytes.HasSuffix(formattedJson, []byte("\n")) {
		formattedJson = append(formattedJson, []byte("\n")...)
	}

	output := formattedJson

	if *templateFile != "" {
		funcMap := template.FuncMap{
			"GetValue": func(m map[string]any, key string) any {
				return m[key]
			},
		}

		tmpl, err := template.New(*templateFile).Funcs(funcMap).ParseFiles(*templateFile)
		if common.Error(err) {
			return err
		}

		buf := bytes.Buffer{}

		err = tmpl.Execute(&buf, jsonObj)
		if common.Error(err) {
			return err
		}

		output = buf.Bytes()
	}

	if *outputFile == "" {
		fmt.Printf("%s", output)
	} else {
		os.WriteFile(*outputFile, output, common.DefaultFileMode)
	}

	return nil
}

func main() {
	common.Run([]string{"i"})
}
