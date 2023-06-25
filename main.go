package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	gs "github.com/otiai10/gosseract/v2"
)

var reDate = regexp.MustCompile(`\b(?:0[1-9]|[1-2][0-9]|3[01])[\.\/](?:0[1-9]|1[0-2])[\.\/](?:\d{4}|\d{2})\b`)

type status int

const (
	Undefined status = iota - 1
	Valid
	Invalid
)

var statusNames = map[status]string{
	Undefined: "undefined",
	Valid:     "valid",
	Invalid:   "invalid",
}

type Case struct {
	filepath string
	date     time.Time
	status   status
}

func (c Case) String() string {
	return fmt.Sprintf("filepath: %s, date: %s", c.filepath, c.date)
}

func getCases() []Case {
	cases := []Case{}
	csvFile, _ := os.Open("test.csv")
	defer csvFile.Close()
	r := csv.NewReader(csvFile)
	records, _ := r.ReadAll()
	for _, record := range records[1:] {
		t, err := time.Parse("2006-01-02 15:04:05+03:00", record[1])
		if err != nil {
			log.Fatal(err)
		}
		cases = append(cases, Case{
			filepath: record[0],
			date:     t,
		})
	}
	return cases
}

func main() {
	var (
		testCases      = getCases()
		actualResults  = make([]Case, 0)
		valid          = make([]Case, 0)
		invalid        = make([]Case, 0)
		undefined      = make([]Case, 0)
		testDataFolder = "./test/data/"
		start          = time.Now()
		langs          = "rus"
	)
	err := filepath.Walk(testDataFolder, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}

		image, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		client := gs.NewClient()
		defer client.Close()
		if err = client.SetLanguage(langs); err != nil {
			return err
		}
		if err := client.SetImageFromBytes(image); err != nil {
			return err
		}

		text, err := client.Text()
		if err != nil {
			return err
		}

		c := Case{filepath: path, status: Undefined}
		defer func() {
			actualResults = append(actualResults, c)
		}()

		cleanText := strings.Replace(text, "\n", " ", -1)
		result := reDate.FindAllString(cleanText, -1)

		if len(result) == 0 {
			undefined = append(undefined, c)
			return nil
		}

		dates := make([]time.Time, 0)
		for _, s := range result {
			s = strings.Replace(s, "/", ".", -1)
			pieces := strings.Split(s, ".")
			if len(pieces[2]) == 2 {
				pieces[2] = strconv.Itoa(time.Now().Year())[:2] + pieces[2]
			}
			converted := make([]int, len(pieces))
			var err error
			for i, p := range pieces {
				converted[i], err = strconv.Atoi(p)
				if err != nil {
					return err
				}
			}
			date := time.Date(converted[2], time.Month(converted[1]), converted[0], 0, 0, 0, 0, time.UTC)
			dates = append(dates, date)
		}

		c.status = Invalid
		switch len(dates) {
		case 1:
			c.date = dates[0]
		case 2:
			if dates[0].After(dates[1]) {
				c.date = dates[0]
			} else {
				c.date = dates[1]
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for i, tc := range actualResults {
		if tc.status == Undefined {
			continue
		}
		if testCases[i].date.Equal(tc.date) {
			tc.status = Valid
			valid = append(valid, tc)
			continue
		}
		invalid = append(invalid, tc)
	}
	fmt.Println("Done in", time.Since(start))
	printCases(Valid, valid)
	printCases(Invalid, invalid)
	printCases(Undefined, undefined)
}

func printCases(status status, cases []Case) {
	fmt.Println(statusNames[status], "cases:", len(cases))
	for _, c := range cases {
		fmt.Println("\t", c.filepath, "-", c.date)
	}
}
