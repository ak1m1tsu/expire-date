package main

import (
	"encoding/csv"
	"flag"
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

type Status int

const (
	Undefined Status = iota
	Valid
	Invalid
)

var StatusNames = map[Status]string{
	Undefined: "Undefined",
	Valid:     "Valid",
	Invalid:   "Invalid",
}

var reDatetime = regexp.MustCompile(`\b(?:0[1-9]|[1-2][0-9]|3[01])[\.\/](?:0[1-9]|1[0-2])[\.\/](?:\d{4}|\d{2})\b`)

type Product struct {
	Filepath       string
	ExpirationDate time.Time
	Status         Status
}

var testFile = flag.String("f", "expire-dates.csv", "test file")

func testCases() []Product {
	cases := []Product{}
	csvFile, _ := os.Open(*testFile)
	defer csvFile.Close()
	r := csv.NewReader(csvFile)
	records, _ := r.ReadAll()
	for _, record := range records[1:] {
		t, err := time.Parse("2006-01-02 15:04:05+03:00", record[1])
		if err != nil {
			log.Fatal(err)
		}
		cases = append(cases, Product{
			Filepath:       record[0],
			ExpirationDate: t,
		})
	}
	return cases
}

func main() {
	flag.Parse()
	var (
		test           = testCases()
		results        = make([]Product, 0)
		valid          = make([]Product, 0)
		invalid        = make([]Product, 0)
		undefined      = make([]Product, 0)
		testDataFolder = "./test/data"
		langs          = []string{"rus"}
		start          = time.Now()
	)
	if err := filepath.Walk(testDataFolder, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return err
		}

		image, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		client := gs.NewClient()
		defer client.Close()
		if err = client.SetLanguage(langs...); err != nil {
			return err
		}
		if err := client.SetImageFromBytes(image); err != nil {
			return err
		}

		text, err := client.Text()
		if err != nil {
			return err
		}

		product := Product{
			Filepath: path,
			Status:   Undefined,
		}
		defer func() {
			results = append(results, product)
		}()

		cleanText := strings.Replace(text, "\n", " ", -1)
		matches := reDatetime.FindAllString(cleanText, -1)

		if len(matches) == 0 {
			undefined = append(undefined, product)
			return nil
		}

		dates := make([]time.Time, 0)
		for _, match := range matches {
			match = strings.Replace(match, "/", ".", -1)
			pieces := strings.Split(match, ".")
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

		product.Status = Invalid
		switch len(dates) {
		case 1:
			product.ExpirationDate = dates[0]
		case 2:
			if dates[0].After(dates[1]) {
				product.ExpirationDate = dates[0]
			} else {
				product.ExpirationDate = dates[1]
			}
		}
		return nil
	},
	); err != nil {
		log.Fatal(err)
	}
	for i, tc := range results {
		if tc.Status == Undefined {
			continue
		}
		if test[i].ExpirationDate.Equal(tc.ExpirationDate) {
			tc.Status = Valid
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

func printCases(status Status, cases []Product) {
	fmt.Println(StatusNames[status], "cases:", len(cases))
	for _, c := range cases {
		fmt.Println("\t", c.Filepath, "-", c.ExpirationDate)
	}
}
