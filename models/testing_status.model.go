package models

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

type TestingStatus struct {
	Id              string `json:"id"`
	Project         string `json:"project"`
	ServerId        string `json:"server_id"`
	LastBuildBy     string `json:"last_build_by"`
	LastBuildOn     int    `json:"last_build_on"`
	Status          bool   `json:"status"`
	StatusChangedBy string `json:"status_changed_by"`
	StatusChangedOn int    `json:"status_changed_on"`
}

func GetServerStatus(Project string) string {
	results, err := DB.Query("SELECT server_id, last_build_by, last_build_on, status, status_changed_by, status_changed_on FROM testing_status where project = ? order by server_id;", Project)
	if err != nil {
		log.Print(err.Error())
	}

	tempList := ""
	i := 1
	for results.Next() {
		var testingStatus TestingStatus

		err = results.Scan(&testingStatus.ServerId, &testingStatus.LastBuildBy, &testingStatus.LastBuildOn, &testingStatus.Status, &testingStatus.StatusChangedBy, &testingStatus.StatusChangedOn)

		if err != nil {
			log.Print(err.Error())
		}

		temp, _ := strconv.ParseInt(strconv.Itoa(testingStatus.LastBuildOn), 10, 64)
		location, _ := time.LoadLocation("Asia/Singapore")
		tempDate := time.Unix(temp, 0).In(location)
		tempStatus := "In use"
		if !testingStatus.Status {
			tempStatus = "Not in use"
		}
		tempList += fmt.Sprintf("%s. *%s* (%s) \n Last build by: %s (%s) \n\n", strconv.Itoa(i), testingStatus.ServerId, tempStatus, testingStatus.LastBuildBy, tempDate)
		i++
	}

	return tempList
}

func UpdateServerStatus(Project string, ServerId string, Name string, Status bool) {
	_, err := DB.Query("UPDATE testing_status set status = %s, status_changed_by =  %s, status_changed_on = %s where project = %s and server_id = %s;", Status, Name, time.Now().Unix(), Project, ServerId)
	if err != nil {
		log.Print(err.Error())
	}
}
