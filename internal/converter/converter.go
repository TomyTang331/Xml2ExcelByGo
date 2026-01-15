package converter

import (
	"fmt"
	"sort"

	"github.com/TomyTang331/Xml2ExcelByGo/internal/config"
	"github.com/TomyTang331/Xml2ExcelByGo/internal/parser"
	"github.com/TomyTang331/Xml2ExcelByGo/internal/writer"
)

type Converter struct {
	bufferSize int
	batchSize  int
}

func NewConverter(bufferSize int) *Converter {
	return &Converter{
		bufferSize: bufferSize,
		batchSize:  config.DefaultBatchSize,
	}
}

func (c *Converter) Convert(inputFile, outputFile string) error {
	p := parser.NewParser(c.bufferSize)
	dataChan, errChan := p.ParseToRows(inputFile)

	headers, dataRows := c.collectHeadersAndData(dataChan, config.HeaderSampleSize)

	if len(headers) == 0 {
		return fmt.Errorf("no data columns found")
	}

	fmt.Printf("Detected %d columns: %v\n", len(headers), headers)

	excelWriter := writer.NewExcelWriter(outputFile, c.batchSize)
	defer excelWriter.Close()

	if err := excelWriter.CreateSheet("Sheet1", headers); err != nil {
		return fmt.Errorf("failed to create sheet: %w", err)
	}

	for _, row := range dataRows {
		if err := excelWriter.WriteRow("Sheet1", row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	count := len(dataRows)
	for data := range dataChan {
		if err := excelWriter.WriteRow("Sheet1", data); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
		count++
		if count%1000 == 0 {
			fmt.Printf("  Processed %d rows...\n", count)
		}
	}

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	fmt.Printf("✓ Data writing completed: %d rows\n", count)
	fmt.Println("\nSaving file...")
	return nil
}

func (c *Converter) collectHeadersAndData(dataChan <-chan map[string]string, sampleSize int) ([]string, []map[string]string) {
	headerSet := make(map[string]bool)
	var dataRows []map[string]string

	count := 0
	for data := range dataChan {
		dataRows = append(dataRows, data)
		for key := range data {
			headerSet[key] = true
		}
		count++
		if count >= sampleSize {
			break
		}
	}

	headers := make([]string, 0, len(headerSet))
	for key := range headerSet {
		headers = append(headers, key)
	}
	sort.Strings(headers)

	return headers, dataRows
}
