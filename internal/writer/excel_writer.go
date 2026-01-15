package writer

import (
	"fmt"

	"github.com/TomyTang331/Xml2ExcelByGo/internal/config"
	"github.com/xuri/excelize/v2"
)

type ExcelWriter struct {
	file          *excelize.File
	filename      string
	batchSize     int
	currentSheets map[string]*SheetWriter
}

type SheetWriter struct {
	streamWriter *excelize.StreamWriter
	headers      []string
	rowIndex     int
	rowBuffer    []map[string]string
	batchSize    int
}

func NewExcelWriter(filename string, batchSize int) *ExcelWriter {
	return &ExcelWriter{
		file:          excelize.NewFile(),
		filename:      filename,
		batchSize:     batchSize,
		currentSheets: make(map[string]*SheetWriter),
	}
}

// CreateSheet creates a new worksheet with headers.
func (ew *ExcelWriter) CreateSheet(sheetName string, headers []string) error {
	index, err := ew.file.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet: %w", err)
	}

	if len(ew.currentSheets) == 0 {
		ew.file.SetActiveSheet(index)
		ew.file.DeleteSheet("Sheet1")
	}

	streamWriter, err := ew.file.NewStreamWriter(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create stream writer: %w", err)
	}

	styleID, _ := ew.file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{config.HeaderStyleBg},
			Pattern: 1,
		},
	})

	headerRow := make([]interface{}, len(headers))
	for i, h := range headers {
		headerRow[i] = excelize.Cell{
			Value:   h,
			StyleID: styleID,
		}
	}

	if err := streamWriter.SetRow("A1", headerRow); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	ew.currentSheets[sheetName] = &SheetWriter{
		streamWriter: streamWriter,
		headers:      headers,
		rowIndex:     2,
		rowBuffer:    make([]map[string]string, 0, ew.batchSize),
		batchSize:    ew.batchSize,
	}

	return nil
}

// WriteRow appends a row to the buffer and flushes if full.
func (ew *ExcelWriter) WriteRow(sheetName string, data map[string]string) error {
	sheet, ok := ew.currentSheets[sheetName]
	if !ok {
		return fmt.Errorf("sheet not found: %s", sheetName)
	}

	sheet.rowBuffer = append(sheet.rowBuffer, data)

	if len(sheet.rowBuffer) >= sheet.batchSize {
		return ew.flushSheet(sheetName)
	}

	return nil
}

// flushSheet writes buffered data to the worksheet.
func (ew *ExcelWriter) flushSheet(sheetName string) error {
	sheet, ok := ew.currentSheets[sheetName]
	if !ok {
		return fmt.Errorf("sheet not found: %s", sheetName)
	}

	if len(sheet.rowBuffer) == 0 {
		return nil
	}

	for _, rowData := range sheet.rowBuffer {
		row := make([]interface{}, len(sheet.headers))
		for i, header := range sheet.headers {
			if val, exists := rowData[header]; exists {
				row[i] = val
			} else {
				row[i] = ""
			}
		}

		cellName, _ := excelize.CoordinatesToCellName(1, sheet.rowIndex)
		if err := sheet.streamWriter.SetRow(cellName, row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}

		sheet.rowIndex++
	}

	sheet.rowBuffer = sheet.rowBuffer[:0]
	return nil
}

// Close flushes all buffers and saves the file.
func (ew *ExcelWriter) Close() error {
	for sheetName := range ew.currentSheets {
		if err := ew.flushSheet(sheetName); err != nil {
			return err
		}
	}

	for _, sheet := range ew.currentSheets {
		if err := sheet.streamWriter.Flush(); err != nil {
			return fmt.Errorf("failed to flush stream writer: %w", err)
		}
	}

	for sheetName, sheet := range ew.currentSheets {
		if err := ew.autoSizeColumns(sheetName, sheet.headers); err != nil {
			fmt.Printf("Warning: failed to auto-size columns: %v\n", err)
		}
	}

	if err := ew.file.SaveAs(ew.filename); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

func (ew *ExcelWriter) autoSizeColumns(sheetName string, headers []string) error {
	for i := range headers {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		if err := ew.file.SetColWidth(sheetName, colName, colName, config.DefaultColWidth); err != nil {
			return err
		}
	}
	return nil
}
