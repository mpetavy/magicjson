package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestClean(t *testing.T) {
	m0 := make(map[string]any)
	m1 := make(map[string]any)
	m2 := make(map[string]any)

	m2["-a"] = "a"
	m1["-b"] = m2
	m0["-c"] = m1

	x := cleanKeys(m0)

	o, _ := json.MarshalIndent(x, "", "    ")

	fmt.Printf("%s\n", o)
}
