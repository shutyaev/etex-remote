package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Makefile - the etex makefile
type Makefile struct {
	FilesPath   string `yaml:"files_path"`
	FiguresPath string `yaml:"figures_path"`
	StylesPath  string `yaml:"styles_path"`
	OutputPath  string `yaml:"output_path"`
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func getFilesToZip(makefilePath string, makefile Makefile) []string {
	makefileDir := filepath.Dir(makefilePath)
	result := []string{makefilePath}
	result = collectFiles(result, filepath.Join(makefileDir, makefile.FilesPath))
	if makefile.FiguresPath != "" {
		result = collectFiles(result, filepath.Join(makefileDir, makefile.FiguresPath))
	}
	if makefile.StylesPath != "" {
		result = collectFiles(result, filepath.Join(makefileDir, makefile.StylesPath))
	}
	return result
}

func collectFiles(result []string, path string) []string {
	filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				result = append(result, path)
			}
			return nil
		})
	return result
}

func createZip(makefilePath string, makefile Makefile) bytes.Buffer {
	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)
	makefileDir := filepath.Dir(makefilePath)
	for _, file := range getFilesToZip(makefilePath, makefile) {
		// get zip path
		zipPath, err := filepath.Rel(makefileDir, file)
		check(err)
		// open file to zip
		fileToZip, err := os.Open(file)
		check(err)
		defer fileToZip.Close()
		// get file information
		info, err := fileToZip.Stat()
		check(err)
		// create header
		header, err := zip.FileInfoHeader(info)
		check(err)
		header.Name = zipPath
		header.Method = zip.Deflate
		// create item writer
		zipItemWriter, err := zipWriter.CreateHeader(header)
		check(err)
		// copy contents
		_, err = io.Copy(zipItemWriter, fileToZip)
		check(err)
	}
	err := zipWriter.Close()
	check(err)
	return *zipBuffer
}

func main() {
	makefilePath := os.Args[1]
	// read yaml
	data, err := ioutil.ReadFile(makefilePath)
	check(err)
	// parse yaml
	makefile := Makefile{}
	yaml.Unmarshal(data, &makefile)
	zipData := createZip(makefilePath, makefile)
	url := fmt.Sprintf("http://localhost:8000/?makefile_name=%v&output_path=%v", filepath.Base(makefilePath), makefile.OutputPath)
	resp, err := http.Post(url, "application/zip", &zipData)
	check(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	check(err)
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	check(err)
	outputPath := filepath.Join(filepath.Dir(makefilePath), makefile.OutputPath)
	for _, f := range zipReader.File {
		fpath := filepath.Join(outputPath, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
		check(err)
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		check(err)
		rc, err := f.Open()
		check(err)
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		check(err)
	}
}
