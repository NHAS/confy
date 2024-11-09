package confy

import (
	"reflect"
	"testing"
)

type testTags struct {
	Notification struct {
		SMTP struct {
			Enabled bool `confy:"enabled" confy_description:"Enable or disable sending notifications via SMTP"`

			Host      string `confy:"host" confy_description:"Host domain/ip"`
			Port      int    `confy:"port" confy_description:"Port"`
			Username  string `confy:"username" confy_description:"Mailing username"`
			Password  string `confy:"password;sensitive" confy_description:"Mailing password"`
			FromEmail string `confy:"from" confy_description:"The sending email address"`
		}

		Webhooks struct {
			Enabled     bool     `confy:"enabled" confy_description:"Enable or disable sending notifications via webhooks"`
			SafeDomains []string `confy:"safe_domains" confy_description:"List of domains that are safe to send to, defaults to [discord.com, slack.com]"`
		}

		Confidential bool `confy:"confidential" confy_description:"Whether to add xss vulnerablity details to notification"`
	}
}

func TestResolvePath(t *testing.T) {
	var c testTags

	expected := []string{
		"Notification",
		"SMTP",
		"enabled",
	}

	actual := resolvePath(&c, []string{"Notification", "SMTP", "Enabled"})

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("resolved path incorrect, expected %v got %v", expected, actual)
	}

}
