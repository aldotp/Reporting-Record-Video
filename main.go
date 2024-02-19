package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type RecordReport struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Result    Result `json:"result"`
}

type Result struct {
	ErrorPercentage  float64         `json:"error_percentage"`
	ErrorTime        int             `json:"error_time"`
	RecordPercentage float64         `json:"record_percentage"`
	RecordTime       int             `json:"record_time"`
	TotalError       int             `json:"total_error"`
	TotalRecording   int             `json:"total_recording"`
	TotalTime        int             `json:"total_time"`
	Error            []ErrorDetail   `json:"error"`
	RecordingFile    []RecordingFile `json:"recording_file"`
}

type ErrorDetail struct {
	Duration  int       `json:"duration"`
	Filename  string    `json:"filename"`
	TimeError TimeError `json:"time_error"`
}

type TimeError struct {
	EndTime   string `json:"end_time"`
	StartTime string `json:"start_time"`
}

type RecordingFile struct {
	Duration int    `json:"duration"`
	Filename string `json:"filename"`
}

func (report *RecordReport) Export() {
	reportJSON, err := json.MarshalIndent(report, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	saveFile := fmt.Sprintf("report/report_%s.json", report.StartTime)
	if _, err := os.Stat(saveFile); err == nil {
		if err := os.Remove(saveFile); err != nil {
			log.Fatalf("Error removing existing file: %v", err)
		}
	}

	err = os.WriteFile(saveFile, reportJSON, 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Report saved, name:", saveFile)
}

const (
	layoutTime string = "2006-01-02 15-04-05"
	directory  string = "record"
)

func main() {
	GenerateReport("2024-01-09 00-00-00", "2024-01-10 00-00-00", directory).Export()
	GenerateReport("2024-01-10 00-00-00", "2024-01-11 00-00-00", directory).Export()
	GenerateReport("2024-01-11 00-00-00", "2024-01-12 00-00-00", directory).Export()
}

func getRecordingFiles(st, et string, dir string) []string {
	var recordingFiles []string
	startTime, _ := time.Parse(layoutTime, st)

	dateString := startTime.Format("2006-01-02")
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal("Error reading directory:", err)
	}
	for _, file := range files {
		if strings.Contains(file.Name(), dateString) && strings.HasSuffix(file.Name(), ".mp4") {
			recordingFiles = append(recordingFiles, file.Name())
		}
	}

	return recordingFiles
}

func getVideoDuration(filename string) (int, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filename)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	durationStr := strings.TrimSpace(string(output))
	duration := 0
	if durationStr != "" {
		FloatDuration, err := strconv.ParseFloat(durationStr, 64)
		if err != nil {
			return 0, err
		}
		duration = int(FloatDuration)
	}

	return duration, nil
}

func GenerateReport(startDate, endDate string, dir string) *RecordReport {
	var (
		errorTime, recordTime             int
		errorPercentage, recordPercentage float64
		errorDetails                      []ErrorDetail
		recordingFiles                    []RecordingFile
	)

	files := getRecordingFiles(startDate, endDate, dir)
	totalRecord := len(files)
	totalTime := 24 * 60 * 60 // 1 day

	for _, filename := range files {
		fileLoc := fmt.Sprintf("%s/%s", dir, filename)
		duration, err := getVideoDuration(fileLoc)
		if err != nil {
			log.Fatal(err)
		}

		if duration < 300 {
			parts := strings.Split(filename, ".")
			startTimeStr := parts[0]
			startTime, err := time.Parse("2006-01-02T15-04-05", startTimeStr)
			if err != nil {
				log.Fatal(err)
			}

			endTime := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), startTime.Hour(), startTime.Minute(), duration, 0, startTime.Location())
			errorTime += duration
			errorDetails = append(errorDetails, ErrorDetail{
				Duration: duration,
				Filename: filename,
				TimeError: TimeError{
					EndTime:   endTime.Format(layoutTime),
					StartTime: startTime.Format(layoutTime),
				},
			})
		}

		recordingFiles = append(recordingFiles, RecordingFile{
			Duration: duration,
			Filename: filename,
		})
	}

	recordTime = totalTime - errorTime
	errorTime = totalTime - recordTime
	totalError := len(errorDetails)
	errorPercentage = float64(errorTime) / float64(totalTime) * 100
	recordPercentage = float64(recordTime) / float64(totalTime) * 100

	report := RecordReport{
		StartTime: startDate,
		EndTime:   endDate,
		Result: Result{
			ErrorPercentage:  errorPercentage,
			ErrorTime:        errorTime,
			RecordPercentage: recordPercentage,
			TotalError:       totalError,
			RecordTime:       recordTime,
			TotalRecording:   totalRecord,
			TotalTime:        totalTime,
			Error:            errorDetails,
			RecordingFile:    recordingFiles,
		},
	}

	return &report
}
