package main

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/stevenfamy/go-slackbot-release/config"
	"github.com/stevenfamy/go-slackbot-release/models"
)

func main() {
	models.ConnectDatabase()

	//define 1 minutes ticker
	ticker := time.NewTicker(5 * time.Second)
	tickerChan := make(chan bool)

	// Load config
	token := config.GetConfig("SLACK_AUTH_TOKEN")
	appToken := config.GetConfig("SLACK_APP_TOKEN")

	// Create a new client to slack by giving token
	// Set debug to true while developing
	// Also add a ApplicationToken option to the client
	client := slack.New(token, slack.OptionDebug(false), slack.OptionAppLevelToken(appToken))

	// go-slack comes with a SocketMode package that we need to use that accepts a Slack client and outputs a Socket mode client instead
	socket := socketmode.New(
		client,
		socketmode.OptionDebug(false),
		// Option to set a custom logger
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	// Create a context that can be used to cancel goroutine
	ctx, cancel := context.WithCancel(context.Background())

	// Make this cancel called properly in a real program , graceful shutdown etc
	defer cancel()

	//thread for receiving event from slack
	go func(ctx context.Context, client *slack.Client, socket *socketmode.Client) {
		// Create a for loop that selects either the context cancellation or the events incomming
		for {
			select {
			// inscase context cancel is called exit the goroutine
			case <-ctx.Done():
				log.Println("Shutting down socketmode listener")
				return
			case event := <-socket.Events:
				// We have a new Events, let's type switch the event
				// Add more use cases here if you want to listen to other events.
				switch event.Type {
				// handle EventAPI events
				case socketmode.EventTypeEventsAPI:
					// The Event sent on the channel is not the same as the EventAPI events so we need to type cast it
					eventsAPI, ok := event.Data.(slackevents.EventsAPIEvent)
					if !ok {
						log.Printf("Could not type cast the event to the EventsAPIEvent: %v\n", event)
						continue
					}
					// We need to send an Acknowledge to the slack server
					socket.Ack(*event.Request)
					// Now we have an Events API event, but this event type can in turn be many types, so we actually need another type switch

					//log.Println(eventsAPI) // commenting for event hanndling

					//------------------------------------
					// Now we have an Events API event, but this event type can in turn be many types, so we actually need another type switch
					err := HandleEventMessage(eventsAPI, client)
					if err != nil {
						// Replace with actual err handeling
						log.Fatal(err)
					}
				}
			}
		}
	}(ctx, client, socket)

	//thread of looping ticker to check every minutes
	go func() {
		for {
			select {
			case <-tickerChan:
				return
			// interval task
			case tm := <-ticker.C:

				//get time now in +8
				location, _ := time.LoadLocation("Asia/Singapore")
				now := tm.In(location)

				///get schedule from db
				results, err := models.DB.Query("SELECT * FROM release_schedule WHERE released = 0")

				if err != nil {
					log.Printf(err.Error())
				}

				//loop all active schedule
				for results.Next() {
					var releaseSchedule models.ReleaseSchedule

					//map to struct
					err = results.Scan(&releaseSchedule.Id, &releaseSchedule.ReleaseOn, &releaseSchedule.ReleaseProject, &releaseSchedule.ReleaseVersion, &releaseSchedule.Released, &releaseSchedule.CreatedAt, &releaseSchedule.CreatedBy)

					if err != nil {
						log.Printf(err.Error())
					}

					//parse and check time
					converted, _ := time.Parse(time.Kitchen, releaseSchedule.ReleaseOn)
					t1 := time.Date(now.Year(), now.Month(), now.Day(), converted.Hour(), converted.Minute(), 0, 0, now.Location())
					if now.After(t1) {
						//time ok
						log.Println("OK Release", releaseSchedule.ReleaseProject, releaseSchedule.ReleaseVersion)
						//update to 1
						callJenkins(releaseSchedule.ReleaseProject, releaseSchedule.ReleaseVersion, false, "0000")
						models.UpdateReleased(releaseSchedule.Id)
					} else {
						log.Println(releaseSchedule.Id, "Time not match yet")
					}
				}
			}
		}
	}()

	socket.Run()
}

// HandleEventMessage will take an event and handle it properly based on the type of event
func HandleEventMessage(event slackevents.EventsAPIEvent, client *slack.Client) error {
	switch event.Type {
	// First we check if this is an CallbackEvent
	case slackevents.CallbackEvent:
		log.Println("received slack event")
		innerEvent := event.InnerEvent
		// Yet Another Type switch on the actual Data to see if its an AppMentionEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			// The application has been mentioned since this Event is a Mention event
			err := HandleAppMentionEventToBot(ev, client)
			if err != nil {
				return err
			}
		}
	default:
		return errors.New("unsupported event type")
	}
	return nil
}

