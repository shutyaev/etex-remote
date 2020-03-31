package main

import (
	"archive/zip"
	"bytes"
	"flag"
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
		zipPath, err := filepath.Rel(makefileDir, file)
		check(err)
		fileToZip, err := os.Open(file)
		check(err)
		defer fileToZip.Close()
		info, err := fileToZip.Stat()
		check(err)
		header, err := zip.FileInfoHeader(info)
		check(err)
		header.Name = zipPath
		header.Method = zip.Deflate
		zipItemWriter, err := zipWriter.CreateHeader(header)
		check(err)
		_, err = io.Copy(zipItemWriter, fileToZip)
		check(err)
	}
	err := zipWriter.Close()
	check(err)
	return *zipBuffer
}

func callEtexServer(host string, port int, makefileName string, outputPath string, data bytes.Buffer) []byte {
	url := fmt.Sprintf("http://%v:%v/?makefile_name=%v&output_path=%v", host, port, makefileName, outputPath)
	resp, err := http.Post(url, "application/zip", &data)
	check(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	check(err)
	return body
}

func unzip(data []byte, dest string) {
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	check(err)
	for _, f := range zipReader.File {
		fpath := filepath.Join(dest, f.Name)
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

func main() {
	hostname := flag.String("host", "localhost", "host of the etex-server")
	port := flag.Int("port", 8000, "port of the etex-server")
	flag.Parse()
	makefilePath := flag.Args()[0]
	data, err := ioutil.ReadFile(makefilePath)
	check(err)
	makefile := Makefile{}
	yaml.Unmarshal(data, &makefile)
	zipData := createZip(makefilePath, makefile)
	body := callEtexServer(*hostname, *port, filepath.Base(makefilePath), makefile.OutputPath, zipData)
	outputPath := filepath.Join(filepath.Dir(makefilePath), makefile.OutputPath)
	unzip(body, outputPath)
}
