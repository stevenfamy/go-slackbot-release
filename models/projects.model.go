package models

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Projects struct {
	Id           string `json:"id"`
	ProjectName  string `json:"project_name"`
	Status       bool   `json:"status"`
	JenkinsToken string `json:"jenkins_token"`
}

func AddNewProject(ProjectName string, ProjectToken string) {
	_, err := DB.Query("INSERT INTO projects values (?,?,?,?)", uuid.New(), strings.ToLower(ProjectName), true, strings.ToUpper(ProjectToken))
	if err != nil {
		log.Print(err.Error())
	}
}

func GetAllProjects() string {
	results, err := DB.Query("SELECT project_name, status, jenkins_token FROM projects;")
	if err != nil {
		log.Print(err.Error())
	}

	tempList := ""
	i := 1
	for results.Next() {
		var projects Projects

		err = results.Scan(&projects.ProjectName, &projects.Status, &projects.JenkinsToken)

		if err != nil {
			log.Print(err.Error())
		}

		tempStatus := "Enable"
		if !projects.Status {
			tempStatus = "Disabled"
		}
		tempList += fmt.Sprintf("%s. *%s* : %s (%s) \n\n", strconv.Itoa(i), projects.ProjectName, projects.JenkinsToken, tempStatus)
		i++
	}

	return tempList
}

func DeleteProject(ProjectName string) {
	_, err := DB.Query("DELETE from projects where project_name = ?", strings.ToLower(ProjectName))
	if err != nil {
		log.Print(err.Error())
	}
}

func ToogleProject(ProjectName string, ProjectStatus bool) {
	_, err := DB.Query("UPDATE projects set status = ? where project_name = ?", ProjectStatus, strings.ToLower(ProjectName))
	if err != nil {
		log.Print(err.Error())
	}
}

func ProjectIsAvailable(ProjectName string) bool {
	var projects Projects

	err := DB.QueryRow("Select id from projects where project_name = ? and status = 1", strings.ToLower(ProjectName)).Scan(&projects.Id)

	return err == nil
}

func GetProjectToken(ProjectName string) string {
	var projects Projects

	row := DB.QueryRow("Select jenkins_token from projects where project_name = ? and status = 1", strings.ToLower(ProjectName)).Scan(&projects.JenkinsToken)

	if row == nil {
		log.Print(row.Error())
		return ""
	}

	return projects.JenkinsToken
}
