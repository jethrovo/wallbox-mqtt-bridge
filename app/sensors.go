package bridge

import (
	"fmt"
	"strconv"

	"wallbox-mqtt-bridge/app/ratelimit"
	"wallbox-mqtt-bridge/app/wallbox"
)

type Entity struct {
	Component string
	Getter    func() string
	Setter    func(string)
	RateLimit *ratelimit.DeltaRateLimit
	Config    map[string]string
}

func strToInt(val string) int {
	i, _ := strconv.Atoi(val)
	return i
}

func strToFloat(val string) float64 {
	f, _ := strconv.ParseFloat(val, 64)
	return f
}

func getEntities(w *wallbox.Wallbox) map[string]Entity {
	return map[string]Entity{
		"added_energy": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisState.ScheduleEnergy) },
			RateLimit: ratelimit.NewDeltaRateLimit(10, 50),
			Config: map[string]string{
				"name":                        "Added energy",
				"device_class":                "energy",
				"unit_of_measurement":         "Wh",
				"state_class":                 "total",
				"suggested_display_precision": "1",
			},
		},
		"added_range": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.SQL.AddedRange) },
			Config: map[string]string{
				"name":                        "Added range",
				"device_class":                "distance",
				"unit_of_measurement":         "km",
				"state_class":                 "total",
				"suggested_display_precision": "1",
				"icon":                        "mdi:map-marker-distance",
			},
		},
		"cable_connected": {
			Component: "binary_sensor",
			Getter:    func() string { return fmt.Sprint(w.CableConnected()) },
			Config: map[string]string{
				"name":         "Cable connected",
				"payload_on":   "1",
				"payload_off":  "0",
				"icon":         "mdi:ev-plug-type1",
				"device_class": "plug",
			},
		},
		"charging_enable": {
			Component: "switch",
			Setter:    func(val string) { w.SetChargingEnable(strToInt(val)) },
			Getter:    func() string { return fmt.Sprint(w.Data.SQL.ChargingEnable) },
			Config: map[string]string{
				"name":        "Charging enable",
				"payload_on":  "1",
				"payload_off": "0",
				"icon":        "mdi:ev-station",
			},
		},
		"charging_power": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.Line1Power + w.Data.RedisM2W.Line2Power + w.Data.RedisM2W.Line3Power)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 100),
			Config: map[string]string{
				"name":                        "Charging power",
				"device_class":                "power",
				"unit_of_measurement":         "W",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"charging_power_l1": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.Line1Power)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 100),
			Config: map[string]string{
				"name":                        "Charging power L1",
				"device_class":                "power",
				"unit_of_measurement":         "W",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"charging_power_l2": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.Line2Power)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 100),
			Config: map[string]string{
				"name":                        "Charging power L2",
				"device_class":                "power",
				"unit_of_measurement":         "W",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"charging_power_l3": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.Line3Power)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 100),
			Config: map[string]string{
				"name":                        "Charging power L3",
				"device_class":                "power",
				"unit_of_measurement":         "W",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"charging_current_l1": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.Line1Current)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 0.2),
			Config: map[string]string{
				"name":                        "Charging current L1",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"charging_current_l2": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.Line2Current)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 0.2),
			Config: map[string]string{
				"name":                        "Charging current L2",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"charging_current_l3": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.Line3Current)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 0.2),
			Config: map[string]string{
				"name":                        "Charging current L3",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"cumulative_added_energy": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.SQL.CumulativeAddedEnergy) },
			Config: map[string]string{
				"name":                        "Cumulative added energy",
				"device_class":                "energy",
				"unit_of_measurement":         "Wh",
				"state_class":                 "total_increasing",
				"suggested_display_precision": "1",
			},
		},
		"halo_brightness": {
			Component: "number",
			Setter:    func(val string) { w.SetHaloBrightness(strToInt(val)) },
			Getter:    func() string { return fmt.Sprint(w.Data.SQL.HaloBrightness) },
			Config: map[string]string{
				"name":                "Halo Brightness",
				"command_topic":       "~/set",
				"min":                 "0",
				"max":                 "100",
				"icon":                "mdi:brightness-percent",
				"unit_of_measurement": "%",
				"entity_category":     "config",
			},
		},
		"lock": {
			Component: "lock",
			Setter:    func(val string) { w.SetLocked(strToInt(val)) },
			Getter:    func() string { return fmt.Sprint(w.Data.SQL.Lock) },
			Config: map[string]string{
				"name":           "Lock",
				"payload_lock":   "1",
				"payload_unlock": "0",
				"state_locked":   "1",
				"state_unlocked": "0",
				"command_topic":  "~/set",
			},
		},
		"max_charging_current": {
			Component: "number",
			Setter:    func(val string) { w.SetMaxChargingCurrent(strToInt(val)) },
			Getter:    func() string { return fmt.Sprint(w.Data.SQL.MaxChargingCurrent) },
			Config: map[string]string{
				"name":                "Max charging current",
				"command_topic":       "~/set",
				"min":                 "6",
				"max":                 fmt.Sprint(w.AvailableCurrent()),
				"unit_of_measurement": "A",
				"device_class":        "current",
			},
		},
		"status": {
			Component: "sensor",
			Getter:    w.EffectiveStatus,
			Config: map[string]string{
				"name": "Status",
			},
		},
		"temp_l1": {
		    Component: "sensor",
		    Getter:    func() string { return fmt.Sprint(w.Data.RedisM2W.TempL1) },
		    Config: map[string]string{
			"name":                 "Temperature Line 1",
			"unit_of_measurement":  "°C",
			"device_class":         "temperature",
			"state_class":          "measurement",
			"suggested_display_precision": "1",
		    },
		},
		"temp_l2": {
		    Component: "sensor",
		    Getter:    func() string { return fmt.Sprint(w.Data.RedisM2W.TempL2) },
		    Config: map[string]string{
			"name":                 "Temperature Line 2",
			"unit_of_measurement":  "°C",
			"device_class":         "temperature",
			"state_class":          "measurement",
			"suggested_display_precision": "1",
		    },
		},
		"temp_l3": {
		    Component: "sensor",
		    Getter:    func() string { return fmt.Sprint(w.Data.RedisM2W.TempL3) },
		    Config: map[string]string{
			"name":                 "Temperature Line 3",
			"unit_of_measurement":  "°C",
			"device_class":         "temperature",
			"state_class":          "measurement",
			"suggested_display_precision": "1",
		    },
		},		
	}
}

