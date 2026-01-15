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
	channelBufferSize = 100
)

type Parser struct {
	bufferSize int
}

func NewParser(bufferSize int) *Parser {
	return &Parser{
		bufferSize: bufferSize,
	}
}

// ParseToRows parses XML and flattens to row data.
// Auto-detects repeating elements as "rows" and child elements as "columns".
func (p *Parser) ParseToRows(filename string) (<-chan map[string]string, <-chan error) {
	dataChan := make(chan map[string]string, channelBufferSize)
	errChan := make(chan error, 1)

	go func() {
		defer close(dataChan)
		defer close(errChan)

		file, err := os.Open(filename)
		if err != nil {
			errChan <- err
			return
		}
		defer file.Close()

		// First pass: detect repeating element
		reader := bufio.NewReaderSize(file, p.bufferSize)
		decoder := xml.NewDecoder(reader)
		repeatingElement := p.detectRepeatingElementFromDecoder(decoder)

		if repeatingElement == "" {
			errChan <- fmt.Errorf("failed to detect repeating XML element")
			return
		}

		fmt.Printf("Detected repeating element: <%s>\n", repeatingElement)

		// Rewind file for second pass
		if _, err := file.Seek(0, 0); err != nil {
			errChan <- fmt.Errorf("failed to rewind file: %w", err)
			return
		}

		// Second pass: extract data
		reader = bufio.NewReaderSize(file, p.bufferSize)
		decoder = xml.NewDecoder(reader)

		var currentRow map[string]string
		var pathStack []string
		var currentText string
		inTargetElement := false
		rowCount := 0

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
				currentText = ""

				if elem.Name.Local == repeatingElement {
					inTargetElement = true
					currentRow = make(map[string]string)
				}

			case xml.CharData:
				currentText += string(elem)

			case xml.EndElement:
				text := strings.TrimSpace(currentText)

				// Save child element data if inside target element
				if inTargetElement && len(pathStack) >= 2 {
					parentElement := pathStack[len(pathStack)-2]
					if parentElement == repeatingElement && elem.Name.Local != repeatingElement {
						currentRow[elem.Name.Local] = text
					}
				}

				// Check if target element ended
				if elem.Name.Local == repeatingElement && inTargetElement {
					dataChan <- currentRow
					rowCount++
					inTargetElement = false
					currentRow = nil
				}

				if len(pathStack) > 0 {
					pathStack = pathStack[:len(pathStack)-1]
				}
				currentText = ""
			}
		}

		fmt.Printf("Parsing completed: %d rows\n", rowCount)
	}()

	return dataChan, errChan
}

// detectRepeatingElementFromDecoder detects the most common repeating element from an active decoder.
func (p *Parser) detectRepeatingElementFromDecoder(decoder *xml.Decoder) string {
	elementCounts := make(map[string]int)
	var pathStack []string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ""
		}

		switch elem := token.(type) {
		case xml.StartElement:
			pathStack = append(pathStack, elem.Name.Local)
			// Count elements with depth > 1 (exclude root element)
			if len(pathStack) > 1 {
				elementCounts[elem.Name.Local]++
			}

		case xml.EndElement:
			if len(pathStack) > 0 {
				pathStack = pathStack[:len(pathStack)-1]
			}
		}
	}

	// Find most common element
	maxCount := 0
	repeatingElement := ""
	for elem, count := range elementCounts {
		if count > maxCount && count > 1 {
			maxCount = count
			repeatingElement = elem
		}
	}

	return repeatingElement
}
