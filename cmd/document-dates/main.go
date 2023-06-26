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

type Document struct {
	Filepath string
	Dates    []time.Time
	Status   Status
}

var reDatetime = regexp.MustCompile(`\b(?:0[1-9]|[1-2][0-9]|3[01])[\.\/](?:0[1-9]|1[0-2])[\.\/](?:\d{4}|\d{2})\b`)

var testFile = flag.String("f", "/home/ak1m1tsu/Code/expiration-date-fetcher/cmd/document-dates/documents.csv", "test file")

func testCases() []Document {
	cases := []Document{}
	csvFile, _ := os.Open(*testFile)
	defer csvFile.Close()
	r := csv.NewReader(csvFile)
	records, _ := r.ReadAll()
	for _, record := range records[1:] {
		t, err := time.Parse("2006-01-02 15:04:05+03:00", record[1])
		if err != nil {
			log.Fatal(err)
		}
		t2, err := time.Parse("2006-01-02 15:04:05+03:00", record[2])
		if err != nil {
			log.Fatal(err)
		}
		cases = append(cases, Document{
			Filepath: record[0],
			Dates:    []time.Time{t, t2},
		})
	}
	return cases
}

func main() {
	flag.Parse()
	var (
		test           = testCases()
		results        = make([]Document, 0)
		testDataFolder = "/home/ak1m1tsu/Code/expiration-date-fetcher/test/document"
		langs          = []string{"rus"}
		start          = time.Now()
	)
	err := filepath.Walk(
		testDataFolder,
		func(path string, info os.FileInfo, err error) error {
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

			c := Document{
				Filepath: path,
				Status:   Undefined,
			}
			defer func() {
				results = append(results, c)
			}()

			cleanText := strings.Replace(text, "\n", " ", -1)
			matches := reDatetime.FindAllString(cleanText, -1)

			if len(matches) == 0 {
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

			c.Dates = dates
			c.Status = Invalid
			return nil
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	for i, doc := range results {
		if len(doc.Dates) == 0 {
			continue
		}
		if test[i].Dates[0].Equal(doc.Dates[0]) && test[i].Dates[1].Equal(doc.Dates[1]) {
			results[i].Status = Valid
		}
	}
	fmt.Println("Done in", time.Since(start))
	printDocuments(results)
}

func printDocuments(docs []Document) {
	for _, doc := range docs {
		fmt.Println("\t", doc.Filepath, StatusNames[doc.Status])
		for _, date := range doc.Dates {
			fmt.Println("\t\t", date)
		}
	}
}
