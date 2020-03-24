package main

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

type singleProvince []record
type singleRegion map[int]singleProvince
type data struct {
	all           []record
	regions       map[string]singleRegion
	regionNames   []string
	provinces     map[int]singleProvince
	provinceCodes map[string]int
	provinceNames map[int]string
}

func (d *data) casesByProvince(region string) [][]interface{} {
	var (
		provinces      = d.regions[region]
		provincesNames []string
	)
	for provinceID := range provinces {
		provincesNames = append(provincesNames, d.provinceNames[provinceID])
	}

	var (
		dates        = make([]time.Time, 0)
		rowsByDate   = make(map[time.Time][]interface{})
		totalsByDate = make([]int, 0)
	)
	sort.Strings(provincesNames)
	for i, name := range provincesNames {
		province := provinces[d.provinceCodes[name]]
		for _, r := range province {
			if _, found := rowsByDate[r.Date]; !found {
				rowsByDate[r.Date] = make([]interface{}, len(provincesNames))
				totalsByDate = append(totalsByDate, 0)
				dates = append(dates, r.Date)
			}
			rowsByDate[r.Date][i] = r.TotalCases
		}
	}

	// sort dates
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	var rows = make([][]interface{}, len(rowsByDate)+1)
	rows[0] = make([]interface{}, len(provincesNames)+1)
	for i := range provincesNames {
		rows[0][i+1] = provincesNames[i]
	}
	rows[0] = append(rows[0], "Total")

	for i := range dates {
		date := dates[i]
		values := rowsByDate[date]
		row := append([]interface{}{date}, values...)
		var total int
		for _, v := range values {
			vInt, ok := v.(int)
			if !ok {
				continue
			}
			total += vInt
		}
		row = append(row, total)
		rows[i+1] = row
	}
	return rows
}

func recordsFromStrings(strs [][]string) ([]record, error) {
	var records = make([]record, len(strs))
	for i := range strs {
		if i == 0 {
			continue
		}
		r, err := recordFromStrings(strs[i])
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

func dataFromStrings(strs [][]string) (*data, error) {
	records, err := recordsFromStrings(strs)
	if err != nil {
		return nil, err
	}
	regions := make(map[string]singleRegion)
	regionNames := make([]string, 0)
	provinces := make(map[int]singleProvince)
	provinceCodes := make(map[string]int)
	provinceNames := make(map[int]string)
	for i := range records {
		r := records[i]
		if strings.HasPrefix(r.ProvinceName, "In fase di definizione") {
			continue
		}

		p, found := provinces[r.ProvinceID]
		if !found {
			p = make([]record, 0, 1)
		}
		reg, found := regions[r.RegionName]
		if !found {
			reg = make(map[int]singleProvince)
			regions[r.RegionName] = reg
			regionNames = append(regionNames, r.RegionName)
		}
		provinces[r.ProvinceID] = append(p, r)
		provinceCodes[r.ProvinceName] = r.ProvinceID
		provinceNames[r.ProvinceID] = r.ProvinceName
		reg[r.ProvinceID] = provinces[r.ProvinceID]
	}

	sort.Strings(regionNames)
	return &data{
		all:           records,
		regions:       regions,
		provinces:     provinces,
		provinceCodes: provinceCodes,
		provinceNames: provinceNames,
		regionNames:   regionNames,
	}, nil
}

type record struct {
	Date         time.Time `csv:"data"`
	State        string    `csv:"stato"`
	RegionID     int       `csv:"codice_regione"`
	RegionName   string    `csv:"denominazione_regione"`
	ProvinceID   int       `csv:"codice_provincia"`
	ProvinceName string    `csv:"denominazione_provincia"`
	Province     string    `csv:"sigla_provincia"`
	Latitude     float64   `csv:"lat"`
	Longitude    float64   `csv:"long"`
	TotalCases   int       `csv:"totale_casi"`
}

func recordFromStrings(fields []string) (record, error) {
	if len(fields) < 9 {
		return record{}, errors.New("malformed")
	}
	data, err := time.Parse("2006-01-02 15:04:05", fields[0])
	if err != nil {
		return record{}, err
	}

	regionID, err := strconv.Atoi(fields[2])
	if err != nil {
		return record{}, err
	}

	provinceID, err := strconv.Atoi(fields[4])
	if err != nil {
		return record{}, err
	}

	lat, err := strconv.ParseFloat(fields[7], 64)
	if err != nil {
		return record{}, err
	}

	long, err := strconv.ParseFloat(fields[8], 64)
	if err != nil {
		return record{}, err
	}

	total, err := strconv.Atoi(fields[9])
	if err != nil {
		return record{}, err
	}

	return record{
		Date:         data,
		State:        fields[1],
		RegionID:     regionID,
		RegionName:   fields[3],
		ProvinceID:   provinceID,
		ProvinceName: fields[5],
		Province:     fields[6],
		Latitude:     lat,
		Longitude:    long,
		TotalCases:   total,
	}, nil
}
