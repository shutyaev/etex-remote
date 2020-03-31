package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestMakefileStruct(t *testing.T) {
	data, err := ioutil.ReadFile(filepath.Join("testdata", "sample.yaml"))
	check(err)
	makefile := Makefile{}
	yaml.Unmarshal(data, &makefile)
	errorMessage := "Found differences at %v"
	assertEquals("md_sources", makefile.FilesPath, fmt.Sprintf(errorMessage, "FilesPath"), t)
	assertEquals("images", makefile.FiguresPath, fmt.Sprintf(errorMessage, "FiguresPath"), t)
	assertEquals("styles", makefile.StylesPath, fmt.Sprintf(errorMessage, "StylesPath"), t)
	assertEquals("text", makefile.OutputPath, fmt.Sprintf(errorMessage, "OutputPath"), t)
}

func TestGetFilesToZip(t *testing.T) {
	makefilePath := filepath.Join("testdata", "sample.yaml")
	makefile := sampleMakefile()
	actual := getFilesToZip(makefilePath, makefile)
	expected := []string{
		filepath.Join("testdata", "sample.yaml"),
		filepath.Join("testdata", "md_sources", "0_front_page.md"),
		filepath.Join("testdata", "md_sources", "1_text.md"),
		filepath.Join("testdata", "md_sources", "2_footnotes.md"),
		filepath.Join("testdata", "md_sources", "3_colophon.md"),
		filepath.Join("testdata", "images", "bar", "baz.txt"),
		filepath.Join("testdata", "images", "foo.txt"),
		filepath.Join("testdata", "styles", "colophon.css"),
		filepath.Join("testdata", "styles", "common.css"),
		filepath.Join("testdata", "styles", "footnotes.css"),
		filepath.Join("testdata", "styles", "front_page.css")}
	assertSliceEquals(expected, actual, "Found difference in filesToZip%v", t)
}

func TestCreateZip(t *testing.T) {
	makefilePath := filepath.Join("testdata", "sample.yaml")
	makefile := sampleMakefile()
	actual := createZip(makefilePath, makefile)
	expected, err := ioutil.ReadFile(filepath.Join("testdata", "sample.zip"))
	check(err)
	if bytes.Compare(expected, actual.Bytes()) != 0 {
		t.Errorf("The actual created zip doesn't match the expected one")
	}
}

func TestUnzip(t *testing.T) {
	data, err := ioutil.ReadFile(filepath.Join("testdata", "testUnzip.zip"))
	check(err)
	dir, err := ioutil.TempDir("", "example")
	check(err)
	defer os.RemoveAll(dir)
	unzip(data, dir)
	actual, err := ioutil.ReadFile(filepath.Join(dir, "foo.txt"))
	check(err)
	assertEquals("Hello, world!", string(actual), "foo.txt comparison failed", t)
	actual, err = ioutil.ReadFile(filepath.Join(dir, "bar", "baz.txt"))
	check(err)
	assertEquals("Goodbye, world!", string(actual), "bar/baz.txt comparison failed", t)
}

func sampleMakefile() Makefile {
	result := Makefile{}
	result.FilesPath = "md_sources"
	result.FiguresPath = "/images"
	result.StylesPath = "/styles"
	result.OutputPath = "/text"
	return result
}

func assertEquals(expected interface{}, actual interface{}, message string, t *testing.T) {
	if expected != actual {
		t.Errorf("%v, expected %v, but was %v", message, expected, actual)
	}
}

func assertSliceEquals(expected interface{}, actual interface{}, message string, t *testing.T) {
	expectedSlice := reflect.ValueOf(expected)
	actualSlice := reflect.ValueOf(actual)
	expectedLen := expectedSlice.Len()
	actualLen := actualSlice.Len()
	if expectedLen != actualLen {
		t.Errorf("%v, expected slice of len %v, but was %v", fmt.Sprintf(message, ""), expectedLen, actualLen)
	}
	for i := 0; i < expectedLen; i++ {
		expectedValue := expectedSlice.Index(i).Interface()
		actualValue := actualSlice.Index(i).Interface()
		assertEquals(expectedValue, actualValue, fmt.Sprintf(message, fmt.Sprintf("[%v]", i)), t)
	}
}

func assertMapEquals(expected interface{}, actual interface{}, message string, t *testing.T) {
	expectedMap := reflect.ValueOf(expected)
	actualMap := reflect.ValueOf(actual)
	expectedLen := expectedMap.Len()
	actualLen := actualMap.Len()
	if expectedLen != actualLen {
		t.Errorf("%v, expected map of len %v, but was %v", fmt.Sprintf(message, ""), expectedLen, actualLen)
	}
	iter := expectedMap.MapRange()
	for iter.Next() {
		key := iter.Key()
		expectedValue := iter.Value().Interface()
		actualValue := actualMap.MapIndex(key).Interface()
		switch iter.Value().Kind() {
		case reflect.Slice:
			assertSliceEquals(expectedValue, actualValue, fmt.Sprintf(message, fmt.Sprintf("[%v]%%v", key)), t)
		default:
			assertEquals(expectedValue, actualValue, fmt.Sprintf(message, fmt.Sprintf("[%v]", key)), t)
		}
	}
}
