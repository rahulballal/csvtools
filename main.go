package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/xuri/excelize/v2"
	"io/ioutil"
	"log/slog"
	"os"
	"strings"
	"time"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	var srcDir string
	var destDir string
	flag.StringVar(&srcDir, "src", "unknown", "source directory for csv files")
	flag.StringVar(&destDir, "dest", "unknown", "destination directory for xlsx file")

	flag.Parse()

	if srcDir == "unknown" || destDir == "unknown" {
		logger.Error("🧨  src and dst are required")
		os.Exit(1)
	}

	logger.Info("ℹ️ Using srcDir and destDir", "srcDir", srcDir, "destDir", destDir)

	fileMetadata, err := getFileNames(srcDir)
	if err != nil {
		logger.Error("🧨  Failed to get names of CSV files", "error", err)
		os.Exit(1)
	}
	if len(fileMetadata) == 0 {
		logger.Error("🧨  No CSV files found")
		os.Exit(1)
	}

	xlsxFile := excelize.NewFile()

	defer func() {
		if err := xlsxFile.Close(); err != nil {
			logger.Error("🧨  Failed to close xlsx file", "error", err)
		}
	}()

	for _, fileMetadatum := range fileMetadata {
		sheetName := fileMetadatum.NameWithoutExt
		logger.Info("🔍  Reading file", "file", fileMetadatum.FullPath)
		logger.Info("✏️  Writing to sheet", "sheet", sheetName)
		_, err := xlsxFile.NewSheet(sheetName)
		if err != nil {
			logger.Error("🧨  Failed to create sheet", "sheet", sheetName, "error", err)
			os.Exit(1)
		}
		csvFile, err := os.Open(fileMetadatum.FullPath)
		if err != nil {
			logger.Error("🧨  Failed to open csvFile", "file", fileMetadatum.FullPath, "error", err)
			os.Exit(1)
		}

		rowIdx := 1
		scanner := bufio.NewScanner(csvFile)
		for scanner.Scan() {
			line := scanner.Text()
			cells := strings.Split(line, ",")
			cellIdx := 1
			for _, cell := range cells {
				cellRef, _ := excelize.CoordinatesToCellName(cellIdx, rowIdx)
				if err := xlsxFile.SetCellStr(sheetName, cellRef, cell); err != nil {
					logger.Error("🧨  Failed to set cell value", "error", err)
					os.Exit(1)
				}
				cellIdx++
			}
			rowIdx++
		}
		if err := scanner.Err(); err != nil {
			logger.Error("🧨  Error reading csvFile", "file", fileMetadatum.FullPath, "error", err)
			os.Exit(1)
		}
		err = csvFile.Close()
		if err != nil {
			logger.Error("🧨  Failed to close csvFile", "file", fileMetadatum.FullPath, "error", err)
			os.Exit(1)
		}
		logger.Info("✅  Successfully written sheet", "sheet", sheetName)
	}

	_ = xlsxFile.DeleteSheet("Sheet1")

	currDt := fmt.Sprintf("%d", time.Now().Unix())
	xlsxFileSavePath := destDir + "/output_" + currDt + ".xlsx"
	err = xlsxFile.SaveAs(xlsxFileSavePath)
	if err != nil {
		logger.Error("🧨  Failed to save xlsx file", "error", err)
		os.Exit(1)
	}
	logger.Info("✅ Excel file created", "file", xlsxFileSavePath)
}

type FileMetadata struct {
	NameWithoutExt string
	FullPath       string
}

func getFileNames(directory string) ([]FileMetadata, error) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	var csvFiles []FileMetadata
	for _, file := range files {
		if !file.IsDir() {
			name := file.Name()
			if len(name) > 4 && name[len(name)-4:] == ".csv" {
				csvFiles = append(csvFiles, FileMetadata{
					NameWithoutExt: name[:len(name)-4],
					FullPath:       directory + "/" + name,
				})
			}
		}
	}
	return csvFiles, nil
}
