package parser

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	peripheralIDPrefix = "P"
	registerIDPrefix   = "R"
	fieldIDPrefix      = "F"
	idFormatWidth      = 4
)

// SVDParser parses CMSIS-SVD format XML files
type SVDParser struct {
	bufferSize int
}

func NewSVDParser(bufferSize int) *SVDParser {
	return &SVDParser{bufferSize: bufferSize}
}

// ParseSVD parses SVD file and returns 3 channels for peripherals, registers, and fields
func (p *SVDParser) ParseSVD(filename string) (
	<-chan map[string]string,
	<-chan map[string]string,
	<-chan map[string]string,
	<-chan error,
) {
	peripheralChan := make(chan map[string]string, channelBufferSize)
	registerChan := make(chan map[string]string, channelBufferSize)
	fieldChan := make(chan map[string]string, channelBufferSize)
	errChan := make(chan error, 1)

	go func() {
		defer close(peripheralChan)
		defer close(registerChan)
		defer close(fieldChan)
		defer close(errChan)

		file, err := os.Open(filename)
		if err != nil {
			errChan <- err
			return
		}
		defer file.Close()

		reader := bufio.NewReaderSize(file, p.bufferSize)
		decoder := xml.NewDecoder(reader)

		var currentPeripheral map[string]string
		var currentRegister map[string]string
		var currentField map[string]string

		var pathStack []string
		var textBuilder string

		peripheralCount := 0
		registerCount := 0
		fieldCount := 0

		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				errChan <- fmt.Errorf("XML parsing error: %w", err)
				return
			}

			switch elem := token.(type) {
			case xml.StartElement:
				pathStack = append(pathStack, elem.Name.Local)
				textBuilder = ""

				// Detect peripheral start
				if elem.Name.Local == "peripheral" && isInPath(pathStack, "peripherals") {
					currentPeripheral = make(map[string]string)
					currentPeripheral["_id"] = fmt.Sprintf("%s%0*d", peripheralIDPrefix, idFormatWidth, peripheralCount)
					peripheralCount++
				}

				// Detect register start
				if elem.Name.Local == "register" && isInPath(pathStack, "registers") && currentPeripheral != nil {
					currentRegister = make(map[string]string)
					currentRegister["_id"] = fmt.Sprintf("%s%0*d", registerIDPrefix, idFormatWidth, registerCount)
					currentRegister["_peripheral_id"] = currentPeripheral["_id"]
					currentRegister["_peripheral_name"] = currentPeripheral["name"]
					registerCount++
				}

				// Detect field start
				if elem.Name.Local == "field" && isInPath(pathStack, "fields") && currentRegister != nil {
					currentField = make(map[string]string)
					currentField["_id"] = fmt.Sprintf("%s%0*d", fieldIDPrefix, idFormatWidth, fieldCount)
					currentField["_register_id"] = currentRegister["_id"]
					currentField["_register_name"] = currentRegister["name"]
					currentField["_peripheral_id"] = currentRegister["_peripheral_id"]
					currentField["_peripheral_name"] = currentRegister["_peripheral_name"]
					fieldCount++
				}

			case xml.CharData:
				textBuilder += string(elem)

			case xml.EndElement:
				text := strings.TrimSpace(textBuilder)

				// Save peripheral child elements
				if currentPeripheral != nil && len(pathStack) >= 2 {
					parentElement := pathStack[len(pathStack)-2]
					if parentElement == "peripheral" && elem.Name.Local != "peripheral" && elem.Name.Local != "registers" && text != "" {
						currentPeripheral[elem.Name.Local] = text
					}
				}

				// Save register child elements
				if currentRegister != nil && len(pathStack) >= 2 {
					parentElement := pathStack[len(pathStack)-2]
					if parentElement == "register" && elem.Name.Local != "register" && elem.Name.Local != "fields" && text != "" {
						currentRegister[elem.Name.Local] = text
					}
				}

				// Save field child elements
				if currentField != nil && len(pathStack) >= 2 {
					parentElement := pathStack[len(pathStack)-2]
					if parentElement == "field" && text != "" {
						currentField[elem.Name.Local] = text
					}
				}

				// Send completed peripheral
				if elem.Name.Local == "peripheral" && currentPeripheral != nil {
					peripheralChan <- currentPeripheral
					currentPeripheral = nil
				}

				// Send completed register
				if elem.Name.Local == "register" && currentRegister != nil {
					registerChan <- currentRegister
					currentRegister = nil
				}

				// Send completed field
				if elem.Name.Local == "field" && currentField != nil {
					fieldChan <- currentField
					currentField = nil
				}

				if len(pathStack) > 0 {
					pathStack = pathStack[:len(pathStack)-1]
				}
				textBuilder = ""
			}
		}

		fmt.Printf("SVD parsing completed: %d peripherals, %d registers, %d fields\n",
			peripheralCount, registerCount, fieldCount)
	}()

	return peripheralChan, registerChan, fieldChan, errChan
}

func isInPath(stack []string, target string) bool {
	for _, s := range stack {
		if s == target {
			return true
		}
	}
	return false
}
