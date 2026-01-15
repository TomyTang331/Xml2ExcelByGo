package converter

import (
	"fmt"
	"sync"

	"github.com/TomyTang331/Xml2ExcelByGo/internal/config"
	"github.com/TomyTang331/Xml2ExcelByGo/internal/parser"
	"github.com/TomyTang331/Xml2ExcelByGo/internal/writer"
)

type SVDConverter struct {
	bufferSize int
	batchSize  int
}

func NewSVDConverter(bufferSize int) *SVDConverter {
	return &SVDConverter{
		bufferSize: bufferSize,
		batchSize:  config.DefaultBatchSize,
	}
}

func (c *SVDConverter) ConvertSVD(inputFile, outputFile string) error {
	p := parser.NewSVDParser(c.bufferSize)
	peripheralChan, registerChan, fieldChan, errChan := p.ParseSVD(inputFile)

	peripheralHeaders := []string{
		"_id", "name", "description", "groupName", "baseAddress",
		"size", "access", "resetValue",
	}

	registerHeaders := []string{
		"_id", "_peripheral_id", "_peripheral_name",
		"name", "displayName", "description",
		"addressOffset", "size", "access", "resetValue",
	}

	fieldHeaders := []string{
		"_id", "_register_id", "_register_name", "_peripheral_id", "_peripheral_name",
		"name", "description", "bitOffset", "bitWidth", "access",
	}

	excelWriter := writer.NewExcelWriter(outputFile, c.batchSize)
	defer excelWriter.Close()

	if err := excelWriter.CreateSheet("Peripherals", peripheralHeaders); err != nil {
		return fmt.Errorf("failed to create Peripherals sheet: %w", err)
	}

	if err := excelWriter.CreateSheet("Registers", registerHeaders); err != nil {
		return fmt.Errorf("failed to create Registers sheet: %w", err)
	}

	if err := excelWriter.CreateSheet("Fields", fieldHeaders); err != nil {
		return fmt.Errorf("failed to create Fields sheet: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(3)

	errors := make(chan error, 3)

	// Process peripherals
	go func() {
		defer wg.Done()
		count := 0
		for data := range peripheralChan {
			if err := excelWriter.WriteRow("Peripherals", data); err != nil {
				errors <- fmt.Errorf("failed to write peripheral: %w", err)
				return
			}
			count++
			if count%100 == 0 {
				fmt.Printf("  Processed %d peripherals...\n", count)
			}
		}
		fmt.Printf("✓ Peripherals: %d rows\n", count)
	}()

	// Process registers
	go func() {
		defer wg.Done()
		count := 0
		for data := range registerChan {
			if err := excelWriter.WriteRow("Registers", data); err != nil {
				errors <- fmt.Errorf("failed to write register: %w", err)
				return
			}
			count++
			if count%1000 == 0 {
				fmt.Printf("  Processed %d registers...\n", count)
			}
		}
		fmt.Printf("✓ Registers: %d rows\n", count)
	}()

	// Process fields
	go func() {
		defer wg.Done()
		count := 0
		for data := range fieldChan {
			if err := excelWriter.WriteRow("Fields", data); err != nil {
				errors <- fmt.Errorf("failed to write field: %w", err)
				return
			}
			count++
			if count%1000 == 0 {
				fmt.Printf("  Processed %d fields...\n", count)
			}
		}
		fmt.Printf("✓ Fields: %d rows\n", count)
	}()

	// Check for parsing errors
	go func() {
		for err := range errChan {
			if err != nil {
				errors <- err
			}
		}
	}()

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		if err != nil {
			return err
		}
	}

	fmt.Println("\nSaving file...")
	return nil
}
