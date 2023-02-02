package models

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

type ReleaseSchedule struct {
	Id             string `json:"id"`
	ReleaseOn      string `json:"release_on"`
	ReleaseProject string `json:"release_project"`
	ReleaseVersion string `json:"release_version"`
	Released       int    `json:"released"`
	CreatedAt      int    `json:"created_at"`
	CreatedBy      string `json:"created_by"`
}

func UpdateReleased(Id string) {
	_, err := DB.Query("UPDATE release_schedule set released = 1 where id = ?", Id)
	if err != nil {
		log.Printf(err.Error())
	}
}

func GetActiveRelease() string {
	results, err := DB.Query("SELECT * FROM release_schedule WHERE released = 0")
	if err != nil {
		log.Printf(err.Error())
	}

	tempList := ""
	i := 1
	for results.Next() {
		var releaseSchedule ReleaseSchedule

		//map to struct
		err = results.Scan(&releaseSchedule.Id, &releaseSchedule.ReleaseOn, &releaseSchedule.ReleaseProject, &releaseSchedule.ReleaseVersion, &releaseSchedule.Released, &releaseSchedule.CreatedAt, &releaseSchedule.CreatedBy)

		if err != nil {
			log.Printf(err.Error())
		}

		temp, _ := strconv.ParseInt(strconv.Itoa(releaseSchedule.CreatedAt), 10, 64)
		location, _ := time.LoadLocation("Asia/Singapore")
		tempDate := time.Unix(temp, 0).In(location)
		tempList += fmt.Sprintf("%s. *%s* > %s \n\t Will be release on: %s (Asia/Singapore) \n\t Id: %s \n\t Created by: %s \n\t Created on: %s \n\n", strconv.Itoa(i), releaseSchedule.ReleaseProject, releaseSchedule.ReleaseVersion, releaseSchedule.ReleaseOn, releaseSchedule.Id, releaseSchedule.CreatedBy, tempDate.String())
		i++
	}

	return tempList
}

func CheckActiveRelease(Id string) bool {
	var releaseSchedule ReleaseSchedule

	err := DB.QueryRow("Select * from release_schedule where id = ? and released = 0", Id).Scan(&releaseSchedule.Id, &releaseSchedule.ReleaseOn, &releaseSchedule.ReleaseProject, &releaseSchedule.ReleaseVersion, &releaseSchedule.Released, &releaseSchedule.CreatedAt, &releaseSchedule.CreatedBy)

	return err == nil
}
