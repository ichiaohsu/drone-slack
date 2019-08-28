package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bluele/slack"
	"github.com/drone/drone-template-lib/template"
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

	// Parse Recipient
	usermaps := make(map[string]string)
	if p.Config.Usermaps != "" {
		if err := json.Unmarshal([]byte(p.Config.Usermaps), &usermaps); err != nil {
			return err
		}
	}
	if p.Config.Recipient != "" {
		if val, ok := usermaps[p.Config.Recipient]; ok {
			p.Config.Recipient = val
		}
		payload.Channel = prepend("@", p.Config.Recipient)
	} else if p.Config.Channel != "" {
		payload.Channel = prepend("#", p.Config.Channel)
	}
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

	targetList := make([]string, 0)
	if p.Config.Recipient != "" {
		fmt.Printf("Config.Recipient:%s\n", p.Config.Recipient)
		recipients := strings.Split(p.Config.Recipient, ",")
		for _, recipient := range recipients {

			if val, ok := usermaps[recipient]; ok {
				// p.Config.Recipient = val
				targetList = append(targetList, prepend("@", val))
			} else {
				targetList = append(targetList, prepend("@", recipient))
			}
		}
		// payload.Channel = prepend("@", p.Config.Recipient)
	}
	if p.Config.Channel != "" {
		channels := strings.Split(p.Config.Channel, ",")
		for _, channel := range channels {
			// payload.Channel = prepend("#", p.Config.Channel)
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
