package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sachamorard/swapper/response"
	"github.com/sachamorard/swapper/yaml"
	"net/http"
	"time"
)


type SlackRequestBody struct {
	Text string `json:"text"`
	Channel string `json:"channel"`
	Username string  `json:"username"`
	Emoji string `json:"icon_emoji"`
	Attachments []SlackAttachment `json:"attachments"`
}

type SlackAttachment struct {
	Text string `json:"text"`
	Color string `json:"color"`
}

func SlackSendSuccess(message string, yamlConf yaml.YamlConf) (err error) {
	if yamlConf.Slack.WebHookUrl != "" && yamlConf.Slack.Channel != "" {
		return SlackPush(message, "#00FF00", yamlConf.Slack.Channel, yamlConf.Slack.WebHookUrl)
	}
	return nil
}

func SlackSendError(message string, yamlConf yaml.YamlConf) (err error) {
	if yamlConf.Slack.WebHookUrl != "" && yamlConf.Slack.Channel != "" {
		return SlackPush(message, "#D0021B", yamlConf.Slack.Channel, yamlConf.Slack.WebHookUrl)
	}
	return nil
}

func SlackPush(message string, color string, channel string, webhookUrl string) (err error) {

	hostname, _ := GetHostname()
	att := SlackAttachment{Text: "*"+hostname+"*\n"+message, Color: color}
	var atts []SlackAttachment
	atts = append(atts, att)
	slackBody, _ := json.Marshal(SlackRequestBody{
		Channel: channel,
		Username: "swapper",
		Emoji: ":robot_face:",
		Attachments: atts,
	})
	req, err := http.NewRequest(http.MethodPost, webhookUrl, bytes.NewBuffer(slackBody))
	if err != nil {
		return errors.New(fmt.Sprintf(response.ErrorMessages["request_failed"], err))
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
