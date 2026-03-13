package bridge

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"wallbox-mqtt-bridge/app/wallbox"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	buildVersion = "dev"
)

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	panic("Connection to MQTT lost")
}

func RunBridge(configPath string) {
	c := LoadConfig(configPath)
	if c.Settings.OCPPMismatchSeconds == 0 {
		c.Settings.OCPPMismatchSeconds = 60
	}
	if c.Settings.OCPPRestartCooldown == 0 {
		c.Settings.OCPPRestartCooldown = 600
	}
	if c.Settings.OCPPMaxRestarts <= 0 {
		// Default to a small number of restart attempts before either giving up
		// or escalating to a full reboot (if enabled).
		c.Settings.OCPPMaxRestarts = 3
	}
	if c.Settings.PilotErrorSeconds == 0 {
		c.Settings.PilotErrorSeconds = 300
	}

	w := wallbox.New()
	w.RefreshData()
	w.StartRedisSubscriptions()
	w.StartOCPPJournalWatcher()
	defer w.StopRedisSubscriptions()
	defer w.StopOCPPJournalWatcher()

	serialNumber := w.SerialNumber()
	firmwareVersion := w.FirmwareVersion()
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

	ocppMismatchState := "0"
	ocppLastRestart := "never"
	ocppLastHealAction := "idle"
	ocppLastHealAt := "never"
	ocppLastHealDetail := ""
	var mismatchStart time.Time
	var lastRestart time.Time
	var ocppRestartCount int
	var lastFullReboot time.Time
	var pilotErrorStart time.Time
	var lastPilotErrorReboot time.Time

	entityConfig["ocpp_mismatch"] = Entity{
		Component: "binary_sensor",
		Getter:    func() string { return ocppMismatchState },
		Config: map[string]string{
			"name":            "OCPP mismatch",
			"payload_on":      "1",
			"payload_off":     "0",
			"device_class":    "problem",
			"entity_category": "diagnostic",
		},
	}

	entityConfig["ocpp_last_restart"] = Entity{
		Component: "sensor",
		Getter:    func() string { return ocppLastRestart },
		Config: map[string]string{
			"name":            "OCPP last restart",
			"entity_category": "diagnostic",
		},
	}

	entityConfig["ocpp_enabled"] = Entity{
		Component: "binary_sensor",
		Getter:    w.OCPPEnabled,
		Config: map[string]string{
			"name":         "OCPP enabled",
			"payload_on":   "1",
			"payload_off":  "0",
			"device_class": "power",
		},
	}

	entityConfig["ocpp_connected"] = Entity{
		Component: "binary_sensor",
		Getter:    w.OCPPConnected,
		Config: map[string]string{
			"name":         "OCPP connected",
			"payload_on":   "1",
			"payload_off":  "0",
			"device_class": "connectivity",
		},
	}

	entityConfig["ocpp_last_heal_action"] = Entity{
		Component: "sensor",
		Getter:    func() string { return ocppLastHealAction },
		Config: map[string]string{
			"name":            "OCPP last heal action",
			"entity_category": "diagnostic",
		},
	}

	entityConfig["ocpp_last_heal_at"] = Entity{
		Component: "sensor",
		Getter:    func() string { return ocppLastHealAt },
		Config: map[string]string{
			"name":            "OCPP last heal at",
			"entity_category": "diagnostic",
		},
	}

	entityConfig["ocpp_last_heal_detail"] = Entity{
		Component: "sensor",
		Getter: func() string {
			if ocppLastHealDetail == "" {
				return "none"
			}
			return ocppLastHealDetail
		},
		Config: map[string]string{
			"name":            "OCPP last heal detail",
			"entity_category": "diagnostic",
		},
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
				"sw_version":  fmt.Sprintf("%s (FW %s)", bridgeVersion(), firmwareVersion),
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

	for {
		select {
		case <-ticker.C:
			w.RefreshData()
			now := time.Now()

			pilotConnected := w.HasTelemetry && (w.CableConnected() == 1 || w.IsChargingPilot())
			ocppCode := w.OCPPStatusCode()
			ocppIndicatesDisconnect := w.OCPPIndicatesDisconnect()

			if pilotConnected && ocppIndicatesDisconnect {
				if mismatchStart.IsZero() {
					mismatchStart = now
					ocppRestartCount = 0
					log.Printf("OCPP mismatch detected: pilot=%d (%s), OCPP=%d (%s)", w.ControlPilotCode(), w.ControlPilotStatus(), ocppCode, w.OCPPStatusDescription())
				}
				ocppMismatchState = "1"
			} else {
				if ocppMismatchState != "0" {
					log.Println("OCPP mismatch cleared")
				}
				ocppMismatchState = "0"
				mismatchStart = time.Time{}
				ocppRestartCount = 0
			}

			if c.Settings.AutoRestartOCPP && ocppMismatchState == "1" && !mismatchStart.IsZero() {
				threshold := time.Duration(c.Settings.OCPPMismatchSeconds) * time.Second
				cooldown := time.Duration(c.Settings.OCPPRestartCooldown) * time.Second

				if now.Sub(mismatchStart) >= threshold && (lastRestart.IsZero() || now.Sub(lastRestart) >= cooldown) {
					// First try a bounded number of OCPP service restarts. If
					// those do not clear the mismatch and full reboot is
					// enabled, we can optionally escalate to a complete
					// Wallbox reboot as a last resort.
					if c.Settings.OCPPMaxRestarts == 0 || ocppRestartCount < c.Settings.OCPPMaxRestarts {
						log.Printf("Restarting ocppwallbox.service after %s mismatch (OCPP %d: %s) [attempt %d/%d]",
							now.Sub(mismatchStart).Round(time.Second), ocppCode, w.OCPPStatusDescription(), ocppRestartCount+1, c.Settings.OCPPMaxRestarts)
						action, detail, err := restartCriticalServices()
						ocppLastHealAction = action
						ocppLastHealDetail = detail
						ocppLastHealAt = now.Format(time.RFC3339)
						if err != nil {
							log.Printf("Failed to restart charging stack: %v", err)
							continue
						}
						ocppRestartCount++
						lastRestart = now
						mismatchStart = now
						ocppLastRestart = now.Format(time.RFC3339)
					} else if c.Settings.OCPPFullReboot {
						// Only perform a full reboot if we have not recently done so.
						if lastFullReboot.IsZero() || now.Sub(lastFullReboot) >= cooldown {
							log.Printf("Escalating to full system reboot after %d failed OCPP restart attempts and %s mismatch (OCPP %d: %s)",
								ocppRestartCount, now.Sub(mismatchStart).Round(time.Second), ocppCode, w.OCPPStatusDescription())
							go func() {
								if err := rebootSystem(); err != nil {
									log.Printf("Failed to reboot system for OCPP heal: %v", err)
								}
							}()
							lastFullReboot = now
						}
					}
				}
			}

			// Independent safety net: if control pilot reports error state 14 for a sustained period, reboot.
			if c.Settings.PilotErrorReboot {
				if w.ControlPilotCode() == 14 {
					if pilotErrorStart.IsZero() {
						pilotErrorStart = now
						log.Printf("Control pilot entered error state 14; starting reboot timer (%ds)", c.Settings.PilotErrorSeconds)
					}
					if now.Sub(pilotErrorStart) >= time.Duration(c.Settings.PilotErrorSeconds)*time.Second {
						if lastPilotErrorReboot.IsZero() || now.Sub(lastPilotErrorReboot) >= time.Duration(c.Settings.PilotErrorSeconds)*time.Second {
							log.Printf("Rebooting due to sustained control pilot error state 14 for %s", now.Sub(pilotErrorStart).Round(time.Second))
							go func() {
								if err := rebootSystem(); err != nil {
									log.Printf("Failed to reboot after control pilot error: %v", err)
								}
							}()
							lastPilotErrorReboot = now
							pilotErrorStart = time.Time{}
						}
					}
				} else {
					pilotErrorStart = time.Time{}
				}
			}

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

func restartCriticalServices() (action string, detail string, err error) {
	// Basic dependency sanity checks. If Redis/MySQL are down, restarting OCPP
	// will likely flap; log but do not block the heal.
	checkService := func(name string) {
		if err := exec.Command("systemctl", "is-active", "--quiet", name).Run(); err != nil {
			log.Printf("warning: dependency %s is not active: %v", name, err)
		}
	}
	checkService("redis.service")
	checkService("mysqld.service")

	services := []string{
		"ocppwallbox.service",
	}

	for _, svc := range services {
		// Prefer a graceful stop + start to let the service flush state.
		stopCmd := exec.Command("systemctl", "stop", svc)
		startCmd := exec.Command("systemctl", "start", svc)

		stopErr := stopCmd.Run()
		if stopErr == nil {
			log.Printf("heal: stopped %s", svc)
			if startErr := startCmd.Run(); startErr == nil {
				log.Printf("heal: started %s", svc)
				return "stop_start", fmt.Sprintf("%s stopped+started", svc), nil
			}
			log.Printf("heal: start %s failed after stop, will retry with restart", svc)
		} else {
			log.Printf("heal: stop %s failed (%v), will retry with restart", svc, stopErr)
		}

		// If stop/start fails, fall back to a direct restart.
		restartCmd := exec.Command("systemctl", "restart", svc)
		if err := restartCmd.Run(); err != nil {
			// As a final safeguard, invoke the Wallbox reboot flow.
			log.Printf("restart %s failed (%v); escalating to full reboot", svc, err)
			if rebootErr := rebootSystem(); rebootErr != nil {
				return "reboot", fmt.Sprintf("reboot failed after restart error: %v", rebootErr), rebootErr
			}
			return "reboot", "reboot issued after restart failure", nil
		}
		log.Printf("heal: restarted %s via systemctl restart", svc)
		return "restart", fmt.Sprintf("%s restarted", svc), nil
	}

	return "noop", "no services to restart", nil
}

// rebootSystem triggers the Wallbox-provided reboot flow. Prefer the vendor
// script for a graceful shutdown sequence (flush telemetry, stop services per
// config) and fall back to a raw systemd reboot if unavailable. This is used
// as an optional last‑resort healing step when repeated service restarts have
// not cleared a persistent OCPP/pilot mismatch. Use with care.
func rebootSystem() error {
	// Preferred: Wallbox official reboot wrapper.
	scriptCmd := exec.Command("/home/root/.wallbox/reboot.sh")
	if err := scriptCmd.Run(); err == nil {
		return nil
	}

	// Fallback: direct systemd reboot.
	cmd := exec.Command("systemctl", "reboot")
	return cmd.Run()
}

func bridgeVersion() string {
	if buildVersion != "" && buildVersion != "dev" {
		return buildVersion
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if ok && buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		return buildInfo.Main.Version
	}
	return "dev"
}
