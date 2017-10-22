package main

import (
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

const AppName = "cw2slack"
const AppVersion = "0.0.1"
const BaseURL = "https://api.chatwork.com/v2"

type Config struct {
	ChatWorkToken       string `toml:"chatwork_token"`
	SlackEndpoint       string `toml:"slack_endpoint"`
	SlackDefaultChannel string `toml:"slack_channel"`
	Mappings            map[string]ConfigMapping
}

type ConfigMapping struct {
	Room    string
	Channel string
}

type Room struct {
	RoomId   int    `json:"room_id"`
	Name     string `json:"name"`
	IconPath string `json:"icon_path"`
}

type Message struct {
	MessageId string         `json:"message_id"`
	Body      string         `json:"body"`
	Account   MessageAccount `json:"account"`
	SendTime  int            `json:"send_time"`
}

type MessageAccount struct {
	Name           string `json:"name"`
	AvatarImageURL string `json:"avatar_image_url"`
}

type Slack struct {
	Username   string            `json:"username"`
	Channel    string            `json:"channel"`
	IconURL    string            `json:"icon_url"`
	Attachment []SlackAttachment `json:"attachments"`
}

type SlackAttachment struct {
	Text       string `json:"text"`
	AuthorName string `json:"author_name"`
	AuthorIcon string `json:"author_icon"`
	AuthorLink string `json:"author_link"`
	Color      string `json:"color"`
	Footer     string `json:"footer"`
	Timestamp  int    `json:"ts"`
}

var config Config

func main() {
	config = loadConfig()

	app := cli.NewApp()
	app.Name = AppName
	app.Version = AppVersion
	app.Usage = ""
	app.Action = action
	app.Run(os.Args)
}

func action(c *cli.Context) error {
	rooms := rooms()
	for _, r := range rooms {
		messages := messages(r.RoomId)
		if len(messages) > 0 {
			channel := config.SlackDefaultChannel

			for _, mapping := range config.Mappings {
				mr, _ := strconv.Atoi(mapping.Room)
				if mr == r.RoomId {
					channel = mapping.Channel
				}
			}

			for _, m := range messages {
				url := "https://chatwork.com/#!rid/" + strconv.Itoa(r.RoomId) + "-" + m.MessageId
				attachments := []SlackAttachment{SlackAttachment{
					m.Body,
					m.Account.Name,
					m.Account.AvatarImageURL,
					url,
					"#EEEEEE",
					url,
					m.SendTime,
				}}
				postToSlack(Slack{
					r.Name,
					channel,
					r.IconPath,
					attachments,
				})
			}
		}
	}
	return nil
}

func rooms() []Room {
	req, err := http.NewRequest("GET", BaseURL+"/rooms", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("X-ChatWorkToken", config.ChatWorkToken)
	client := new(http.Client)
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		log.Fatal(fmt.Errorf("Error: %d\n", res.StatusCode))
	}
	defer res.Body.Close()
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	var rooms []Room
	err = json.Unmarshal(bytes, &rooms)
	if err != nil {
		log.Fatal(err)
	}
	return rooms
}

func messages(id int) []Message {
	req, err := http.NewRequest("GET", BaseURL+"/rooms/"+strconv.Itoa(id)+"/messages", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("X-ChatWorkToken", config.ChatWorkToken)
	client := new(http.Client)
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		log.Fatal(fmt.Errorf("Error: %d\n", res.StatusCode))
	}
	defer res.Body.Close()
	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	var messages []Message
	if len(string(bytes)) > 0 {
		err = json.Unmarshal(bytes, &messages)
		if err != nil {
			log.Fatal(err)
		}
	}
	return messages
}

func postToSlack(r Slack) {
	p, err := json.Marshal(r)
	if err != nil {
		log.Fatal(err)
	}
	_, err = http.PostForm(config.SlackEndpoint, url.Values{"payload": {string(p)}})
	if err != nil {
		log.Fatal(err)
	}
}

func loadConfig() Config {
	home := os.Getenv("HOME")
	filename := filepath.Join(home, ".config", AppName, "config.toml")
	var c Config
	_, err := toml.DecodeFile(filename, &c)
	if err != nil {
		log.Fatal(err)
	}
	return c
}
