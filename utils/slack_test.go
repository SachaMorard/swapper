package utils

import (
	"github.com/sachamorard/swapper/yaml"
	"testing"
)

func TestSlackPush(t *testing.T) {
	err := SlackPush("test", "#FF0000", "test", "https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS")
	if err.Error() != "Post https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS: dial tcp: lookup hooks.slackss.com: no such host" {
		t.Fail()
	}
}

func TestSlackSendSuccess(t *testing.T) {
	slack := yaml.Slack{WebHookUrl: "https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS", Channel: "test"}
	yamlConf := yaml.YamlConf{Slack: slack}
	err := SlackSendSuccess("test", yamlConf)
	if err.Error() != "Post https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS: dial tcp: lookup hooks.slackss.com: no such host" {
		t.Fail()
	}
}

func TestSlackSendError(t *testing.T) {
	slack := yaml.Slack{WebHookUrl: "https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS", Channel: "test"}
	yamlConf := yaml.YamlConf{Slack: slack}
	err := SlackSendError("test", yamlConf)
	if err.Error() != "Post https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS: dial tcp: lookup hooks.slackss.com: no such host" {
		t.Fail()
	}
}