// HandleAppMentionEventToBot is used to take care of the AppMentionEvent when the bot is mentioned
func HandleAppMentionEventToBot(event *slackevents.AppMentionEvent, client *slack.Client) error {

	// Grab the user name based on the ID of the one who mentioned the bot
	user, err := client.GetUserInfo(event.User)
	if err != nil {
		return err
	}
	// Check if the user said Hello to the bot
	text := strings.ToLower(html.UnescapeString(event.Text))

	// Create the attachment and assigned based on the message
	attachment := slack.Attachment{}

	projectList := []string{"gla-platform", "gla-parent", "gla-admin", "logistics-backend", "logistics-web", "logistics-mobile"}

	if strings.Contains(text, "my id") {
		// Send a message to the user
		attachment.Text = fmt.Sprintf("Psst <@%s> your slack id is %s", user.ID, user.ID)
		// attachment.Pretext = "How can I be of service"
		attachment.Color = "#563a9b"
	} else if strings.Contains(text, "how to schedule release") {
		// Send a message to the user
		attachment.Text = fmt.Sprintf("Easy <@%s>, you just need to mention me with this message format 'schedule release projectname <<version>> at hh:mma' in Asia/Singapore timezone", user.ID)
		// attachment.Pretext = "How can I be of service"
		attachment.Footer = "Example: schedule release logistics-backend <<backend-1.1.0-beta>> at 09:25PM"
		attachment.Color = "#563a9b"
	} else if strings.Contains(text, "how to remove schedule") {
		// Send a message to the user
		attachment.Text = fmt.Sprintf("Easy <@%s>, you just need to mention me with this message format 'remove schedule id'", user.ID)
		// attachment.Pretext = "How can I be of service"
		attachment.Footer = "Example: remove schedule 7ede5801-f6bb-4eaf-926f-54ee7f65905c "
		attachment.Color = "#563a9b"
	} else if strings.Contains(text, "how to release") {
		// Send a message to the user
		attachment.Text = fmt.Sprintf("Ok <@%s>, you just need to mention me with this message format 'release projectname <<version>>'", user.ID)
		// attachment.Pretext = "How can I be of service"
		attachment.Footer = "Example: release logistics-backend <<backend-1.1.0-beta>>"
		attachment.Color = "#563a9b"
	} else if strings.Contains(text, "help") {
		// Send a message to the user
		attachment.Text = fmt.Sprintf("Howdy <@%s> :mixue:, this is the availble command list\n 1. how to schedule release \n 2. how to remove schedule \n 3. how to release \n 4. project list \n 5. who are you \n 6. schedule release ... \n 7. release ... \n 8. active schedule \n 9. remove schedule ...", user.ID)
		// attachment.Pretext = "How can I be of service"
		attachment.Footer = "GRIP Release Bot."
		attachment.Color = "#563a9b"
	} else if strings.Contains(text, "who are you") {
		// Send a message to the user
		attachment.Text = fmt.Sprintf("Yo <@%s>, I'm ~Snow King~ I mean Bot that handle release or deploy a project to server and living in GRIP Principle Slack 😁😁", user.ID)
		attachment.Footer = "Build using Go."
		attachment.Color = "#563a9b"
	} else if strings.Contains(text, "env") {
		// Send a message to the user
		attachment.Text = fmt.Sprintf("Bibop <@%s>, current env is set to %s", user.ID, config.GetConfig("ENVIRONMENT"))
		attachment.Footer = "Build using Go."
		attachment.Color = "#563a9b"
	} else if strings.Contains(text, "active schedule") {
		// Send a message to the user
		result := models.GetActiveRelease()
		if result == "" {
			attachment.Text = fmt.Sprintf("Woah <@%s>, currently no active schedule", user.ID)
		} else {
			attachment.Text = fmt.Sprintf("Wow <@%s>, this is the active schedule: \n\n %s", user.ID, result)
		}
		attachment.Footer = "Build using Go."
		attachment.Color = "#563a9b"
	} else if strings.Contains(text, "remove schedule") {
		if models.UserHasAccess((user.ID)) {

			re := regexp.MustCompile(`remove schedule ([^}]*).*`)
			match := re.FindStringSubmatch(text)
			fmt.Println(text)
			if match != nil {
				result := models.CheckActiveRelease(match[1])
				if !result {
					attachment.Text = fmt.Sprintf("Hmm <@%s>, no active schedule with this Id", user.ID)
				} else {
					models.UpdateReleased(match[1])
					attachment.Text = fmt.Sprintf("Noted <@%s>, this schedule is removed :noted:", user.ID)
				}
				attachment.Footer = "Build using Go."
				attachment.Color = "#4af030"

			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, You need to supply the active schedule id", user.ID)
				attachment.Color = "#e20228"
				attachment.Footer = fmt.Sprintf("GRIP Release Bot cannot continue, '%s'", text)
			}
		}
	} else if strings.Contains(text, "schedule release") {
		if models.UserHasAccess((user.ID)) {
			fmt.Println("schedule release is executed", text)
			re := regexp.MustCompile(`schedule release ([^}]*) \<<([^}]*)\>> at ([^}]*).*`)
			match := re.FindStringSubmatch(text)

			if match != nil {
				timeInput := strings.ToUpper(match[3])
				timeRegex := regexp.MustCompile(`^(0?[1-9]|1[012]):([0-5][0-9])[AP]M$`)
				timeMatch := timeRegex.FindStringSubmatch(timeInput)

				if timeMatch != nil {
					if contains(projectList, match[1]) {
						attachment.Text = fmt.Sprintf("Roger <@%s>, Create release schedule for %s version %s at %s", user.ID, match[1], match[2], timeInput)
						attachment.Color = "#4af030"
						attachment.Footer = "GRIP Release Bot create release schedule."

						models.CreateSchedule(match[1], match[2], timeInput, user.Name)
					} else {
						attachment.Text = fmt.Sprintf("Sorry <@%s>, the project %s is not found, use command project list to see the supported project", user.ID, match[1])
						attachment.Color = "#e20228"
						attachment.Footer = fmt.Sprintf("GRIP Release Bot cannot continue, '%s'", text)
					}
				} else {
					attachment.Text = fmt.Sprintf("Sorry <@%s>, looks like your time format is wrong, should be hh:mma in Asia/Singapore timezone, e.g 09:00PM", user.ID)
					attachment.Color = "#e20228"
					attachment.Footer = fmt.Sprintf("GRIP Release Bot cannot continue, '%s'", text)
				}

			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, you need to tell me the project name, version, and the release time, or make sure the message format is correct, use command 'how to schedule release' to see the format 😉", user.ID)
				attachment.Color = "#e20228"
				attachment.Footer = fmt.Sprintf("GRIP Release Bot cannot continue, '%s'", text)
			}
		} else {
			attachment.Text = "Sorry you don't have permission 🙏"
			attachment.Color = "#e20228"
			attachment.Footer = "GRIP Release Bot cannot continue"
		}
	} else if strings.Contains(text, "release") {
		if models.UserHasAccess((user.ID)) {

			re := regexp.MustCompile(`release ([^}]*) \<<([^}]*)\>>.*`)
			match := re.FindStringSubmatch(text)

			if match != nil {
				if contains(projectList, match[1]) {
					attachment.Text = fmt.Sprintf("Affirmative <@%s>, Releasing %s version %s now.", user.ID, match[1], match[2])
					attachment.Color = "#4af030"
					attachment.Footer = "GRIP Release Bot calling Jenkins..."

					callJenkins(match[1], match[2], false, "0000")
					log.Println(match[1], match[2])
				} else {
					attachment.Text = fmt.Sprintf("Sorry <@%s>, the project %s is not found, use command project list to see the supported project", user.ID, match[1])
					attachment.Color = "#e20228"
					attachment.Footer = fmt.Sprintf("GRIP Release Bot cannot continue, '%s'", text)
				}

			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, you need to tell me the project name & version, or make sure the message format is correct, use command 'how to release' to see the format 😉", user.ID)
				attachment.Color = "#e20228"
				attachment.Footer = fmt.Sprintf("GRIP Release Bot cannot continue, '%s'", text)
			}
		} else {
			attachment.Text = "Sorry you don't have permission 🙏"
			attachment.Color = "#e20228"
			attachment.Footer = "GRIP Release Bot cannot continue"
		}
	} else if strings.Contains(text, "access list") {
		if models.UserIsAdmin((user.ID)) {
			result := models.GetAllUsers()
			if result != "" {
				attachment.Text = fmt.Sprintf("Gotcha <@%s>, this is the access list: \n\n %s", user.ID, result)
			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, access list is empty.", user.ID)
			}
		} else {
			attachment.Text = fmt.Sprintf("Sorry <@%s>, you don't have the permission to do that", user.ID)
		}
	} else if strings.Contains(text, "add access") {
		if models.UserIsAdmin((user.ID)) {
			re := regexp.MustCompile(`add access ([^}]*)\-([^}]*).*`)
			match := re.FindStringSubmatch(text)
			if match != nil {
				attachment.Text = fmt.Sprintf("Roger <@%s>, Adding access for %s.", user.ID, match[2])

				models.AddNewUser(match[1], match[2])
			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, make sure the format is 'add access SLACKID-name'.", user.ID)
			}
		} else {
			attachment.Text = fmt.Sprintf("Sorry <@%s>, you don't have the permission to do that", user.ID)
		}
	} else if strings.Contains(text, "delete access") {
		if models.UserIsAdmin((user.ID)) {
			re := regexp.MustCompile(`delete access ([^}]*).*`)
			match := re.FindStringSubmatch(text)
			if match != nil {
				attachment.Text = fmt.Sprintf("Roger <@%s>, Removing access for %s.", user.ID, strings.ToUpper(match[1]))

				models.DeleteUserAccess(match[1])
			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, make sure the format is 'delete access SLACKID'.", user.ID)
			}
		} else {
			attachment.Text = fmt.Sprintf("Sorry <@%s>, you don't have the permission to do that", user.ID)
		}
	} else if strings.Contains(text, "enable access") {
		if models.UserIsAdmin((user.ID)) {
			re := regexp.MustCompile(`enable access ([^}]*).*`)
			match := re.FindStringSubmatch(text)
			if match != nil {
				attachment.Text = fmt.Sprintf("Roger <@%s>, Enabling access for %s.", user.ID, strings.ToUpper(match[1]))

				models.ToogleUserStatus(match[1], true)
			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, make sure the format is 'enable access SLACKID'.", user.ID)
			}
		} else {
			attachment.Text = fmt.Sprintf("Sorry <@%s>, you don't have the permission to do that", user.ID)
		}
	} else if strings.Contains(text, "disable access") {
		if models.UserIsAdmin((user.ID)) {
			re := regexp.MustCompile(`disable access ([^}]*).*`)
			match := re.FindStringSubmatch(text)
			if match != nil {
				attachment.Text = fmt.Sprintf("Roger <@%s>, Disabling access for %s.", user.ID, strings.ToUpper(match[1]))

				models.ToogleUserStatus(match[1], false)
			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, make sure the format is 'disable access SLACKID'.", user.ID)
			}
		} else {
			attachment.Text = fmt.Sprintf("Sorry <@%s>, you don't have the permission to do that", user.ID)
		}
	} else if strings.Contains(text, "test access") {
		if models.UserHasAccess((user.ID)) {
			attachment.Text = fmt.Sprintf("Congrats <@%s>, you have the access", user.ID)

		} else {
			attachment.Text = fmt.Sprintf("Sorry <@%s>, you don't have the permission to do that", user.ID)
		}
	} else if strings.Contains(text, "project list") {
		if models.UserIsAdmin((user.ID)) {
			result := models.GetAllProjects()
			if result != "" {
				attachment.Text = fmt.Sprintf("Gotcha <@%s>, this is the project list: \n\n %s", user.ID, result)
			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, project list is empty.", user.ID)
			}
		} else {
			attachment.Text = fmt.Sprintf("Sorry <@%s>, you don't have the permission to do that", user.ID)
		}
	} else if strings.Contains(text, "add project") {
		if models.UserIsAdmin((user.ID)) {
			re := regexp.MustCompile(`add project ([^}]*)\-([^}]*).*`)
			match := re.FindStringSubmatch(text)
			if match != nil {
				attachment.Text = fmt.Sprintf("Roger <@%s>, Adding project %s.", user.ID, match[1])

				models.AddNewProject(match[1], match[2])
			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, make sure the format is 'add project project name-jenkins token'.", user.ID)
			}
		} else {
			attachment.Text = fmt.Sprintf("Sorry <@%s>, you don't have the permission to do that", user.ID)
		}
	} else if strings.Contains(text, "delete project") {
		if models.UserIsAdmin((user.ID)) {
			re := regexp.MustCompile(`delete project ([^}]*).*`)
			match := re.FindStringSubmatch(text)
			if match != nil {
				attachment.Text = fmt.Sprintf("Roger <@%s>, Removing access for %s.", user.ID, strings.ToUpper(match[1]))

				models.DeleteProject(match[1])
			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, make sure the format is 'delete project project-name'.", user.ID)
			}
		} else {
			attachment.Text = fmt.Sprintf("Sorry <@%s>, you don't have the permission to do that", user.ID)
		}
	} else if strings.Contains(text, "enable project") {
		if models.UserIsAdmin((user.ID)) {
			re := regexp.MustCompile(`enable project ([^}]*).*`)
			match := re.FindStringSubmatch(text)
			if match != nil {
				attachment.Text = fmt.Sprintf("Roger <@%s>, Enabling project for %s.", user.ID, strings.ToUpper(match[1]))

				models.ToogleProject(match[1], true)
			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, make sure the format is 'enable project project-name'.", user.ID)
			}
		} else {
			attachment.Text = fmt.Sprintf("Sorry <@%s>, you don't have the permission to do that", user.ID)
		}
	} else if strings.Contains(text, "disable project") {
		if models.UserIsAdmin((user.ID)) {
			re := regexp.MustCompile(`disable project ([^}]*).*`)
			match := re.FindStringSubmatch(text)
			if match != nil {
				attachment.Text = fmt.Sprintf("Roger <@%s>, Disabling project for %s.", user.ID, strings.ToUpper(match[1]))

				models.ToogleProject(match[1], false)
			} else {
				attachment.Text = fmt.Sprintf("Sorry <@%s>, make sure the format is 'disable project project-name'.", user.ID)
			}
		} else {
			attachment.Text = fmt.Sprintf("Sorry <@%s>, you don't have the permission to do that", user.ID)
		}
	} else {
		if user.ID == "U023A0BJUB1" {
			attachment.Text = ":ice_cube: :tea:"
			attachment.Color = "#FF00FF"
		} else {
			// Send a message to the user
			attachment.Text = fmt.Sprintf("Hi <@%s>", user.ID)
			// attachment.Pretext = "How can I be of service"
			attachment.Color = "#563a9b"
		}
	}

	// Send the message to the channel
	// The Channel is available in the event message
	_, _, err = client.PostMessage(event.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}
	return nil
}

