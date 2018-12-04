package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"gopkg.in/go-playground/webhooks.v5/github"
	tb "gopkg.in/tucnak/telebot.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var telegramToken string
var allowedGroup string
var githubSecret string
var group *Group

type Group struct {
	GroupRecipient string
}

func (g *Group) Recipient() string {
	return g.GroupRecipient
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatalln("No .env file found :(")
	}

	telegramToken = os.Getenv("TELEGRAM_TOKEN")
	allowedGroup = os.Getenv("ALLOWED_GROUP")
	githubSecret = os.Getenv("GITHUB_SECRET")

	rawGroup, err := ioutil.ReadFile(".group")
	if err == nil {
		json.Unmarshal(rawGroup, &group)
	} else {
		log.Println(err)
		log.Println("No .group file found")
	}

	bot, err := tb.NewBot(tb.Settings{
		Token:  telegramToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatalln(err)
	}

	bot.Handle("/start", func(m *tb.Message) {

		if m.Chat.Title != allowedGroup {
			log.Printf("(Username:%s id: %d from %s) tried to access bot >:(\n", m.Sender.Username, m.Sender.ID, m.Chat.Title)
			return
		}

		if group == nil {
			group = &Group{m.Chat.Recipient()}

			json, _ := json.Marshal(group)
			ioutil.WriteFile(".group", json, 0644)

			bot.Send(group, "Sälü, ig bi dr ESE 🤖")
			return
		}

		group = &Group{m.Chat.Recipient()}
		bot.Send(group, "Ig läbe noh 🏋️‍")

		json, _ := json.Marshal(group)
		ioutil.WriteFile(".group", json, os.ModePerm)
	})

	go bot.Start()

	hook, _ := github.New(github.Options.Secret(githubSecret))

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintln(writer, "Hello world!")
	})

	http.HandleFunc("/web", func(w http.ResponseWriter, r *http.Request) {
		payload, err := hook.Parse(r, github.PullRequestEvent, github.PullRequestReviewEvent)

		if err != nil {
			log.Println(err)
			if err == github.ErrEventNotFound {
				// ok event wasn;t one of the ones asked to be parsed
			}
		}
		switch payload.(type) {

		/*case github.PullRequestReviewPayload:
		pullRequestReview :=payload.(github.PullRequestReviewPayload)
		if group != nil {
			message := createPullRequestReviewMessage(pullRequestReview)
			if message != "" {
				bot.Send(group, message, tb.ModeMarkdown)
			}
		}*/
		case github.PullRequestPayload:
			pullRequest := payload.(github.PullRequestPayload)
			if group != nil {
				message := createPullRequestCreateMessage(pullRequest)
				if message != "" {
					_, err := bot.Send(group, message, tb.ModeMarkdown)
					if err != nil {
						log.Println("Error")
						log.Println(message)
						log.Println(err)
					}
				}
			}
		}
	})

	log.Println("Running 🤖")
	http.ListenAndServe(":3000", nil)
}

func createPullRequestCreateMessage(pullRequest github.PullRequestPayload) string {

	var builder strings.Builder

	if pullRequest.Action == "closed" {
		log.Println(pullRequest.PullRequest)

		if pullRequest.PullRequest.Merged {
			builder.WriteString(fmt.Sprintf("*Pull request* [%s](%s) wurde von *%s* auf _%s_ *gemergt* ❤️", pullRequest.PullRequest.Title, pullRequest.PullRequest.HTMLURL, pullRequest.PullRequest.MergedBy.Login, pullRequest.PullRequest.Base.Ref))
			return builder.String()
		}

		builder.WriteString(fmt.Sprintf("*Pull request* [%s](%s) wurde *geschlossen*.", pullRequest.PullRequest.Title, pullRequest.PullRequest.HTMLURL))
		return builder.String()
	}

	if pullRequest.Action == "reopened" {
		builder.WriteString(fmt.Sprintf("*Pull request* [%s](%s) wurde *wieder geöffnet* 🤔", pullRequest.PullRequest.Title, pullRequest.PullRequest.HTMLURL))
		return builder.String()
	}

	if pullRequest.Action == "review_requested" {

		var reviewers strings.Builder
		for i, v := range pullRequest.PullRequest.RequestedReviewers {
			reviewers.WriteString(fmt.Sprintf("*%s*", v.Login))
			if i < len(pullRequest.PullRequest.RequestedReviewers)-1 {
				reviewers.WriteString(", ")
			}

		}

		builder.WriteString(fmt.Sprintf("*Review* von %s für [%s](%s)", reviewers.String(), pullRequest.PullRequest.Title, pullRequest.PullRequest.HTMLURL))
		builder.WriteString(" angefordert ❤️")

		return builder.String()
	}

	if pullRequest.Action == "opened" {
		builder.WriteString(fmt.Sprintf("*Pull request* [%s](%s) von *%s* eröffnet 🤩", pullRequest.PullRequest.Title, pullRequest.PullRequest.HTMLURL, pullRequest.Sender.Login))

		if len(pullRequest.PullRequest.RequestedReviewers) > 0 {
			var reviewers strings.Builder
			for i, v := range pullRequest.PullRequest.RequestedReviewers {
				reviewers.WriteString(fmt.Sprintf("*%s*", v.Login))
				if i < len(pullRequest.PullRequest.RequestedReviewers)-1 {
					reviewers.WriteString(", ")
				}

			}

			var reviewersMessage strings.Builder

			reviewersMessage.WriteString(fmt.Sprintf("*Review* von "))
			reviewersMessage.WriteString(fmt.Sprintf("%s", reviewers.String()))
			reviewersMessage.WriteString(fmt.Sprintf(" für* [%s](%s)", pullRequest.PullRequest.Title, pullRequest.PullRequest.HTMLURL))
			reviewersMessage.WriteString(" angefordert ❤️")

			builder.WriteString("\n\n")
			builder.WriteString(reviewersMessage.String())
		}

		return builder.String()
	}

	return ""
}

func createPullRequestReviewMessage(reviewRequest github.PullRequestReviewPayload) string {
	log.Println("Review request")
	log.Println(reviewRequest.Action)

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("*Review für* [%s](%s)", reviewRequest.PullRequest.Title, reviewRequest.PullRequest.HTMLURL))
	builder.WriteString(" von ")
	builder.WriteString(fmt.Sprintf("%s", reviewRequest.Review.User.Login))
	builder.WriteString(" ❤️")

	return builder.String()
}
