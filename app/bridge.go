package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"wallbox-mqtt-bridge/app/wallbox"
)

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	panic("Connection to MQTT lost")
}

func RunBridge(configPath string) {
	c := LoadConfig(configPath)
	w := wallbox.New()
	w.RefreshData()

	serialNumber := w.SerialNumber()
	entityConfig := getEntities(w)
	if c.Settings.DebugSensors {
		for k, v := range getDebugEntities(w) {
			entityConfig[k] = v
		}
		for k, v := range getTelemetryEventEntities(w) {
			entityConfig[k] = v
		}
	}

	if c.Settings.PowerBoostEnabled {
		for k, v := range getPowerBoostEntities(w, c) {
			entityConfig[k] = v
		}
	}

	topicPrefix := "wallbox_" + serialNumber
	availabilityTopic := topicPrefix + "/availability"

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", c.MQTT.Host, c.MQTT.Port))
	opts.SetUsername(c.MQTT.Username)
	opts.SetPassword(c.MQTT.Password)
	opts.SetWill(availabilityTopic, "offline", 1, true)
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	for key, val := range entityConfig {
		component := val.Component
		uid := serialNumber + "_" + key
		config := map[string]interface{}{
			"~":                  topicPrefix + "/" + key,
			"availability_topic": availabilityTopic,
			"state_topic":        "~/state",
			"unique_id":          uid,
			"device": map[string]string{
				"identifiers": serialNumber,
				"name":        c.Settings.DeviceName,
			},
		}
		if val.Setter != nil {
			config["command_topic"] = "~/set"
		}
		for k, v := range val.Config {
			config[k] = v
		}
		jsonPayload, _ := json.Marshal(config)
		token := client.Publish("homeassistant/"+component+"/"+uid+"/config", 1, true, jsonPayload)
		token.Wait()
	}

	token := client.Publish(availabilityTopic, 1, true, "online")
	token.Wait()

	messageHandler := func(client mqtt.Client, msg mqtt.Message) {
		field := strings.Split(msg.Topic(), "/")[1]
		payload := string(msg.Payload())
		setter := entityConfig[field].Setter
		fmt.Println("Setting", field, payload)
		setter(payload)
	}

	topic := topicPrefix + "/+/set"
	client.Subscribe(topic, 1, messageHandler)

	ticker := time.NewTicker(time.Duration(c.Settings.PollingIntervalSeconds) * time.Second)
	defer ticker.Stop()

	published := make(map[string]interface{})

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// for debugging purposes, only run for the first 2 minutes
	// w.StartTimeConstrainedRedisSubscriptions(2 * time.Minute)
	w.StartRedisSubscriptions()

	for {
		select {
		case <-ticker.C:
			w.RefreshData()
			for key, val := range entityConfig {
				payload := val.Getter()
				bytePayload := []byte(fmt.Sprint(payload))
				if published[key] != payload {
					if val.RateLimit != nil && !val.RateLimit.Allow(strToFloat(payload)) {
						continue
					}
					fmt.Println("Publishing: ", key, payload)
					token := client.Publish(topicPrefix+"/"+key+"/state", 1, true, bytePayload)
					token.Wait()
					published[key] = payload
				}
			}
		case <-interrupt:
			fmt.Println("Interrupted. Exiting...")
			token := client.Publish(availabilityTopic, 1, true, "offline")
			token.Wait()
			client.Disconnect(250)
			return
		}
	}

	w.StopRedisSubscriptions()
}
