package cmd

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TomyTang331/Xml2ExcelByGo/internal/config"
	"github.com/TomyTang331/Xml2ExcelByGo/internal/converter"
	"github.com/spf13/cobra"
)

var (
	inputFile  string
	outputFile string
	bufferSize int
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert XML file to Excel",
	Long:  `Auto-detect repeating XML elements and convert to Excel table. Each repeating element becomes a row, child elements become columns.`,
	RunE:  runConvert,
}

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input XML file path (required)")
	convertCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output Excel file path (default: input_file.xlsx)")
	convertCmd.Flags().IntVarP(&bufferSize, "buffer-size", "b", config.DefaultXMLBufferSize, "XML parser buffer size in bytes")

	convertCmd.MarkFlagRequired("input")
}

func runConvert(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}

	if outputFile == "" {
		ext := filepath.Ext(inputFile)
		outputFile = strings.TrimSuffix(inputFile, ext) + ".xlsx"
	}

	fmt.Printf("Starting conversion...\n")
	fmt.Printf("Input:  %s\n", inputFile)
	fmt.Printf("Output: %s\n", outputFile)

	if isSVDFormat(inputFile) {
		fmt.Println("Detected CMSIS-SVD format, using multi-sheet converter...")
		svdConv := converter.NewSVDConverter(bufferSize)
		if err := svdConv.ConvertSVD(inputFile, outputFile); err != nil {
			return fmt.Errorf("conversion failed: %w", err)
		}
	} else {
		fmt.Println("Using generic flattening converter...")
		conv := converter.NewConverter(bufferSize)
		if err := conv.Convert(inputFile, outputFile); err != nil {
			return fmt.Errorf("conversion failed: %w", err)
		}
	}

	fmt.Printf("✓ Conversion completed successfully!\n")
	return nil
}

// isSVDFormat detects CMSIS-SVD format by file extension or root element.
func isSVDFormat(filename string) bool {
	if strings.HasSuffix(strings.ToLower(filename), ".svd") {
		return true
	}

	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	for {
		token, err := decoder.Token()
		if err != nil {
			return false
		}

		if elem, ok := token.(xml.StartElement); ok {
			return elem.Name.Local == "device"
		}
	}
}
