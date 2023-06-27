package models

type ProjectConfig struct {
	Id           string `json:"id"`
	ProjectId    string `json:"project_id"`
	ProjectName  string `json:"project_name"`
	JenkinsToken string `json:"jenkins_token"`
	Status       bool   `json:"status"`
}

func GetProjectDetails(ProjectId string) {
	var projectConfig ProjectConfig

	result := DB.QueryRow("SELECT * FROM project_config").Scan(&projectConfig.Id, &projectConfig.ProjectId, &projectConfig.ProjectName, &projectConfig.JenkinsToken, &projectConfig.Status)

}
