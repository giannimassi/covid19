package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, "Error: "+err.Error()+"\n")
	}
}

var update = flag.Bool("update", false, "update data from github")

func run() error {
	flag.Parse()
	data, err := loadData(*update)
	if err != nil {
		return err
	}

	rd := csv.NewReader(data)
	rows, err := rd.ReadAll()
	if err != nil {
		return err
	}
	italy, err := dataFromStrings(rows)
	if err != nil {
		return err
	}

	client, err := getGoogleSheetsClient()
	if err != nil {
		return fmt.Errorf("while getting google sheet client: %w", err)
	}

	err = writeToGoogleSheets(client, "1G2E9PF7ZJ0YGI8Fc0niZ7nHFM3i4IslQznTpQEbXkR0", "data", "data!A1", italy.casesByProvince("Toscana"))
	if err != nil {
		return fmt.Errorf("while writing to google sheet (id:%s): %v", "1G2E9PF7ZJ0YGI8Fc0niZ7nHFM3i4IslQznTpQEbXkR0", err)
	}

	id := os.Getenv("COVID_GOOGLESHEET")

	for _, region := range italy.regionNames {
		fmt.Println("Updating", region)
		if region == "" {
			continue
		}
		err = writeToGoogleSheets(client, id, region, region+"!A1", italy.casesByProvince(region))
		if err != nil {
			return fmt.Errorf("while writing to google sheet (id:%s): %v", id, err)
		}
	}

	return nil
}

func loadData(update bool) (io.Reader, error) {
	if update {
		return loadFromGithub()
	}
	return loadFromFile()
}

func loadFromGithub() (io.Reader, error) {
	const provinceURL = `https://raw.githubusercontent.com/pcm-dpc/COVID-19/master/dati-province/dpc-covid19-ita-province.csv`
	response, err := http.Get(provinceURL)
	if err != nil {
		return nil, fmt.Errorf("while querying latest data: %w", err)
	}
	if response.StatusCode != 200 {
		return nil, errors.New("bad response: " + response.Status)
	}
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, response.Body); err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(dataFile(), buf.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("while writing cached data file: %w", err)
	}
	return buf, err
}

func loadFromFile() (io.Reader, error) {
	byts, err := ioutil.ReadFile(dataFile())
	if err != nil {
		return nil, fmt.Errorf("while reading cached data file: %w", err)
	}
	return bytes.NewReader(byts), nil
}

func dataFile() string {
	const dataFileName = `/dpc-covid19-ita-province.csv`
	base, _ := os.UserCacheDir()
	return filepath.Join(base, dataFileName)
}
