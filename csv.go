package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func main() {
	fmt.Println("Парсер логов грейлога")
	fmt.Println()
	fmt.Println("Сформируйте csv из грейлога. для этого зайдите по адресу")
	fmt.Println("выберите сохраненный запрос: \"ВГ timer\"")
	fmt.Println("More actions -> Export as CSV")
	fmt.Println("Укажите путь до файла:")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		text := scanner.Text()
		if len(text) != 0 {
			parseCsvFile(text)
			break
		} else {
			break
		}

	}
	if scanner.Err() != nil {
		fmt.Println("Error: ", scanner.Err())
	}

	//parseCsvFile("graylog-search-result-absolute-2023-03-03T21_00_00.000Z-2023-03-07T20_59_59.000Z.csv")
}

type ChartField struct {
	key  string
	val  float64
	date time.Time
}

func parseCsvFile(filePath string) {
	pattern, _ := regexp.Compile(`(?P<DATETIME>\d{4}\-\d{2}\-\d{2}\s\d{2}\:\d{2}\:\d{2}).+timer[\:](?P<KEY>.+)[\:]\s(?P<VALUE>[\-]*\d+)`)

	fout, _ := os.Create("parsed_" + filePath)
	w := csv.NewWriter(fout)
	defer w.Flush()

	// Load a csv file.
	f, _ := os.Open(filePath)
	r := csv.NewReader(f)
	r.Read()

	items := make([]ChartField, 0)
	for {
		record, err := r.Read()
		// Stop at EOF.
		if err == io.EOF {
			break
		}

		if err != nil {
			panic(err)
		}

		match := pattern.FindStringSubmatch(record[len(record)-1])
		matchIndex := pattern.SubexpIndex("DATETIME")
		if matchIndex > len(match) {
			fmt.Println("error in line:" + record[len(record)-1])
		} else {
			dateTimeString := match[matchIndex]
			parsedDateTime, err := time.Parse("2006-01-02 15:04:05", dateTimeString)
			if err != nil {
				panic(err)

			}
			key := match[pattern.SubexpIndex("KEY")]

			value, _ := strconv.ParseFloat(match[pattern.SubexpIndex("VALUE")], 64)

			items = append(items, ChartField{key, value, parsedDateTime})
		}
	}

	// sort data by date ank key
	sort.SliceStable(items, func(i, j int) bool {
		idate := dateFormat(items[i].date)
		jdate := dateFormat(items[j].date)
		switch {
		case items[i].key != items[j].key:
			return items[i].key < items[j].key

		case idate != jdate:
			return items[i].date.Before(items[j].date)

		default:
			return items[i].key < items[j].key
		}
	})

	groupMap := make(map[string]map[string]float64)

	for _, v := range items {
		date := dateFormat(v.date)

		addToGroupMap(groupMap, v.key, date, v.val)

	}

	// create a new bar instance
	bar := charts.NewBar()
	bar.SetGlobalOptions(charts.WithTitleOpts(opts.Title{
		Title: "Время выполнения этапов создания документа, мс",
	}))
	axises := make([]string, 0)

	for key, catVal := range groupMap {

		bars := make([]opts.BarData, 0)

		for k, v := range catVal {
			if !contains(axises, k) {
				axises = append(axises, k)
			}
			bars = append(bars, opts.BarData{Value: v, Name: k, Tooltip: &opts.Tooltip{Show: true}})
		}
		bar.AddSeries(key, bars)
	}
	// Put data into instance
	bar.SetXAxis(axises)

	// Where the magic happens
	f, err := os.Create(strings.TrimSuffix(filePath, ".csv") + ".html")

	if err != nil {
		panic(err)
	}

	bar.Render(f)

}

func contains(axises []string, k string) bool {
	for _, v := range axises {
		if v == k {
			return true
		}
	}
	return false
}

func addToGroupMap(m map[string]map[string]float64, date string, category string, value float64) {
	// init date
	_, ok := m[date]
	if !ok {
		m[date] = make(map[string]float64)
	}
	// init cat
	_, okcat := m[date][category]
	if !okcat {
		m[date][category] = 0
	}
	// set value
	m[date][category] += value
}

func dateFormat(date time.Time) string {
	return strconv.Itoa(date.Year()) + "-" + date.Month().String() + "-" + strconv.Itoa(date.Day())
}
