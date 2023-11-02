package main

import (
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"
)

var c mqtt.Client

var myName string

type discoveryT struct {
	Name                string   `json:"name"`
	StateTopic          string   `json:"state_topic"`
	CommandTopic        string   `json:"command_topic"`
	UniqueId            string   `json:"unique_id"`
	AvailabilityTopic   string   `json:"availability_topic"`
	PayloadAvailable    string   `json:"payload_available"`
	PayloadNotAvailable string   `json:"payload_not_available"`
	Options             []string `json:"options"`
}

func (d *discoveryT) toString() string {
	out, _ := json.Marshal(d)
	return string(out)
}

var myDiscovery discoveryT

func autorandrList() ([]string, error) {
	cmd := exec.Command("autorandr", "--list")
	cmd.Stdin = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(output), "\n")
	var result []string
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

var mu sync.Mutex

func autorandrLoad(name string) error {
	mu.Lock()
	defer mu.Unlock()
	cmd := exec.Command("autorandr", "--load", name)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	return err

}

var lastConnect time.Time = time.Unix(0, 0)

func main() {
	var err error
	myName = os.Getenv("MY_NAME")
	if myName == "" {
		log.Fatalf("Env MY_NAME must be set!")
	}

	opts := mqtt.NewClientOptions().AddBroker(os.Getenv("MQTT_URL")).SetClientID(myName + "-autorandr2mqtt")
	if username := os.Getenv("MQTT_USERNAME"); username != "" {
		opts.SetUsername(username)
	}
	if password := os.Getenv("MQTT_PASSWORD"); password != "" {
		opts.SetPassword(password)
	}

	msgchan := make(chan mqtt.Message)

	opts.SetKeepAlive(20 * time.Second)
	opts.SetDefaultPublishHandler(func(_ mqtt.Client, message mqtt.Message) {
		msgchan <- message

	})
	myDiscovery.Name = myName
	myDiscovery.CommandTopic = "cmnd/autorandr2mqtt/" + myName + "/load"
	myDiscovery.StateTopic = "stat/autorandr2mqtt/" + myName + "/load"
	myDiscovery.PayloadAvailable = "online"
	myDiscovery.PayloadNotAvailable = "offline"
	myDiscovery.AvailabilityTopic = "stat/autorandr2mqtt/" + myName + "/status"
	myDiscovery.Options, err = autorandrList()
	if err != nil {
		log.Fatalf("%s", err)
	}
	opts.SetWill(myDiscovery.AvailabilityTopic, myDiscovery.PayloadNotAvailable, 0, true)
	opts.SetOnConnectHandler(func(_ mqtt.Client) {
		log.Printf("(re)connected")
		if time.Since(lastConnect) < time.Second {
			log.Fatalf("Reconnected AGAIN less than 1 second after previous reconnect. Is another instance with the same client ID running?")
		}
		lastConnect = time.Now()
		token := c.Subscribe(myDiscovery.CommandTopic, 0, nil)
		token.Wait()
		if token.Error() != nil {
			log.Fatal(token.Error())
		}
		token = c.Publish(myDiscovery.AvailabilityTopic, 0, true, myDiscovery.PayloadAvailable)
		token.Wait()
		if token.Error() != nil {
			log.Fatal(token.Error())
		}
		token = c.Publish("homeassistant/select/autorandr2mqtt_"+myName+"/config", 0, true, myDiscovery.toString())
		token.Wait()
		if token.Error() != nil {
			log.Fatal(token.Error())
		}

	})
	opts.SetPingTimeout(1 * time.Second)
	opts.SetAutoReconnect(true)
	c = mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	for {
		message := <-msgchan
		if message.Topic() == myDiscovery.CommandTopic {
			target := string(message.Payload())
			if slices.Contains(myDiscovery.Options, target) {
				//profile existed when we started
				err := autorandrLoad(target)
				if err != nil {
					log.Printf("Failed to load profile: %s", err)
				} else {
					token := c.Publish(myDiscovery.StateTopic, 0, true, target)
					token.Wait()
					if token.Error() != nil {
						log.Fatal(token.Error())
					}
				}

			} else {
				log.Printf("Profile %s does not exist", target)
			}
		}
	}

}
