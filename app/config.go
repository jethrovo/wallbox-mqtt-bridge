package bridge

import (
	"gopkg.in/ini.v1"
)

type WallboxConfig struct {
	MQTT struct {
		Host     string `ini:"host"`
		Port     int    `ini:"port"`
		Username string `ini:"username"`
		Password string `ini:"password"`
	} `ini:"mqtt"`

	Settings struct {
		PollingIntervalSeconds int    `ini:"polling_interval_seconds"`
		DeviceName             string `ini:"device_name"`
		DebugSensors           bool   `ini:"debug_sensors"`
		PowerBoostEnabled      bool   `ini:"power_boost_enabled"`
		AutoRestartOCPP        bool   `ini:"auto_restart_ocpp"`
		OCPPMismatchSeconds    int    `ini:"ocpp_mismatch_seconds"`
		OCPPRestartCooldown    int    `ini:"ocpp_restart_cooldown_seconds"`
		OCPPMaxRestarts        int    `ini:"ocpp_max_restarts"`
		OCPPFullReboot         bool   `ini:"ocpp_full_reboot"`
		PilotErrorReboot       bool   `ini:"pilot_error_reboot"`
		PilotErrorSeconds      int    `ini:"pilot_error_seconds"`
	} `ini:"settings"`
}

func (w *WallboxConfig) SaveTo(path string) {
	cfg := ini.Empty()
	cfg.ReflectFrom(w)
	cfg.SaveTo(path)
}

func LoadConfig(path string) *WallboxConfig {
	cfg, _ := ini.Load(path)

	var config WallboxConfig
	if err := cfg.MapTo(&config); err != nil {
		return nil
	}

	return &config
}
