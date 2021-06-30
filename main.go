package main

import (
	"os"
)

func main() {
	var (
		host = os.Getenv("GO_QLIK_HOST")
		path = os.Getenv("GO_QLIK_CERTS_PATH")
	)

	qrs := NewQRS(host, path)
	ftd := qrs.getFailedTasksData()
	WriteCSV(ftd)
}
