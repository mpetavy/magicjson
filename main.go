package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	xj "github.com/basgys/goxml2json"
	"github.com/mpetavy/common"
	"github.com/paulrosania/go-charset/charset"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed go.mod
var resources embed.FS

var (
	inputFile      = flag.String("i", "", "Input filename. Use . to read from STDIN")
	inputEncoding  = flag.String("ie", "", "Input encoding")
	outputFile     = flag.String("o", "", "Output filename. Omit to print to STDOUT")
	outputEncoding = flag.String("oe", "", "Output encoding")
	templateFile   = flag.String("t", "", "Template filename, directory or direct template content")
	clean          = flag.Bool("c", false, "Clean key names")

	hl7Doc *Hl7Message
)

type JsonMap map[string]any

func init() {
	common.Init("", "", "", "", "", "", "", "", &resources, nil, nil, run, 0)
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
	var err error

	hl7Doc, err = NewHL7Message(ba)
	if common.Error(err) {
		return nil, err
	}

	ba, err = json.MarshalIndent(hl7Doc, "", "    ")
	if common.Error(err) {
		return nil, err
	}

	var result JsonMap

	err = json.Unmarshal(ba, &result)
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

		ba, err = io.ReadAll(os.Stdin)
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

	var err error

	ba, err = common.ToUTF8(ba, *inputEncoding)
	if common.Error(err) {
		return err
	}

	var jsonObj JsonMap
	var funcMap template.FuncMap

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

		funcMap = template.FuncMap{
			"GetValue": func(location string) (any, error) {
				v, err := hl7Doc.GetValue(location)
				if common.Error(err) {
					return nil, err
				}

				return v, nil
			},
		}
	case strings.HasSuffix(*inputFile, ".xml"):
		var err error

		jsonObj, err = ReadXML(ba)
		if common.Error(err) {
			return err
		}

		funcMap = template.FuncMap{
			"GetValue": func(m map[string]any, key string) any {
				return m[key]
			},
		}
	default:
		return fmt.Errorf("Unsupport file type. only .hl7 .json and .xml allowed")
	}

	if *clean {
		jsonObj = cleanKeys(jsonObj)
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
		var tmpl *template.Template

		if common.FileExists(*templateFile) {
			var err error

			tmpl, err = template.New(filepath.Base(*templateFile)).Funcs(funcMap).ParseFiles(*templateFile)
			if common.Error(err) {
				return err
			}
		} else {
			tmpl, err = template.New("template").Funcs(funcMap).Parse(*templateFile)
			if common.Error(err) {
				return err
			}
		}

		buf := bytes.Buffer{}

		if hl7Doc != nil {
			err = tmpl.Execute(&buf, hl7Doc)
			if common.Error(err) {
				return err
			}
		} else {
			err = tmpl.Execute(&buf, jsonObj)
			if common.Error(err) {
				return err
			}
		}

		output = buf.Bytes()
	}

	if *outputEncoding != "" && *outputEncoding != "UTF-8" {
		buf := bytes.Buffer{}

		w, err := charset.NewWriter(*outputEncoding, &buf)
		if common.Error(err) {
			return err
		}

		_, err = w.Write(output)
		if common.Error(err) {
			return err
		}

		err = w.Close()
		if common.Error(err) {
			return err
		}

		output = buf.Bytes()
	}

	if *outputFile == "" {
		fmt.Printf("%s", output)
	} else {
		err := os.WriteFile(*outputFile, output, common.DefaultFileMode)
		if common.Error(err) {
			return err
		}
	}

	return nil
}

func main() {
	common.Run([]string{"i"})
}