func getPowerBoostEntities(w *wallbox.Wallbox, c *WallboxConfig) map[string]Entity {
	return map[string]Entity{
		"power_boost_power_l1": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.PowerBoostLine1Power)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 100),
			Config: map[string]string{
				"name":                        "Power Boost L1",
				"device_class":                "power",
				"unit_of_measurement":         "W",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"power_boost_power_l2": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.PowerBoostLine2Power)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 100),
			Config: map[string]string{
				"name":                        "Power Boost L2",
				"device_class":                "power",
				"unit_of_measurement":         "W",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"power_boost_power_l3": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.PowerBoostLine3Power)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 100),
			Config: map[string]string{
				"name":                        "Power Boost L3",
				"device_class":                "power",
				"unit_of_measurement":         "W",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"power_boost_current_l1": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.PowerBoostLine1Current)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 0.2),
			Config: map[string]string{
				"name":                        "Power Boost current L1",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"power_boost_current_l2": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.PowerBoostLine2Current)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 0.2),
			Config: map[string]string{
				"name":                        "Power Boost current L2",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"power_boost_current_l3": {
			Component: "sensor",
			Getter: func() string {
				return fmt.Sprint(w.Data.RedisM2W.PowerBoostLine3Current)
			},
			RateLimit: ratelimit.NewDeltaRateLimit(10, 0.2),
			Config: map[string]string{
				"name":                        "Power Boost current L3",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"power_boost_cumulative_added_energy": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisM2W.PowerBoostCumulativeEnergy) },
			Config: map[string]string{
				"name":                        "Power Boost Cumulative added energy",
				"device_class":                "energy",
				"unit_of_measurement":         "Wh",
				"state_class":                 "total_increasing",
				"suggested_display_precision": "1",
			},
		},
	}
}

func getDebugEntities(w *wallbox.Wallbox) map[string]Entity {
	return map[string]Entity{
		"control_pilot": {
			Component: "sensor",
			Getter:    w.ControlPilotStatus,
			Config: map[string]string{
				"name": "Control pilot",
			},
		},
		"m2w_status": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisM2W.ChargerStatus) },
			Config: map[string]string{
				"name": "M2W Status",
			},
		},
		"state_machine_state": {
			Component: "sensor",
			Getter:    w.StateMachineState,
			Config: map[string]string{
				"name": "State machine",
			},
		},
		"s2_open": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisState.S2open) },
			Config: map[string]string{
				"name": "S2 open",
			},
		},
	}
}

