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
	_, err := DB.Query("INSERT INTO projects values (?,?,?,?)", uuid.New(), strings.ToLower(ProjectName), true, strings.ToLower(ProjectToken))
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
	_, err := DB.Query("DELETE from projects where project_name = ?", strings.ToUpper(ProjectName))
	if err != nil {
		log.Print(err.Error())
	}
}

func ToogleProject(ProjectName string, ProjectStatus bool) {
	_, err := DB.Query("UPDATE projects set status = ? where project_name = ?", ProjectStatus, strings.ToUpper(ProjectName))
	if err != nil {
		log.Print(err.Error())
	}
}
