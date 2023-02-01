package models

type ReleaseSchedule struct {
	Id             string `json:"id"`
	ReleaseOn      string `json:"release_on"`
	ReleaseProject string `json:"release_project"`
	ReleaseVersion string `json:"release_version"`
	Released       int    `json:"released"`
	CreatedAt      int    `json:"created_at"`
	CreatedBy      string `json:"created_by"`
}