// getTelemetryEventEntities creates entities for sensor data from the telemetry events
func getTelemetryEventEntities(w *wallbox.Wallbox) map[string]Entity {
	entities := map[string]Entity{
		// Power and Current Related
		"icp_max_current": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.ICPMaxCurrent) },
			Config: map[string]string{
				"name":                        "ICP Max Current",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"internal_meter_current_l1": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.InternalMeterCurrentL1) },
			RateLimit: ratelimit.NewDeltaRateLimit(10, 0.2),
			Config: map[string]string{
				"name":                        "Internal Meter Current L1",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"internal_meter_current_l2": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.InternalMeterCurrentL2) },
			RateLimit: ratelimit.NewDeltaRateLimit(10, 0.2),
			Config: map[string]string{
				"name":                        "Internal Meter Current L2",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"internal_meter_current_l3": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.InternalMeterCurrentL3) },
			RateLimit: ratelimit.NewDeltaRateLimit(10, 0.2),
			Config: map[string]string{
				"name":                        "Internal Meter Current L3",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"icon":                        "mdi:leaf",
				"entity_category":             "diagnostic",
			},
		},
		"user_current_proposal": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.UserCurrentProposal) },
			Config: map[string]string{
				"name":                        "User Current Proposal",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		
		// Voltage Related
		"internal_meter_voltage_l1": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.InternalMeterVoltageL1) },
			RateLimit: ratelimit.NewDeltaRateLimit(10, 2),
			Config: map[string]string{
				"name":                        "Internal Meter Voltage L1",
				"device_class":                "voltage",
				"unit_of_measurement":         "V",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"internal_meter_voltage_l2": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.InternalMeterVoltageL2) },
			RateLimit: ratelimit.NewDeltaRateLimit(10, 2),
			Config: map[string]string{
				"name":                        "Internal Meter Voltage L2",
				"device_class":                "voltage",
				"unit_of_measurement":         "V",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"internal_meter_voltage_l3": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.InternalMeterVoltageL3) },
			RateLimit: ratelimit.NewDeltaRateLimit(10, 2),
			Config: map[string]string{
				"name":                        "Internal Meter Voltage L3",
				"device_class":                "voltage",
				"unit_of_measurement":         "V",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"control_pilot_high_voltage": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.ControlPilotHighVolts / 10.0) }, // Convert tenths to volts
			Config: map[string]string{
				"name":                        "Control Pilot High Voltage",
				"device_class":                "voltage",
				"unit_of_measurement":         "V",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"control_pilot_low_voltage": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.ControlPilotLowVolts / 10.0) }, // Convert tenths to volts
			Config: map[string]string{
				"name":                        "Control Pilot Low Voltage",
				"device_class":                "voltage",
				"unit_of_measurement":         "V",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		
		// Energy Related
		"internal_meter_energy": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.InternalMeterEnergy) },
			Config: map[string]string{
				"name":                        "Internal Meter Energy",
				"device_class":                "energy",
				"unit_of_measurement":         "Wh",
				"state_class":                 "total_increasing",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"ecosmart_green_energy": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.EcosmartGreenEnergy) },
			Config: map[string]string{
				"name":                        "EcoSmart Green Energy",
				"device_class":                "energy",
				"unit_of_measurement":         "Wh",
				"state_class":                 "total_increasing",
				"suggested_display_precision": "1",
				"icon":                        "mdi:leaf",
				"entity_category":             "diagnostic",
			},
		},
		"ecosmart_energy_total": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.EcosmartEnergyTotal) },
			Config: map[string]string{
				"name":                        "EcoSmart Total Energy",
				"device_class":                "energy",
				"unit_of_measurement":         "Wh",
				"state_class":                 "total_increasing",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		
		// System Status
		"ecosmart_mode": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.EcosmartMode) },
			Config: map[string]string{
				"name":            "EcoSmart Mode",
				"icon":            "mdi:leaf",
				"entity_category": "diagnostic",
			},
		},
		"ecosmart_status": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.EcosmartStatus) },
			Config: map[string]string{
				"name":            "EcoSmart Status",
				"icon":            "mdi:leaf",
				"entity_category": "diagnostic",
			},
		},
		"ecosmart_current_proposal": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.EcosmartCurrentProposal) },
			Config: map[string]string{
				"name":                        "EcoSmart Current Proposal",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"icon":                        "mdi:leaf",
				"entity_category":             "diagnostic",
			},
		},
		
		// Frequency
		"internal_meter_frequency": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.InternalMeterFrequency) },
			Config: map[string]string{
				"name":                        "Internal Meter Frequency",
				"device_class":                "frequency",
				"unit_of_measurement":         "Hz",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		
		// Schedule and PowerBoost
		"schedule_status": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.ScheduleStatus) },
			Config: map[string]string{
				"name":            "Schedule Status",
				"icon":            "mdi:calendar-clock",
				"entity_category": "diagnostic",
			},
		},
		"schedule_current_proposal": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.ScheduleCurrentProposal) },
			Config: map[string]string{
				"name":                        "Schedule Current Proposal",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"icon":                        "mdi:calendar-clock",
				"entity_category":             "diagnostic",
			},
		},
		"powerboost_status": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.PowerboostStatus) },
			Config: map[string]string{
				"name":            "PowerBoost Status",
				"entity_category": "diagnostic",
			},
		},
		"powerboost_proposal_current": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.PowerboostProposalCurrent) },
			Config: map[string]string{
				"name":                        "PowerBoost Current Proposal",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		
		// New sensors from JSON data
		"charging_enable_sensor": {
			Component: "binary_sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.ChargingEnable) },
			Config: map[string]string{
				"name":            "Charging Enable Status",
				"payload_on":      "1",
				"payload_off":     "0",
				"entity_category": "diagnostic",
			},
		},
		"control_pilot_duty": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.ControlPilotDuty) },
			Config: map[string]string{
				"name":            "Control Pilot Duty",
				"state_class":     "measurement",
				"entity_category": "diagnostic",
			},
		},
		"control_pilot_high_voltage_tenths": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.ControlPilotHighVolts) }, // Raw tenths of volts
			Config: map[string]string{
				"name":                        "Control Pilot High Voltage (tenths)",
				"entity_category":             "diagnostic",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"control_pilot_low_voltage_tenths": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.ControlPilotLowVolts) }, // Raw tenths of volts
			Config: map[string]string{
				"name":                        "Control Pilot Low Voltage (tenths)",
				"entity_category":             "diagnostic",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
			},
		},
		"control_pilot_status_raw": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.ControlPilotStatus) },
			Config: map[string]string{
				"name":            "Control Pilot Status Raw",
				"entity_category": "diagnostic",
			},
		},
		"max_available_current": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.MaxAvailableCurrent) },
			Config: map[string]string{
				"name":                        "Max Available Current",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"max_charging_current_sensor": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.MaxChargingCurrent) },
			Config: map[string]string{
				"name":                        "Max Charging Current (sensor)",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"mid_status": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.MidStatus) },
			Config: map[string]string{
				"name":            "MID Status",
				"entity_category": "diagnostic",
			},
		},
		"power_sharing_status": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.PowerSharingStatus) },
			Config: map[string]string{
				"name":            "Power Sharing Status",
				"entity_category": "diagnostic",
			},
		},
		"temperature_l1_sensor": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.TempL1) },
			Config: map[string]string{
				"name":                        "Temperature Line 1 (sensor)",
				"device_class":                "temperature",
				"unit_of_measurement":         "°C",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"temperature_l2_sensor": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.TempL2) },
			Config: map[string]string{
				"name":                        "Temperature Line 2 (sensor)",
				"device_class":                "temperature",
				"unit_of_measurement":         "°C",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"temperature_l3_sensor": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.TempL3) },
			Config: map[string]string{
				"name":                        "Temperature Line 3 (sensor)",
				"device_class":                "temperature",
				"unit_of_measurement":         "°C",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"welding": {
			Component: "binary_sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.Welding) },
			Config: map[string]string{
				"name":            "Welding Detection",
				"device_class":    "problem",
				"payload_on":      "1",
				"payload_off":     "0",
				"entity_category": "diagnostic",
			},
		},
		"firmware_error": {
			Component: "binary_sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.FirmwareError) },
			Config: map[string]string{
				"name":            "Firmware Error",
				"device_class":    "problem",
				"payload_on":      "1",
				"payload_off":     "0",
				"entity_category": "diagnostic",
			},
		},
		"dynamic_power_sharing_max_current": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.DynamicPowerSharingMaxCurrent) },
			Config: map[string]string{
				"name":                        "Dynamic Power Sharing Max Current",
				"device_class":                "current",
				"unit_of_measurement":         "A",
				"state_class":                 "measurement",
				"suggested_display_precision": "1",
				"entity_category":             "diagnostic",
			},
		},
		"power_relay_management_command": {
			Component: "sensor",
			Getter:    func() string { return fmt.Sprint(w.Data.RedisTelemetry.PowerRelayManagementCommand) },
			Config: map[string]string{
				"name":            "Power Relay Management Command",
				"entity_category": "diagnostic",
			},
		},
	}
	
	return entities
}