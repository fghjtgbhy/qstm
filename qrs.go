package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type QRS struct {
	cfg *Config
}

type Config struct {
	host      string
	certsPath string
}

type Task []struct {
	ID                 string        `json:"id"`
	Createddate        time.Time     `json:"createdDate"`
	Modifieddate       time.Time     `json:"modifiedDate"`
	Modifiedbyusername string        `json:"modifiedByUserName"`
	Customproperties   []interface{} `json:"customProperties"`
	Path               string        `json:"path,omitempty"`
	Parameters         string        `json:"parameters,omitempty"`
	Qlikuser           interface{}   `json:"qlikUser,omitempty"`
	Operational        struct {
		ID                  string `json:"id"`
		Lastexecutionresult struct {
			ID                 string    `json:"id"`
			Executingnodename  string    `json:"executingNodeName"`
			Status             int       `json:"status"`
			Starttime          time.Time `json:"startTime"`
			Stoptime           time.Time `json:"stopTime"`
			Duration           int       `json:"duration"`
			Filereferenceid    string    `json:"fileReferenceID"`
			Scriptlogavailable bool      `json:"scriptLogAvailable"`
			Details            []struct {
				ID                string      `json:"id"`
				Detailstype       int         `json:"detailsType"`
				Message           string      `json:"message"`
				Detailcreateddate time.Time   `json:"detailCreatedDate"`
				Privileges        interface{} `json:"privileges"`
			} `json:"details"`
			Scriptloglocation string      `json:"scriptLogLocation"`
			Scriptlogsize     int         `json:"scriptLogSize"`
			Privileges        interface{} `json:"privileges"`
		} `json:"lastExecutionResult"`
		Nextexecution time.Time   `json:"nextExecution"`
		Privileges    interface{} `json:"privileges"`
	} `json:"operational"`
	Name               string        `json:"name"`
	Tasktype           int           `json:"taskType"`
	Enabled            bool          `json:"enabled"`
	Tasksessiontimeout int           `json:"taskSessionTimeout"`
	Maxretries         int           `json:"maxRetries"`
	Tags               []interface{} `json:"tags"`
	Privileges         interface{}   `json:"privileges"`
	Schemapath         string        `json:"schemaPath"`
	App                struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Appid       string    `json:"appId"`
		Publishtime time.Time `json:"publishTime"`
		Published   bool      `json:"published"`
		Stream      struct {
			ID         string      `json:"id"`
			Name       string      `json:"name"`
			Privileges interface{} `json:"privileges"`
		} `json:"stream"`
		Savedinproductversion string      `json:"savedInProductVersion"`
		Migrationhash         string      `json:"migrationHash"`
		Availabilitystatus    int         `json:"availabilityStatus"`
		Privileges            interface{} `json:"privileges"`
	} `json:"app,omitempty"`
	Ismanuallytriggered bool `json:"isManuallyTriggered,omitempty"`
	Userdirectory       struct {
		ID         string      `json:"id"`
		Name       string      `json:"name"`
		Type       string      `json:"type"`
		Privileges interface{} `json:"privileges"`
	} `json:"userDirectory,omitempty"`
}

type FailedTaskData struct {
	ID           string
	TaskID       string
	start        time.Time
	stop         time.Time
	date_entered time.Time
	name         string
	errMessage   string
}

type FailedTaskDataArr []FailedTaskData

type ReloadToken struct {
	Value string `json:"value"`
}

func NewQRS(host, certsPath string) *QRS {
	qrs := &QRS{
		cfg: &Config{
			host:      host,
			certsPath: certsPath,
		},
	}

	return qrs
}

func (qrs QRS) MakeRequest(endPoint string) []byte {
	url := fmt.Sprintf("https://%s:4242/qrs/%s", qrs.cfg.host, endPoint)
	xrfkey := "ABCDEFG123456789"

	certFile := qrs.cfg.certsPath + "client.pem"
	keyFile := qrs.cfg.certsPath + "client_key.pem"
	caFile := qrs.cfg.certsPath + "root.pem"

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		panic(err)
	}

	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		panic(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
	}

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: tlsConfig,
	}}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("X-Qlik-xrfkey", xrfkey)
	req.Header.Add("X-Qlik-User", fmt.Sprintf("UserDirectory=%s; UserId=%s", "internal", "sa_api"))

	q := req.URL.Query()
	q.Add("xrfkey", xrfkey)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body
}

func (qrs QRS) getTask() Task {
	var t Task

	path := "task/full"
	body := qrs.MakeRequest(path)

	err := json.Unmarshal(body, &t)
	if err != nil {
		panic(err)
	}

	return t
}

func (qrs QRS) getTaskToken(taskId, fileRefId string) ReloadToken {
	var rt ReloadToken

	path := fmt.Sprintf("ReloadTask/%s/scriptlog?fileReferenceId=%s", taskId, fileRefId)
	token := qrs.MakeRequest(path)

	err := json.Unmarshal(token, &rt)
	if err != nil {
		panic(err)
	}

	return rt
}

func (qrs QRS) getTaskLog(taskId, taskName, fileRefId string) (log string) {
	rt := qrs.getTaskToken(taskId, fileRefId)

	path := fmt.Sprintf("download/reloadtask/%s/%s.log", rt.Value, taskName)
	log = string(qrs.MakeRequest(path))

	return
}

func (qrs QRS) getFailedTasksData() FailedTaskDataArr {
	var ft Task
	var ftd FailedTaskDataArr

	tasks := qrs.getTask()

	for _, task := range tasks {
		if task.Operational.Lastexecutionresult.Status == 8 {
			ft = append(ft, task)
		}
	}

	for _, task := range ft {
		log := qrs.getTaskLog(task.ID, task.Name, task.Operational.Lastexecutionresult.Filereferenceid)
		ftd = append(ftd, FailedTaskData{
			ID:           task.Operational.Lastexecutionresult.ID,
			TaskID:       task.ID,
			start:        task.Operational.Lastexecutionresult.Starttime,
			stop:         task.Operational.Lastexecutionresult.Stoptime,
			date_entered: time.Now(),
			name:         task.Name,
			errMessage:   LogError(log),
		})
	}

	return ftd
}
