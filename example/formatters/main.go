// package main is an example program to demonstrate how to use custom column formatters.
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"go.enc.dev/multiwriter"
)

var columns = []string{"name", "department", "size", "color"}
var records = [][]string{
	[]string{"Bob", "Engineering", "10", "blue"},
	[]string{"Sally", "Engineering", "1000", "orange"},
	[]string{"Vivek", "Leadership", "23129", "purple"},
}

func main() {
	nameFormatter := multiwriter.BasicFormatter{"** %s **"}
	departmentFormatter := multiwriter.FuncFormatter(func(str string) string {
		return strings.ToUpper(str)
	})
	sizeFormatter := multiwriter.FuncFormatter(func(str string) string {
		fl, err := strconv.ParseFloat(str, 10)
		if err != nil {
			log.Printf("could not parse float: %s, skipping formatting", str)
			return str
		}
		return fmt.Sprintf("%.4f", fl*1.9074)
	})

	writer := multiwriter.New(
		os.Stdout,
		columns,
		multiwriter.TextFormat,
		multiwriter.WithFormatter(columns[0], nameFormatter),
		multiwriter.WithFormatter(columns[1], departmentFormatter),
		multiwriter.WithFormatter(columns[2], sizeFormatter),
	)
	defer func(w *multiwriter.Writer) {
		w.Flush()
		if err := w.Error(); err != nil {
			log.Fatal(err)
		}
	}(writer)

	for i, record := range records {
		if err := writer.Write(record); err != nil {
			log.Printf("could not write record %d", i)
		}
	}
}