// self-explanatory
func callJenkins(project string, version string, isSchedule bool, time string) {
	env := config.GetConfig("ENVIRONMENT")
	isTesting := true
	jenkinsAddress := config.GetConfig("JENKINS_HOST")
	jenkinsToken := ""

	if env == "production" {
		isTesting = false
	}

	switch project {
	case "logistics-backend":
		log.Printf("Execute webhook logistics-backend")

		jenkinsToken = config.GetConfig("JENKINS_LOGISTICS_BACKEND_TOKEN")

	case "logistics-web":
		log.Printf("Execute webhook logistics-web")

		jenkinsToken = config.GetConfig("JENKINS_LOGISTICS_WEB_TOKEN")

	case "logistics-mobile":
		log.Printf("Execute webhook logistics-mobile")

		jenkinsToken = config.GetConfig("JENKINS_LOGISTICS_MOBILE_TOKEN")

	case "gla-platform":
		log.Printf("Execute webhook gla-platform")

		jenkinsToken = config.GetConfig("JENKINS_GLA_PLATFORM_TOKEN")

	case "gla-parent":
		log.Printf("Execute webhook gla-parent")

		jenkinsToken = config.GetConfig("JENKINS_GLA_PARENT_TOKEN")

	case "gla-admin":
		log.Printf("Execute webhook gla-admin")

		jenkinsToken = config.GetConfig("JENKINS_GLA_ADMIN_TOKEN")
	// case "smartapes":
	// 	log.Printf("Execute webhook smartapes")

	// 	jenkinsToken = config.GetConfig("JENKINS_SMARTAPES_TOKEN")

	default:
		log.Printf("Not calling webhooks")
	}

	jenkinsWebhook := "http://" + jenkinsAddress + "/generic-webhook-trigger/invoke?token=" + jenkinsToken + "&buildEnv=production&release_version=" + version + "&project_id=null&release_id=null&release_timer=" + strconv.FormatBool(isSchedule) + "&release_at=" + time + "&test_release=" + strconv.FormatBool(isTesting)

	// fmt.Println(jenkinsWebhook)
	_, err := http.Get(jenkinsWebhook)

	if err != nil {
		log.Println("error calling webhooks: " + err.Error())
	}
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
