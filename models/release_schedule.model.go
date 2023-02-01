package models

import "log"

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
