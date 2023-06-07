package models

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type SlackUserAccess struct {
	Id       string `json:"id"`
	SlackId  string `json:"slack_id"`
	Status   bool   `json:"status"`
	AddedAt  int    `json:"added_at"`
	FullName string `json:"full_name"`
}

func AddNewUser(SlackId string, FullName string) {
	_, err := DB.Query("INSERT INTO slack_user_access values (?,?,?,?,?)", uuid.New(), SlackId, true, time.Now().Unix(), FullName)
	if err != nil {
		log.Print(err.Error())
	}
}

func GetAllUsers() string {
	results, err := DB.Query("SELECT * FROM slack_user_access;")
	if err != nil {
		log.Print(err.Error())
	}

	tempList := ""
	i := 1
	for results.Next() {
		var slackUserAccess SlackUserAccess

		err = results.Scan(&slackUserAccess.Id, &slackUserAccess.SlackId, &slackUserAccess.Status, &slackUserAccess.AddedAt, &slackUserAccess.FullName)

		if err != nil {
			log.Print(err.Error())
		}

		// temp, _ := strconv.ParseInt(strconv.Itoa(slackUserAccess.AddedAt), 10, 64)
		// location, _ := time.LoadLocation("Asia/Singapore")
		// tempDate := time.Unix(temp, 0).In(location)
		tempStatus := "Enable"
		if !slackUserAccess.Status {
			tempStatus = "Disabled"
		}
		tempList += fmt.Sprintf("%s. *%s* : %s (%s) \n\t", strconv.Itoa(i), slackUserAccess.FullName, slackUserAccess.SlackId, tempStatus)
		i++
	}

	return tempList
}

func DeleteUserAccess(SlackId string) {
	_, err := DB.Query("DELETE from slack_user_access where slack_id = ?", SlackId)
	if err != nil {
		log.Print(err.Error())
	}
}

func ToogleUserStatus(SlackId string) {
	var slackUserAccess SlackUserAccess

	err := DB.QueryRow("Select * from slack_user_access where slack_id = ?", SlackId).Scan(&slackUserAccess.Id, &slackUserAccess.SlackId, &slackUserAccess.Status, &slackUserAccess.AddedAt, &slackUserAccess.FullName)

	if err != nil {
		log.Print(err.Error())
	}

	_, err2 := DB.Query("UPDATE slack_user_access set status = ? where id = ?", !slackUserAccess.Status, SlackId)
	if err2 != nil {
		log.Print(err.Error())
	}
}