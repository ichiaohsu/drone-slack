package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/bluele/slack"
	"github.com/drone/drone-template-lib/template"
	"github.com/google/go-github/v28/github"
)

type (
	Repo struct {
		Owner string
		Name  string
	}

	Build struct {
		Tag         string
		Event       string
		Number      int
		Commit      string
		Ref         string
		Branch      string
		Author      string
		Pull        string
		Message     string
		DeployTo    string
		Status      string
		Link        string
		Started     int64
		Created     int64
		ServiceAddr string
	}

	Config struct {
		Webhook   string
		Channel   string
		Recipient string
		Username  string
		Template  string
		ImageURL  string
		IconURL   string
		IconEmoji string
		LinkNames bool
		Usermaps  string
		PRMessage bool
	}

	Job struct {
		Started int64
	}

	Plugin struct {
		Repo   Repo
		Build  Build
		Config Config
		Job    Job
	}
)

func (p Plugin) Exec() error {
	attachment := slack.Attachment{
		Text:       message(p.Repo, p.Build),
		Fallback:   fallback(p.Repo, p.Build),
		Color:      color(p.Build),
		MarkdownIn: []string{"text", "fallback"},
		ImageURL:   p.Config.ImageURL,
	}

	payload := slack.WebHookPostPayload{}
	payload.Username = p.Config.Username
	payload.Attachments = []*slack.Attachment{&attachment}
	payload.IconUrl = p.Config.IconURL
	payload.IconEmoji = p.Config.IconEmoji

	if p.Config.LinkNames == true {
		payload.LinkNames = "1"
	}
	if p.Config.Template != "" {
		txt, err := template.RenderTrim(p.Config.Template, p)
		if err != nil {
			return err
		}
		attachment.Text = txt
	}
	// Generate extra message when pr-message: true
	if p.Config.PRMessage {
		pullNum, err := strconv.Atoi(p.Build.Pull)
		if err != nil {
			return err
		}
		pr, err := pullMessage(p.Repo, pullNum)
		if err != nil {
			return err
		}
		prAttachment := slack.Attachment{
			Fallback:   fallback(p.Repo, p.Build),
			Color:      color(p.Build),
			MarkdownIn: []string{"text", "fallback"},
			ImageURL:   p.Config.ImageURL,

			Pretext:    "The pull request content for this build is listed below",
			AuthorName: p.Build.Author,
			Title:      *pr.Title,
			TitleLink:  *pr.HTMLURL,
			Text:       *pr.Body,
		}
		payload.Attachments = append(payload.Attachments, &prAttachment)
	}

	// Parse Recipient
	usermaps := make(map[string]string)
	if p.Config.Usermaps != "" {
		if err := json.Unmarshal([]byte(p.Config.Usermaps), &usermaps); err != nil {
			return err
		}
	}
	// Parse all target, including channels
	targetList := make([]string, 0)
	if p.Config.Recipient != "" {
		recipients := strings.Split(p.Config.Recipient, ",")
		for _, recipient := range recipients {

			if val, ok := usermaps[recipient]; ok {
				targetList = append(targetList, prepend("@", val))
			} else {
				targetList = append(targetList, prepend("@", recipient))
			}
		}
	}
	if p.Config.Channel != "" {
		channels := strings.Split(p.Config.Channel, ",")
		for _, channel := range channels {
			targetList = append(targetList, prepend("#", channel))
		}
	}
	client := slack.NewWebHook(p.Config.Webhook)
	var err error
	for _, receiver := range targetList {
		payload.Channel = receiver
		err = client.PostMessage(&payload)
	}
	return err
}

func message(repo Repo, build Build) string {
	return fmt.Sprintf("*%s* <%s|%s/%s#%s> (%s) by %s",
		build.Status,
		build.Link,
		repo.Owner,
		repo.Name,
		build.Commit[:8],
		build.Branch,
		build.Author,
	)
}
func pullMessage(repo Repo, pullNum int) (*github.PullRequest, error) {
	client := github.NewClient(nil)
	var ctx = context.Background()

	pr, _, err := client.PullRequests.Get(ctx, repo.Owner, repo.Name, 947)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func fallback(repo Repo, build Build) string {
	return fmt.Sprintf("%s %s/%s#%s (%s) by %s",
		build.Status,
		repo.Owner,
		repo.Name,
		build.Commit[:8],
		build.Branch,
		build.Author,
	)
}

func color(build Build) string {
	switch build.Status {
	case "success":
		return "good"
	case "failure", "error", "killed":
		return "danger"
	default:
		return "warning"
	}
}

func prepend(prefix, s string) string {
	if !strings.HasPrefix(s, prefix) {
		return prefix + s
	}

	return s
}
