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
			fmt.Printf("Usermaps parsing error:%s\n", err.Error())
			return err
		}
	}
	fmt.Printf("Usermaps:%v\n", usermaps)
	if p.Config.Recipient != "" {
		fmt.Printf("Recipient before usermap check:%s\n", p.Config.Recipient)
		if val, ok := usermaps[p.Config.Recipient]; ok {
			p.Config.Recipient = val
		}
		fmt.Printf("Recipient after usermap check:%s\n", p.Config.Recipient)
		payload.Channel = prepend("@", p.Config.Recipient)
	} else if p.Config.Channel != "" {
		payload.Channel = prepend("#", p.Config.Channel)
	}
	if p.Config.LinkNames == true {
		payload.LinkNames = "1"
	}
	fmt.Printf("The whole p structure:%v\n", p)
	fmt.Printf("service address:%s\n", p.Build.ServiceAddr)
	if p.Config.Template != "" {
		txt, err := template.RenderTrim(p.Config.Template, p)
		fmt.Printf(txt)
		if err != nil {
			fmt.Printf("template rendering error:%s\n", err.Error())
			return err
		}

		attachment.Text = txt
	}

	client := slack.NewWebHook(p.Config.Webhook)
	return client.PostMessage(&payload)
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
