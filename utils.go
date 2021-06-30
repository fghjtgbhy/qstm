package main

import (
	"encoding/csv"
	"os"
	"strings"
	"time"
)

func LogError(log string) (err string) {
	logLines := strings.Split(log, "\n")
	for i := len(logLines) - 2; i > 0; i-- {
		lineArray := strings.Fields(logLines[i])
		if lineArray[1] == "Error:" {
			return strings.Join(lineArray[2:], " ")
		}
	}

	return "no error message"
}

func WriteCSV(ftd FailedTaskDataArr) {
	filename := string(time.Now().Format("02_01_2006")) + ".csv"
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	fi, err := file.Stat()
	if err != nil {
		panic(err)
	}

	if fi.Size() == 0 {
		err = writer.Write([]string{"id", "task_id", "start", "stop", "date_entered", "name", "error"})
		if err != nil {
			panic(err)
		}
	}

	for _, task := range ftd {
		arr := []string{task.ID, task.TaskID, task.start.String(), task.stop.String(), task.date_entered.String(), task.name, task.errMessage}
		err = writer.Write(arr)
		if err != nil {
			panic(err)
		}
	}
}
