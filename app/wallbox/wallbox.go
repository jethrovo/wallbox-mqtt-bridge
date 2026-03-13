package wallbox

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type DataCache struct {
	SQL struct {
		Lock                     int     `db:"lock"`
		ChargingEnable           int     `db:"charging_enable"`
		MaxChargingCurrent       int     `db:"max_charging_current"`
		HaloBrightness           int     `db:"halo_brightness"`
		CumulativeAddedEnergy    float64 `db:"cumulative_added_energy"`
		AddedRange               float64 `db:"added_range"`
		ActiveSessionEnergyTotal float64 `db:"active_session_energy_total"`
	}

	RedisState struct {
		SessionState   int     `redis:"session.state"`
		ControlPilot   int     `redis:"ctrlPilot"`
		S2open         int     `redis:"S2open"`
		ScheduleEnergy float64 `redis:"scheduleEnergy"`
	}

	RedisM2W struct {
		ChargerStatus              int     `redis:"tms.charger_status"`
		Line1Power                 float64 `redis:"tms.line1.power_watt.value"`
		Line2Power                 float64 `redis:"tms.line2.power_watt.value"`
		Line3Power                 float64 `redis:"tms.line3.power_watt.value"`
		Line1Current               float64 `redis:"tms.line1.current_amp.value"`
		Line2Current               float64 `redis:"tms.line2.current_amp.value"`
		Line3Current               float64 `redis:"tms.line3.current_amp.value"`
		PowerBoostLine1Power       float64 `redis:"PBO.line1.power.value"`
		PowerBoostLine2Power       float64 `redis:"PBO.line2.power.value"`
		PowerBoostLine3Power       float64 `redis:"PBO.line3.power.value"`
		PowerBoostLine1Current     float64 `redis:"PBO.line1.current.value"`
		PowerBoostLine2Current     float64 `redis:"PBO.line2.current.value"`
		PowerBoostLine3Current     float64 `redis:"PBO.line3.current.value"`
		PowerBoostCumulativeEnergy float64 `redis:"PBO.energy_wh.value"`
		TempL1                     float64 `redis:"tms.line1.temp_deg.value"`
		TempL2                     float64 `redis:"tms.line2.temp_deg.value"`
		TempL3                     float64 `redis:"tms.line3.temp_deg.value"`
	}

	RedisTelemetry struct {
		ICPMaxCurrent                 float64 `redis:"telemetry.SENSOR_ICP_MAX_CURRENT"`
		InternalMeterCurrentL1        float64 `redis:"telemetry.SENSOR_INTERNAL_METER_CURRENT_L1"`
		InternalMeterCurrentL2        float64 `redis:"telemetry.SENSOR_INTERNAL_METER_CURRENT_L2"`
		InternalMeterCurrentL3        float64 `redis:"telemetry.SENSOR_INTERNAL_METER_CURRENT_L3"`
		MaxAvailableCurrent           float64 `redis:"telemetry.SENSOR_MAX_AVAILABLE_CURRENT"`
		UserCurrentProposal           float64 `redis:"telemetry.SENSOR_USER_CURRENT_PROPOSAL"`
		DynamicPowerSharingMaxCurrent float64 `redis:"telemetry.SENSOR_DYNAMIC_POWER_SHARING_MAX_CURRENT"`

		InternalMeterVoltageL1           float64 `redis:"telemetry.SENSOR_INTERNAL_METER_VOLTAGE_L1"`
		InternalMeterVoltageL2           float64 `redis:"telemetry.SENSOR_INTERNAL_METER_VOLTAGE_L2"`
		InternalMeterVoltageL3           float64 `redis:"telemetry.SENSOR_INTERNAL_METER_VOLTAGE_L3"`
		InternalMeterVoltageFilterStatus float64 `redis:"telemetry.SENSOR_INTERNAL_METER_VOLTAGE_FILTER_STATUS"`
		ControlPilotHighVolts            float64 `redis:"telemetry.SENSOR_CONTROL_PILOT_HIGH_TENTHS_OF_VOLTS"`
		ControlPilotLowVolts             float64 `redis:"telemetry.SENSOR_CONTROL_PILOT_LOW_TENTHS_OF_VOLTS"`

		InternalMeterEnergy float64 `redis:"telemetry.SENSOR_INTERNAL_METER_ENERGY"`
		EcosmartGreenEnergy float64 `redis:"telemetry.SENSOR_ECOSMART_GREEN_ENERGY"`
		EcosmartEnergyTotal float64 `redis:"telemetry.SENSOR_ECOSMART_ENERGY_TOTAL"`

		EcosmartMode            float64 `redis:"telemetry.SENSOR_ECOSMART_MODE"`
		EcosmartStatus          float64 `redis:"telemetry.SENSOR_ECOSMART_STATUS"`
		EcosmartCurrentProposal float64 `redis:"telemetry.SENSOR_ECOSMART_CURRENT_PROPOSAL"`

		InternalMeterFrequency float64 `redis:"telemetry.SENSOR_INTERNAL_METER_FREQUENCY"`

		ScheduleStatus            float64 `redis:"telemetry.SENSOR_SCHEDULE_STATUS"`
		ScheduleCurrentProposal   float64 `redis:"telemetry.SENSOR_SCHEDULE_CURRENT_PROPOSAL"`
		PowerboostStatus          float64 `redis:"telemetry.SENSOR_DCA_POWERBOOST_STATUS"`
		PowerboostProposalCurrent float64 `redis:"telemetry.SENSOR_POWERBOOST_PROPOSAL_CURRENT"`

		// Additional fields referenced in getTelemetryEventEntities
		ChargingEnable              float64 `redis:"telemetry.SENSOR_CHARGING_ENABLE"`
		ControlPilotDuty            float64 `redis:"telemetry.SENSOR_CONTROL_PILOT_DUTY"`
		ControlPilotStatus          float64 `redis:"telemetry.SENSOR_CONTROL_PILOT_STATUS"`
		MaxChargingCurrent          float64 `redis:"telemetry.SENSOR_MAX_CHARGING_CURRENT"`
		MidStatus                   float64 `redis:"telemetry.SENSOR_MID_STATUS"`
		PowerSharingStatus          float64 `redis:"telemetry.SENSOR_POWER_SHARING_STATUS"`
		TempL1                      float64 `redis:"telemetry.SENSOR_TEMP_L1"`
		TempL2                      float64 `redis:"telemetry.SENSOR_TEMP_L2"`
		TempL3                      float64 `redis:"telemetry.SENSOR_TEMP_L3"`
		Welding                     float64 `redis:"telemetry.SENSOR_WELDING"`
		FirmwareError               float64 `redis:"telemetry.SENSOR_FIRMWARE_ERROR"`
		PowerRelayManagementCommand float64 `redis:"telemetry.SENSOR_POWER_RELAY_MANAGEMENT_COMMAND"`
		StateMachine                float64 `redis:"telemetry.SENSOR_STATE_MACHINE"`
		OCPPStatus                  float64 `redis:"telemetry.SENSOR_OCPP_STATUS"`

		// Quiet unmapped telemetry to avoid log spam
		ControlMode             float64 `redis:"telemetry.SENSOR_CONTROL_MODE"`
		DCA_VoltageL1           float64 `redis:"telemetry.SENSOR_DCA_VOLTAGE_L1"`
		DCA_VoltageL2           float64 `redis:"telemetry.SENSOR_DCA_VOLTAGE_L2"`
		DCA_VoltageL3           float64 `redis:"telemetry.SENSOR_DCA_VOLTAGE_L3"`
		DCA_CurrentL1           float64 `redis:"telemetry.SENSOR_DCA_CURRENT_L1"`
		DCA_CurrentL2           float64 `redis:"telemetry.SENSOR_DCA_CURRENT_L2"`
		DCA_CurrentL3           float64 `redis:"telemetry.SENSOR_DCA_CURRENT_L3"`
		DCAMeterFrequency       float64 `redis:"telemetry.SENSOR_DCA_METER_FREQUENCY"`
		ExternalMeterStatus     float64 `redis:"telemetry.SENSOR_EXTERNAL_METER_STATUS"`
		PMSDominantFeature      float64 `redis:"telemetry.SENSOR_PMS_DOMINANT_FEATURE"`
		PMSMetadata             float64 `redis:"telemetry.SENSOR_PMS_METADATA"`
		PMSPhaseSwitch          float64 `redis:"telemetry.SENSOR_PMS_PHASE_SWITCH"`
		GSMRecoTrigger          float64 `redis:"telemetry.SENSOR_GSM_RECO_TRIGGER"`
		ConnectivityStatus      float64 `redis:"telemetry.SENSOR_CONNECTIVITY_STATUS"`
		OnTime                  float64 `redis:"telemetry.SENSOR_ON_TIME"`
		WifiSignalStrength      float64 `redis:"telemetry.SENSOR_WIFI_SIGNAL_STRENGTH"`
		ConnectionType          float64 `redis:"telemetry.SENSOR_CONNECTION_TYPE"`
		GetChargerConfigSend    float64 `redis:"telemetry.SENSOR_GET_CHARGER_CONFIG_SEND"`
		GetChargerConfigReceive float64 `redis:"telemetry.SENSOR_GET_CHARGER_CONFIG_RECEIVE"`
		GetChargerConfigCalls   float64 `redis:"telemetry.SENSOR_GET_CHARGER_CONFIG_CALLS"`

		// Service resource telemetry
		NetworkManagerCPUUsage    float64 `redis:"telemetry.SENSOR_NETWORKMANAGER_CPU_USAGE"`
		NetworkManagerThreads     float64 `redis:"telemetry.SENSOR_NETWORKMANAGER_THREADS"`
		NetworkManagerMemory      float64 `redis:"telemetry.SENSOR_NETWORKMANAGER_MEMORY"`
		NetworkManagerSimpleState float64 `redis:"telemetry.SENSOR_NETWORKMANAGER_SIMPLE_STATE"`

		BLEWallboxCPUUsage    float64 `redis:"telemetry.SENSOR_BLEWALLBOX_CPU_USAGE"`
		BLEWallboxThreads     float64 `redis:"telemetry.SENSOR_BLEWALLBOX_THREADS"`
		BLEWallboxMemory      float64 `redis:"telemetry.SENSOR_BLEWALLBOX_MEMORY"`
		BLEWallboxSimpleState float64 `redis:"telemetry.SENSOR_BLEWALLBOX_SIMPLE_STATE"`

		BluetoothGatewayCPUUsage    float64 `redis:"telemetry.SENSOR_BLUETOOTH_GATEWAY_CPU_USAGE"`
		BluetoothGatewayThreads     float64 `redis:"telemetry.SENSOR_BLUETOOTH_GATEWAY_THREADS"`
		BluetoothGatewayMemory      float64 `redis:"telemetry.SENSOR_BLUETOOTH_GATEWAY_MEMORY"`
		BluetoothGatewaySimpleState float64 `redis:"telemetry.SENSOR_BLUETOOTH_GATEWAY_SIMPLE_STATE"`

		CloudPubSubCommandCPUUsage    float64 `redis:"telemetry.SENSOR_CLOUD_PUB_SUB_COMMAND_CPU_USAGE"`
		CloudPubSubCommandThreads     float64 `redis:"telemetry.SENSOR_CLOUD_PUB_SUB_COMMAND_THREADS"`
		CloudPubSubCommandMemory      float64 `redis:"telemetry.SENSOR_CLOUD_PUB_SUB_COMMAND_MEMORY"`
		CloudPubSubCommandSimpleState float64 `redis:"telemetry.SENSOR_CLOUD_PUB_SUB_COMMAND_SIMPLE_STATE"`

		CloudPubSubTelemetryCPUUsage    float64 `redis:"telemetry.SENSOR_CLOUD_PUB_SUB_TELEMETRY_CPU_USAGE"`
		CloudPubSubTelemetryThreads     float64 `redis:"telemetry.SENSOR_CLOUD_PUB_SUB_TELEMETRY_THREADS"`
		CloudPubSubTelemetryMemory      float64 `redis:"telemetry.SENSOR_CLOUD_PUB_SUB_TELEMETRY_MEMORY"`
		CloudPubSubTelemetrySimpleState float64 `redis:"telemetry.SENSOR_CLOUD_PUB_SUB_TELEMETRY_SIMPLE_STATE"`

		CredentialsGeneratorCPUUsage    float64 `redis:"telemetry.SENSOR_CREDENTIALS_GENERATOR_CPU_USAGE"`
		CredentialsGeneratorThreads     float64 `redis:"telemetry.SENSOR_CREDENTIALS_GENERATOR_THREADS"`
		CredentialsGeneratorMemory      float64 `redis:"telemetry.SENSOR_CREDENTIALS_GENERATOR_MEMORY"`
		CredentialsGeneratorSimpleState float64 `redis:"telemetry.SENSOR_CREDENTIALS_GENERATOR_SIMPLE_STATE"`

		DBUSCPUUsage    float64 `redis:"telemetry.SENSOR_DBUS_CPU_USAGE"`
		DBUSThreads     float64 `redis:"telemetry.SENSOR_DBUS_THREADS"`
		DBUSMemory      float64 `redis:"telemetry.SENSOR_DBUS_MEMORY"`
		DBUSSimpleState float64 `redis:"telemetry.SENSOR_DBUS_SIMPLE_STATE"`

		Micro2WallboxCPUUsage    float64 `redis:"telemetry.SENSOR_MICRO2WALLBOX_CPU_USAGE"`
		Micro2WallboxThreads     float64 `redis:"telemetry.SENSOR_MICRO2WALLBOX_THREADS"`
		Micro2WallboxMemory      float64 `redis:"telemetry.SENSOR_MICRO2WALLBOX_MEMORY"`
		Micro2WallboxSimpleState float64 `redis:"telemetry.SENSOR_MICRO2WALLBOX_SIMPLE_STATE"`

		MySQLDCPUUsage    float64 `redis:"telemetry.SENSOR_MYSQLD_CPU_USAGE"`
		MySQLDThreads     float64 `redis:"telemetry.SENSOR_MYSQLD_THREADS"`
		MySQLDMemory      float64 `redis:"telemetry.SENSOR_MYSQLD_MEMORY"`
		MySQLDSimpleState float64 `redis:"telemetry.SENSOR_MYSQLD_SIMPLE_STATE"`

		MyWallboxCPUUsage    float64 `redis:"telemetry.SENSOR_MYWALLBOX_CPU_USAGE"`
		MyWallboxThreads     float64 `redis:"telemetry.SENSOR_MYWALLBOX_THREADS"`
		MyWallboxMemory      float64 `redis:"telemetry.SENSOR_MYWALLBOX_MEMORY"`
		MyWallboxSimpleState float64 `redis:"telemetry.SENSOR_MYWALLBOX_SIMPLE_STATE"`

		OCPPWallboxCPUUsage    float64 `redis:"telemetry.SENSOR_OCPPWALLBOX_CPU_USAGE"`
		OCPPWallboxThreads     float64 `redis:"telemetry.SENSOR_OCPPWALLBOX_THREADS"`
		OCPPWallboxMemory      float64 `redis:"telemetry.SENSOR_OCPPWALLBOX_MEMORY"`
		OCPPWallboxSimpleState float64 `redis:"telemetry.SENSOR_OCPPWALLBOX_SIMPLE_STATE"`

		OnTimeTrackCPUUsage    float64 `redis:"telemetry.SENSOR_ON_TIME_TRACK_CPU_USAGE"`
		OnTimeTrackThreads     float64 `redis:"telemetry.SENSOR_ON_TIME_TRACK_THREADS"`
		OnTimeTrackMemory      float64 `redis:"telemetry.SENSOR_ON_TIME_TRACK_MEMORY"`
		OnTimeTrackSimpleState float64 `redis:"telemetry.SENSOR_ON_TIME_TRACK_SIMPLE_STATE"`

		PowerManagerCPUUsage    float64 `redis:"telemetry.SENSOR_POWER_MANAGER_CPU_USAGE"`
		PowerManagerThreads     float64 `redis:"telemetry.SENSOR_POWER_MANAGER_THREADS"`
		PowerManagerMemory      float64 `redis:"telemetry.SENSOR_POWER_MANAGER_MEMORY"`
		PowerManagerSimpleState float64 `redis:"telemetry.SENSOR_POWER_MANAGER_SIMPLE_STATE"`

		RedisCPUUsage    float64 `redis:"telemetry.SENSOR_REDIS_CPU_USAGE"`
		RedisThreads     float64 `redis:"telemetry.SENSOR_REDIS_THREADS"`
		RedisMemory      float64 `redis:"telemetry.SENSOR_REDIS_MEMORY"`
		RedisSimpleState float64 `redis:"telemetry.SENSOR_REDIS_SIMPLE_STATE"`

		ResourcesMonitorCPUUsage    float64 `redis:"telemetry.SENSOR_RESOURCES_MONITOR_CPU_USAGE"`
		ResourcesMonitorThreads     float64 `redis:"telemetry.SENSOR_RESOURCES_MONITOR_THREADS"`
		ResourcesMonitorMemory      float64 `redis:"telemetry.SENSOR_RESOURCES_MONITOR_MEMORY"`
		ResourcesMonitorSimpleState float64 `redis:"telemetry.SENSOR_RESOURCES_MONITOR_SIMPLE_STATE"`

		ScheduleManagerCPUUsage    float64 `redis:"telemetry.SENSOR_SCHEDULE_MANAGER_CPU_USAGE"`
		ScheduleManagerThreads     float64 `redis:"telemetry.SENSOR_SCHEDULE_MANAGER_THREADS"`
		ScheduleManagerMemory      float64 `redis:"telemetry.SENSOR_SCHEDULE_MANAGER_MEMORY"`
		ScheduleManagerSimpleState float64 `redis:"telemetry.SENSOR_SCHEDULE_MANAGER_SIMPLE_STATE"`

		SoftwareUpdateCPUUsage    float64 `redis:"telemetry.SENSOR_SOFTWARE_UPDATE_CPU_USAGE"`
		SoftwareUpdateThreads     float64 `redis:"telemetry.SENSOR_SOFTWARE_UPDATE_THREADS"`
		SoftwareUpdateMemory      float64 `redis:"telemetry.SENSOR_SOFTWARE_UPDATE_MEMORY"`
		SoftwareUpdateSimpleState float64 `redis:"telemetry.SENSOR_SOFTWARE_UPDATE_SIMPLE_STATE"`

		SystemSupervisorCPUUsage    float64 `redis:"telemetry.SENSOR_SYSTEM_SUPERVISOR_CPU_USAGE"`
		SystemSupervisorThreads     float64 `redis:"telemetry.SENSOR_SYSTEM_SUPERVISOR_THREADS"`
		SystemSupervisorMemory      float64 `redis:"telemetry.SENSOR_SYSTEM_SUPERVISOR_MEMORY"`
		SystemSupervisorSimpleState float64 `redis:"telemetry.SENSOR_SYSTEM_SUPERVISOR_SIMPLE_STATE"`

		TelemetryServiceCPUUsage    float64 `redis:"telemetry.SENSOR_TELEMETRY_SRVC_CPU_USAGE"`
		TelemetryServiceThreads     float64 `redis:"telemetry.SENSOR_TELEMETRY_SRVC_THREADS"`
		TelemetryServiceMemory      float64 `redis:"telemetry.SENSOR_TELEMETRY_SRVC_MEMORY"`
		TelemetryServiceSimpleState float64 `redis:"telemetry.SENSOR_TELEMETRY_SRVC_SIMPLE_STATE"`

		WallboxCBITCPUUsage    float64 `redis:"telemetry.SENSOR_WALLBOX_CBIT_CPU_USAGE"`
		WallboxCBITThreads     float64 `redis:"telemetry.SENSOR_WALLBOX_CBIT_THREADS"`
		WallboxCBITMemory      float64 `redis:"telemetry.SENSOR_WALLBOX_CBIT_MEMORY"`
		WallboxCBITSimpleState float64 `redis:"telemetry.SENSOR_WALLBOX_CBIT_SIMPLE_STATE"`

		WallboxLoginCPUUsage    float64 `redis:"telemetry.SENSOR_WALLBOX_LOGIN_CPU_USAGE"`
		WallboxLoginThreads     float64 `redis:"telemetry.SENSOR_WALLBOX_LOGIN_THREADS"`
		WallboxLoginMemory      float64 `redis:"telemetry.SENSOR_WALLBOX_LOGIN_MEMORY"`
		WallboxLoginSimpleState float64 `redis:"telemetry.SENSOR_WALLBOX_LOGIN_SIMPLE_STATE"`

		WallboxNetworkCPUUsage    float64 `redis:"telemetry.SENSOR_WALLBOX_NETWORK_CPU_USAGE"`
		WallboxNetworkThreads     float64 `redis:"telemetry.SENSOR_WALLBOX_NETWORK_THREADS"`
		WallboxNetworkMemory      float64 `redis:"telemetry.SENSOR_WALLBOX_NETWORK_MEMORY"`
		WallboxNetworkSimpleState float64 `redis:"telemetry.SENSOR_WALLBOX_NETWORK_SIMPLE_STATE"`

		WallboxSMachineCPUUsage    float64 `redis:"telemetry.SENSOR_WALLBOXSMACHINE_CPU_USAGE"`
		WallboxSMachineThreads     float64 `redis:"telemetry.SENSOR_WALLBOXSMACHINE_THREADS"`
		WallboxSMachineMemory      float64 `redis:"telemetry.SENSOR_WALLBOXSMACHINE_MEMORY"`
		WallboxSMachineSimpleState float64 `redis:"telemetry.SENSOR_WALLBOXSMACHINE_SIMPLE_STATE"`

		WallcoAdapterCPUUsage    float64 `redis:"telemetry.SENSOR_WALLCO_ADAPTER_CPU_USAGE"`
		WallcoAdapterThreads     float64 `redis:"telemetry.SENSOR_WALLCO_ADAPTER_THREADS"`
		WallcoAdapterMemory      float64 `redis:"telemetry.SENSOR_WALLCO_ADAPTER_MEMORY"`
		WallcoAdapterSimpleState float64 `redis:"telemetry.SENSOR_WALLCO_ADAPTER_SIMPLE_STATE"`

		WBXChargerInfoCPUUsage    float64 `redis:"telemetry.SENSOR_WBX_CHARGER_INFO_CPU_USAGE"`
		WBXChargerInfoThreads     float64 `redis:"telemetry.SENSOR_WBX_CHARGER_INFO_THREADS"`
		WBXChargerInfoMemory      float64 `redis:"telemetry.SENSOR_WBX_CHARGER_INFO_MEMORY"`
		WBXChargerInfoSimpleState float64 `redis:"telemetry.SENSOR_WBX_CHARGER_INFO_SIMPLE_STATE"`

		WPASupplicantCPUUsage    float64 `redis:"telemetry.SENSOR_WPA_SUPPLICANT_CPU_USAGE"`
		WPASupplicantThreads     float64 `redis:"telemetry.SENSOR_WPA_SUPPLICANT_THREADS"`
		WPASupplicantMemory      float64 `redis:"telemetry.SENSOR_WPA_SUPPLICANT_MEMORY"`
		WPASupplicantSimpleState float64 `redis:"telemetry.SENSOR_WPA_SUPPLICANT_SIMPLE_STATE"`

		NonWallboxCPUUsage    float64 `redis:"telemetry.SENSOR_NON_WALLBOX_CPU_USAGE"`
		NonWallboxThreads     float64 `redis:"telemetry.SENSOR_NON_WALLBOX_THREADS"`
		NonWallboxMemory      float64 `redis:"telemetry.SENSOR_NON_WALLBOX_MEMORY"`
		NonWallboxSimpleState float64 `redis:"telemetry.SENSOR_NON_WALLBOX_SIMPLE_STATE"`

		AvailableMemory      float64 `redis:"telemetry.SENSOR_AVAILABLE_MEMORY"`
		CMAFreeMemory        float64 `redis:"telemetry.SENSOR_CMAFREE_MEMORY"`
		AvailableStorageRoot float64 `redis:"telemetry.SENSOR_AVAILABLE_STORAGE_ROOTFS"`
		CPUTemperature       float64 `redis:"telemetry.SENSOR_CPU_TEMPERATURE"`
		SystemUptime         float64 `redis:"telemetry.SENSOR_SYSTEM_UPTIME"`
	}
}

type Wallbox struct {
	redisClient          *redis.Client
	sqlClient            *sqlx.DB
	Data                 DataCache
	ChargerType          string `db:"charger_type"`
	telemetryOCPPStatus  int
	telemetryOCPPUpdated time.Time
	journalOCPPStatus    int
	journalOCPPUpdated   time.Time
	ocppStatusMux        sync.RWMutex
	// HasTelemetry becomes true once we have successfully processed at least
	// one telemetry event and mapped it into RedisTelemetry. This lets higher
	// layers prefer telemetry-based values on newer firmware while keeping a
	// fallback to legacy Redis/M2W data for older firmware.
	HasTelemetry          bool
	pubsub                *redis.PubSub
	eventHandler          func(channel string, message string)
	sessionEnergyBaseline float64
	journalStopCh         chan struct{}
}

func New() *Wallbox {
	var w Wallbox

	var err error
	w.sqlClient, err = sqlx.Connect("mysql", "root:fJmExsJgmKV7cq8H@tcp(127.0.0.1:3306)/wallbox")
	if err != nil {
		panic(err)
	}

	query := "select SUBSTRING_INDEX(part_number, '-', 1) AS charger_type from charger_info;"
	w.sqlClient.Get(&w, query)

	w.redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	w.telemetryOCPPStatus = -1
	w.journalOCPPStatus = -1

	return &w
}

func getRedisFields(obj interface{}) []string {
	var result []string
	val := reflect.ValueOf(obj)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		result = append(result, field.Tag.Get("redis"))
	}

	return result
}

func (w *Wallbox) RefreshData() {
	ctx := context.Background()

	stateRes := w.redisClient.HMGet(ctx, "state", getRedisFields(w.Data.RedisState)...)
	if stateRes.Err() != nil {
		panic(stateRes.Err())
	}

	if err := stateRes.Scan(&w.Data.RedisState); err != nil {
		panic(err)
	}

	m2wRes := w.redisClient.HMGet(ctx, "m2w", getRedisFields(w.Data.RedisM2W)...)
	if m2wRes.Err() != nil {
		panic(m2wRes.Err())
	}

	if err := m2wRes.Scan(&w.Data.RedisM2W); err != nil {
		panic(err)
	}

	query := "SELECT " +
		"  `wallbox_config`.`charging_enable`," +
		"  `wallbox_config`.`lock`," +
		"  `wallbox_config`.`max_charging_current`," +
		"  `wallbox_config`.`halo_brightness`," +
		"  `power_outage_values`.`charged_energy` AS cumulative_added_energy," +
		"  IF(`active_session`.`unique_id` != 0," +
		"    `active_session`.`charged_range`," +
		"    `latest_session`.`charged_range`) AS added_range," +
		"  IF(`active_session`.`unique_id` != 0," +
		"    `active_session`.`energy_total`," +
		"    0) AS active_session_energy_total " +
		"FROM `wallbox_config`," +
		"    `active_session`," +
		"    `power_outage_values`," +
		"    (SELECT * FROM `session` ORDER BY `id` DESC LIMIT 1) AS latest_session"
	w.sqlClient.Get(&w.Data.SQL, query)

	// We no longer need to refresh telemetry data from Redis
	// The telemetry data comes directly from Redis subscriptions and is stored only in memory
}

func (w *Wallbox) SerialNumber() string {
	var serialNumber string
	w.sqlClient.Get(&serialNumber, "SELECT `serial_num` FROM charger_info")
	return serialNumber
}

func (w *Wallbox) FirmwareVersion() string {
	var firmware string
	err := w.sqlClient.Get(&firmware, "SELECT `version` FROM `wallbox_version` ORDER BY `id` DESC LIMIT 1")
	if err == nil && firmware != "" {
		return firmware
	}

	var fallback string
	err = w.sqlClient.Get(&fallback, "SELECT `software_version` FROM `charger_info` LIMIT 1")
	if err == nil && fallback != "" {
		return fallback
	}

	return "unknown"
}

func (w *Wallbox) UserId() string {
	var userId string
	w.sqlClient.QueryRow("SELECT `user_id` FROM `users` WHERE `user_id` != 1 ORDER BY `user_id` DESC LIMIT 1").Scan(&userId)
	return userId
}

func (w *Wallbox) AvailableCurrent() int {
	var availableCurrent int
	w.sqlClient.QueryRow("SELECT `max_avbl_current` FROM `state_values` ORDER BY `id` DESC LIMIT 1").Scan(&availableCurrent)
	return availableCurrent
}

// ChargingCurrentL1 returns the phase 1 charging current. On newer firmware
// this is sourced from telemetry events; on older firmware it falls back to
// the legacy m2w Redis hash.
func (w *Wallbox) ChargingCurrentL1() float64 {
	if w.HasTelemetry && w.Data.RedisTelemetry.InternalMeterCurrentL1 != 0 {
		return w.Data.RedisTelemetry.InternalMeterCurrentL1
	}
	return w.Data.RedisM2W.Line1Current
}

// ChargingCurrentL2 returns the phase 2 charging current, using telemetry when
// available and falling back to the legacy m2w Redis hash otherwise.
func (w *Wallbox) ChargingCurrentL2() float64 {
	if w.HasTelemetry && w.Data.RedisTelemetry.InternalMeterCurrentL2 != 0 {
		return w.Data.RedisTelemetry.InternalMeterCurrentL2
	}
	return w.Data.RedisM2W.Line2Current
}

// ChargingCurrentL3 returns the phase 3 charging current, using telemetry when
// available and falling back to the legacy m2w Redis hash otherwise.
func (w *Wallbox) ChargingCurrentL3() float64 {
	if w.HasTelemetry && w.Data.RedisTelemetry.InternalMeterCurrentL3 != 0 {
		return w.Data.RedisTelemetry.InternalMeterCurrentL3
	}
	return w.Data.RedisM2W.Line3Current
}

// linePowerFromTelemetry derives per‑phase power from internal meter voltage
// and current telemetry values. This is primarily used on newer firmware where
// legacy m2w per‑phase power may no longer be populated.
func linePowerFromTelemetry(voltage, current float64) float64 {
	if voltage == 0 || current == 0 {
		return 0
	}
	return voltage * current
}

// ChargingPowerL1 returns per‑phase power for L1. On newer firmware we derive
// this from internal meter telemetry, otherwise we fall back to legacy m2w
// power values.
func (w *Wallbox) ChargingPowerL1() float64 {
	if w.HasTelemetry &&
		(w.Data.RedisTelemetry.InternalMeterVoltageL1 != 0 ||
			w.Data.RedisTelemetry.InternalMeterCurrentL1 != 0) {
		return linePowerFromTelemetry(
			w.Data.RedisTelemetry.InternalMeterVoltageL1,
			w.Data.RedisTelemetry.InternalMeterCurrentL1,
		)
	}
	return w.Data.RedisM2W.Line1Power
}

// ChargingPowerL2 returns per‑phase power for L2. See ChargingPowerL1 for
// details.
func (w *Wallbox) ChargingPowerL2() float64 {
	if w.HasTelemetry &&
		(w.Data.RedisTelemetry.InternalMeterVoltageL2 != 0 ||
			w.Data.RedisTelemetry.InternalMeterCurrentL2 != 0) {
		return linePowerFromTelemetry(
			w.Data.RedisTelemetry.InternalMeterVoltageL2,
			w.Data.RedisTelemetry.InternalMeterCurrentL2,
		)
	}
	return w.Data.RedisM2W.Line2Power
}

// ChargingPowerL3 returns per‑phase power for L3. See ChargingPowerL1 for
// details.
func (w *Wallbox) ChargingPowerL3() float64 {
	if w.HasTelemetry &&
		(w.Data.RedisTelemetry.InternalMeterVoltageL3 != 0 ||
			w.Data.RedisTelemetry.InternalMeterCurrentL3 != 0) {
		return linePowerFromTelemetry(
			w.Data.RedisTelemetry.InternalMeterVoltageL3,
			w.Data.RedisTelemetry.InternalMeterCurrentL3,
		)
	}
	return w.Data.RedisM2W.Line3Power
}

// ChargingPower returns total charging power across all phases.
func (w *Wallbox) ChargingPower() float64 {
	return w.ChargingPowerL1() + w.ChargingPowerL2() + w.ChargingPowerL3()
}

// TemperatureL1 returns the line 1 temperature, preferring telemetry values
// when available and otherwise falling back to legacy m2w data.
func (w *Wallbox) TemperatureL1() float64 {
	if w.HasTelemetry && w.Data.RedisTelemetry.TempL1 != 0 {
		return w.Data.RedisTelemetry.TempL1
	}
	return w.Data.RedisM2W.TempL1
}

// TemperatureL2 returns the line 2 temperature, preferring telemetry values
// when available and otherwise falling back to legacy m2w data.
func (w *Wallbox) TemperatureL2() float64 {
	if w.HasTelemetry && w.Data.RedisTelemetry.TempL2 != 0 {
		return w.Data.RedisTelemetry.TempL2
	}
	return w.Data.RedisM2W.TempL2
}

// TemperatureL3 returns the line 3 temperature, preferring telemetry values
// when available and otherwise falling back to legacy m2w data.
func (w *Wallbox) TemperatureL3() float64 {
	if w.HasTelemetry && w.Data.RedisTelemetry.TempL3 != 0 {
		return w.Data.RedisTelemetry.TempL3
	}
	return w.Data.RedisM2W.TempL3
}

func sendToPosixQueue(path, data string) {
	pathBytes := append([]byte(path), 0)
	mq := mqOpen(pathBytes)

	event := []byte(data)
	eventPaddedBytes := append(event, bytes.Repeat([]byte{0x00}, 1024-len(event))...)

	mqTimedsend(mq, eventPaddedBytes)
	mqClose(mq)
}

func (w *Wallbox) SetLocked(lock int) {
	w.RefreshData()
	if lock == w.Data.SQL.Lock {
		return
	}
	if w.ChargerType == "CPB1" {
		w.sqlClient.MustExec("UPDATE `wallbox_config` SET `lock`=?", lock)
	} else if lock == 1 {
		sendToPosixQueue("WALLBOX_MYWALLBOX_WALLBOX_LOGIN", "EVENT_REQUEST_LOCK")
	} else {
		userId := w.UserId()
		sendToPosixQueue("WALLBOX_MYWALLBOX_WALLBOX_LOGIN", "EVENT_REQUEST_LOGIN#"+userId+".000000")
	}
}

func (w *Wallbox) SetChargingEnable(enable int) {
	w.RefreshData()
	if enable == w.Data.SQL.ChargingEnable {
		return
	}
	if enable == 1 {
		sendToPosixQueue("WALLBOX_MYWALLBOX_WALLBOX_STATEMACHINE", "EVENT_REQUEST_USER_ACTION#1.000000")
	} else {
		sendToPosixQueue("WALLBOX_MYWALLBOX_WALLBOX_STATEMACHINE", "EVENT_REQUEST_USER_ACTION#2.000000")
	}
}

func (w *Wallbox) SetMaxChargingCurrent(current int) {
	w.sqlClient.MustExec("UPDATE `wallbox_config` SET `max_charging_current`=?", current)
}

func (w *Wallbox) SetHaloBrightness(brightness int) {
	w.sqlClient.MustExec("UPDATE `wallbox_config` SET `halo_brightness`=?", brightness)
}

func (w *Wallbox) CableConnected() int {
	if w.HasTelemetry {
		status := int(w.Data.RedisTelemetry.ControlPilotStatus)
		if status != 0 && isTelemetryCableConnected(status) {
			return 1
		}
		return 0
	}

	if w.Data.RedisM2W.ChargerStatus == 0 || w.Data.RedisM2W.ChargerStatus == 6 {
		return 0
	}
	return 1
}

func (w *Wallbox) EffectiveStatus() string {
	if w.HasTelemetry && w.Data.RedisTelemetry.StateMachine != 0 {
		return describeTelemetryStatus(int(w.Data.RedisTelemetry.StateMachine))
	}

	tmsStatus := w.Data.RedisM2W.ChargerStatus
	state := w.Data.RedisState.SessionState

	if override, ok := stateOverrides[state]; ok {
		tmsStatus = override
	}

	if tmsStatus >= 0 && tmsStatus < len(wallboxStatusCodes) {
		return wallboxStatusCodes[tmsStatus]
	}

	return "Unknown"
}

func (w *Wallbox) ControlPilotStatus() string {
	if w.HasTelemetry && w.Data.RedisTelemetry.ControlPilotStatus != 0 {
		status := int(w.Data.RedisTelemetry.ControlPilotStatus)
		if desc, ok := telemetryControlPilotStates[status]; ok {
			return fmt.Sprintf("%d: %s", status, desc)
		}
		return fmt.Sprintf("%d: %s", status, describeTelemetryStatus(status))
	}

	if desc, ok := controlPilotStates[w.Data.RedisState.ControlPilot]; ok {
		return fmt.Sprintf("%d: %s", w.Data.RedisState.ControlPilot, desc)
	}
	return fmt.Sprintf("%d: Unknown", w.Data.RedisState.ControlPilot)
}

func (w *Wallbox) ControlPilotCode() int {
	if w.HasTelemetry && w.Data.RedisTelemetry.ControlPilotStatus != 0 {
		return int(w.Data.RedisTelemetry.ControlPilotStatus)
	}
	return w.Data.RedisState.ControlPilot
}

func (w *Wallbox) ControlPilotLetter() string {
	code := w.ControlPilotCode()
	if letter, ok := telemetryControlPilotLetters[code]; ok {
		return letter
	}
	return "Unknown"
}

func (w *Wallbox) IsChargingPilot() bool {
	return isTelemetryCharging(w.ControlPilotCode())
}

func (w *Wallbox) OCPPStatusCode() int {
	if code, ok := w.getJournalOCPPStatus(); ok {
		return code
	}
	if code, ok := w.getTelemetryOCPPStatus(); ok {
		return code
	}
	return int(w.Data.RedisTelemetry.OCPPStatus)
}

func (w *Wallbox) OCPPStatusDescription() string {
	return describeOCPPStatus(w.OCPPStatusCode())
}

func (w *Wallbox) OCPPIndicatesDisconnect() bool {
	return ocppStatusIndicatesDisconnect(w.OCPPStatusCode())
}

func (w *Wallbox) SetTelemetryOCPPStatus(code int) {
	w.ocppStatusMux.Lock()
	w.telemetryOCPPStatus = code
	w.telemetryOCPPUpdated = time.Now()
	w.ocppStatusMux.Unlock()
}

// OCPPOnlineCode reads the Wallbox redis flag that indicates OCPP online state.
// 4 typically means connected; 1/2 indicate problems; 0/absent often mean disabled.
func (w *Wallbox) OCPPOnlineCode() int {
	val, err := w.redisClient.Get(context.Background(), "wallbox:ocpp::online").Int()
	if err != nil {
		return -1
	}
	return val
}

// OCPPEnabled reports whether OCPP is enabled (any non-zero online flag).
func (w *Wallbox) OCPPEnabled() string {
	if code := w.OCPPOnlineCode(); code > 0 {
		return "1"
	}
	return "0"
}

// OCPPConnected reports whether OCPP is connected to the backend (online flag == 4).
func (w *Wallbox) OCPPConnected() string {
	if w.OCPPOnlineCode() == 4 {
		return "1"
	}
	return "0"
}

func (w *Wallbox) ConnectionType() string {
	if !w.HasTelemetry || w.Data.RedisTelemetry.ConnectionType == 0 {
		return "Unknown"
	}
	code := int(w.Data.RedisTelemetry.ConnectionType)
	return describeConnectionType(code)
}

func (w *Wallbox) ConnectivityStatus() string {
	if !w.HasTelemetry {
		return "Unknown"
	}
	code := int(w.Data.RedisTelemetry.ConnectivityStatus)
	return describeConnectivityStatus(code)
}

func (w *Wallbox) ControlMode() string {
	if !w.HasTelemetry {
		return "Unknown"
	}
	code := int(w.Data.RedisTelemetry.ControlMode)
	return describeControlMode(code)
}

func (w *Wallbox) ScheduleStatus() string {
	if !w.HasTelemetry {
		return "Unknown"
	}
	return describeScheduleStatus(int(w.Data.RedisTelemetry.ScheduleStatus))
}

func (w *Wallbox) EcosmartStatus() string {
	if !w.HasTelemetry {
		return "Unknown"
	}
	return describeEcosmartStatus(int(w.Data.RedisTelemetry.EcosmartStatus))
}

func (w *Wallbox) PowerBoostStatus() string {
	if !w.HasTelemetry {
		return "Unknown"
	}
	return describePowerBoostStatus(int(w.Data.RedisTelemetry.PowerboostStatus))
}

func (w *Wallbox) PowerSharingStatus() string {
	if !w.HasTelemetry {
		return "Unknown"
	}
	return describePowerSharingStatus(int(w.Data.RedisTelemetry.PowerSharingStatus))
}

func (w *Wallbox) MIDStatus() string {
	if !w.HasTelemetry {
		return "Unknown"
	}
	return describeMIDStatus(int(w.Data.RedisTelemetry.MidStatus))
}

func (w *Wallbox) PowerRelayCommand() string {
	if !w.HasTelemetry {
		return "Unknown"
	}
	return describePowerRelayCommand(int(w.Data.RedisTelemetry.PowerRelayManagementCommand))
}

func (w *Wallbox) getTelemetryOCPPStatus() (int, bool) {
	w.ocppStatusMux.RLock()
	code := w.telemetryOCPPStatus
	ts := w.telemetryOCPPUpdated
	w.ocppStatusMux.RUnlock()

	if code >= 0 && time.Since(ts) < 10*time.Minute {
		return code, true
	}
	return 0, false
}

func (w *Wallbox) SetJournalOCPPStatus(code int) {
	w.ocppStatusMux.Lock()
	w.journalOCPPStatus = code
	w.journalOCPPUpdated = time.Now()
	w.ocppStatusMux.Unlock()
}

func (w *Wallbox) getJournalOCPPStatus() (int, bool) {
	w.ocppStatusMux.RLock()
	code := w.journalOCPPStatus
	ts := w.journalOCPPUpdated
	w.ocppStatusMux.RUnlock()

	if code >= 0 && time.Since(ts) < 10*time.Minute {
		return code, true
	}
	return 0, false
}

func (w *Wallbox) StateMachineState() string {
	if w.HasTelemetry && w.Data.RedisTelemetry.StateMachine != 0 {
		status := int(w.Data.RedisTelemetry.StateMachine)
		return fmt.Sprintf("%d: %s", status, describeTelemetryStatus(status))
	}

	if desc, ok := stateMachineStates[w.Data.RedisState.SessionState]; ok {
		return fmt.Sprintf("%d: %s", w.Data.RedisState.SessionState, desc)
	}

	return fmt.Sprintf("%d: Unknown", w.Data.RedisState.SessionState)
}

func (w *Wallbox) ChargingEnable() int {
	if w.HasTelemetry && w.Data.RedisTelemetry.ChargingEnable != 0 {
		return int(w.Data.RedisTelemetry.ChargingEnable)
	}
	return w.Data.SQL.ChargingEnable
}

func (w *Wallbox) S2Open() int {
	if w.HasTelemetry {
		status := int(w.Data.RedisTelemetry.ControlPilotStatus)
		if status != 0 {
			if describeTelemetryStatus(status) == "Charging" {
				return 0
			}
			return 1
		}
	}

	return w.Data.RedisState.S2open
}

func (w *Wallbox) AddedEnergy() float64 {
	if w.Data.SQL.ActiveSessionEnergyTotal > 0 {
		return w.Data.SQL.ActiveSessionEnergyTotal
	}

	if w.HasTelemetry && w.Data.RedisTelemetry.InternalMeterEnergy != 0 {
		status := int(w.Data.RedisTelemetry.StateMachine)
		current := w.Data.RedisTelemetry.InternalMeterEnergy

		if !isChargingTelemetryStatus(status) && current > 0 {
			w.sessionEnergyBaseline = current
			return 0
		}

		if w.sessionEnergyBaseline == 0 {
			w.sessionEnergyBaseline = current
		}

		delta := current - w.sessionEnergyBaseline
		if delta < 0 {
			return 0
		}
		return delta
	}
	return w.Data.RedisState.ScheduleEnergy
}

func (w *Wallbox) SetEventHandler(handler func(channel string, message string)) {
	w.eventHandler = handler
}

func (w *Wallbox) StartRedisSubscriptions() {
	channels := []string{
		"/wbx/telemetry/events",
		"/wbx/charger_state_machine/events",
		"/wbx/charging_regulation/in/session",
		"/wbx/domain_bus/event/CHARGER_STATUS_CHANGED",
	}

	w.pubsub = w.redisClient.Subscribe(context.Background(), channels...)

	// Start goroutine to handle messages
	go func() {
		ch := w.pubsub.Channel()
		for msg := range ch {
			switch msg.Channel {
			case "/wbx/telemetry/events":
				w.ProcessTelemetryEvent(msg.Payload)
			case "/wbx/charger_state_machine/events", "/wbx/charging_regulation/in/session":
				w.ProcessSessionUpdateEvent(msg.Payload)
			case "/wbx/domain_bus/event/CHARGER_STATUS_CHANGED":
				w.ProcessChargerStatusEvent(msg.Payload)
			}

			if w.eventHandler != nil {
				w.eventHandler(msg.Channel, msg.Payload)
			}
		}
	}()
}

func (w *Wallbox) StopRedisSubscriptions() {
	if w.pubsub != nil {
		w.pubsub.Close()
	}
}

// StartOCPPJournalWatcher spawns a background goroutine that tails the
// ocppwallbox journald stream and extracts OCPP StatusNotification "status"
// values (Available, Charging, SuspendedEV, etc). These are mapped to
// numeric OCPP status codes and fed into SetJournalOCPPStatus, which is
// preferred by OCPPStatusCode over session/telemetry-based fallbacks.
func (w *Wallbox) StartOCPPJournalWatcher() {
	// Avoid starting multiple watchers if called more than once.
	if w.journalStopCh != nil {
		return
	}

	stopCh := make(chan struct{})
	w.journalStopCh = stopCh

	go func() {
		defer func() {
			// Best-effort cleanup of the child process if we managed to start it.
			w.journalStopCh = nil
		}()

		cmd := exec.Command("journalctl",
			"-u", "ocppwallbox.service",
			"-f",      // follow new entries
			"-n", "0", // do not replay historical logs
			"-o", "cat", // message only, no metadata
			"-q", // quiet
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("OCPP journal: failed to open stdout: %v", err)
			return
		}

		if err := cmd.Start(); err != nil {
			log.Printf("OCPP journal: failed to start journalctl: %v", err)
			return
		}
		defer func() {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}()

		scanner := bufio.NewScanner(stdout)
		for {
			select {
			case <-stopCh:
				return
			default:
			}

			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					log.Printf("OCPP journal: scanner error: %v", err)
				}
				return
			}

			line := scanner.Text()
			status, ok := parseOCPPStatusFromLogLine(line)
			if !ok {
				continue
			}

			if code, found := LookupOCPPStatusCode(status); found {
				w.SetJournalOCPPStatus(code)
			} else {
				log.Printf("OCPP journal: unknown StatusNotification status %q in line: %s", status, line)
			}
		}
	}()
}

// StopOCPPJournalWatcher signals the background journal watcher (if any) to
// stop and lets the goroutine tear down its journalctl process.
func (w *Wallbox) StopOCPPJournalWatcher() {
	if w.journalStopCh == nil {
		return
	}
	close(w.journalStopCh)
	w.journalStopCh = nil
}

// StartTimeConstrRedisSubscriptions starts Redis subscriptions and automatically stops them after the specified duration
func (w *Wallbox) StartTimeConstrainedRedisSubscriptions(duration time.Duration) {
	w.StartRedisSubscriptions()

	// Set up a timer to stop the subscription after the specified duration
	time.AfterFunc(duration, func() {
		log.Printf("Subscription time limit of %v reached. Stopping subscriptions...", duration)
		w.StopRedisSubscriptions()
	})
}

// TelemetryEvent represents the structure of telemetry events
type TelemetryEvent struct {
	Body struct {
		Sensors []struct {
			ID        string   `json:"id"`
			Metadata  []string `json:"metadata"`
			Timestamp string   `json:"timestamp"`
			Value     float64  `json:"value"`
		} `json:"sensors"`
	} `json:"body"`
	Header struct {
		MessageID string `json:"message_id"`
		Source    string `json:"source"`
		Timestamp string `json:"timestamp"`
	} `json:"header"`
}

type SessionUpdateEvent struct {
	Body struct {
		Session struct {
			State         string `json:"state"`
			InSession     bool   `json:"in_session"`
			ControlMode   string `json:"control_mode"`
			ControlAction string `json:"control_action"`
		} `json:"session"`
	} `json:"body"`
	Header struct {
		MessageID string `json:"message_id"`
		Source    string `json:"source"`
		Timestamp string `json:"timestamp"`
	} `json:"header"`
}

type ChargerStatusEvent struct {
	Body struct {
		OCPPStatusNumeric float64 `json:"ocpp_status"`
		OCPPStatusString  string  `json:"ocpp_status_string"`
	} `json:"body"`
	Header struct {
		MessageID string `json:"message_id"`
		Source    string `json:"source"`
		Timestamp string `json:"timestamp"`
	} `json:"header"`
}

// ProcessTelemetryEvent processes telemetry events and updates the RedisTelemetry struct
func (w *Wallbox) ProcessTelemetryEvent(payload string) {
	var event TelemetryEvent
	err := json.Unmarshal([]byte(payload), &event)
	if err != nil {
		log.Printf("Error unmarshalling telemetry event: %v", err)
		return
	}

	// Process each sensor in the event
	for _, sensor := range event.Body.Sensors {
		// Directly update the RedisTelemetry struct based on the sensor ID
		w.updateTelemetryField(sensor.ID, sensor.Value)
	}
}

// updateTelemetryField updates a specific field in the RedisTelemetry struct by sensor ID
func (w *Wallbox) updateTelemetryField(sensorID string, value float64) {
	// Use reflection to update the appropriate field in the RedisTelemetry struct
	v := reflect.ValueOf(&w.Data.RedisTelemetry).Elem()
	t := v.Type()

	// Iterate through struct fields to find the matching one
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		redisTag := field.Tag.Get("redis")

		// Check if this field's redis tag matches our telemetry key
		if redisTag == "telemetry."+sensorID {
			// Mark that we have seen at least one mapped telemetry sample so
			// higher‑level code can choose telemetry-backed values.
			w.HasTelemetry = true
			// Make sure the field is settable
			if v.Field(i).CanSet() {
				v.Field(i).SetFloat(value)
			}
			return
		}
	}

	// If we get here, we didn't find a matching field (might be a new sensor we're not tracking yet)
	// We could log this for debugging purposes
	log.Printf("No matching struct field found for sensor ID: %s", sensorID)
}

func (w *Wallbox) ProcessSessionUpdateEvent(payload string) {
	var event SessionUpdateEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		log.Printf("Error unmarshalling session event: %v", err)
		return
	}

	if event.Header.MessageID != "EVENT_SESSION_UPDATE" {
		return
	}

	state := event.Body.Session.State
	if state == "" {
		return
	}

	if code, ok := ocppCodeFromSessionState(state); ok {
		w.SetTelemetryOCPPStatus(code)
	} else {
		log.Printf("Unmapped session state for OCPP status: %s", state)
	}
}

func (w *Wallbox) ProcessChargerStatusEvent(payload string) {
	var event ChargerStatusEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		log.Printf("Error unmarshalling charger status event: %v", err)
		return
	}

	if err := w.redisClient.Set(context.Background(), "bridge:last_ocpp_status", payload, 0).Err(); err != nil {
		log.Printf("Failed to cache last OCPP status event: %v", err)
	}

	// We still consume the event for other telemetry fields and to cache the payload,
	// but we no longer override the OCPP status from this channel because the Wallbox
	// session events provide a fresher, more accurate view of the connector state.
}

var statusNotificationStatusRe = regexp.MustCompile(`status"\s*:\s*"([^"]+)"`)

// parseOCPPStatusFromLogLine extracts the OCPP StatusNotification "status" field
// from an ocppwallbox journald line. It returns the status string (e.g. "Available")
// and true on success, or ""/false if the line does not contain a parsable
// StatusNotification payload.
func parseOCPPStatusFromLogLine(line string) (string, bool) {
	if !strings.Contains(line, "StatusNotification") {
		return "", false
	}
	matches := statusNotificationStatusRe.FindStringSubmatch(line)
	if len(matches) < 2 {
		return "", false
	}
	status := strings.TrimSpace(matches[1])
	if status == "" {
		return "", false
	}
	return status, true
}

func ocppCodeFromSessionState(state string) (int, bool) {
	normalized := normalizeSessionState(state)
	switch normalized {
	case "ready":
		return 1, true
	case "finish", "lock", "waitunlock":
		return 6, true
	case "reserved":
		return 7, true
	case "updating", "unavailable", "psunconfig":
		return 8, true
	case "error", "unviable":
		return 9, true
	}

	if strings.HasPrefix(normalized, "connected") {
		return 5, true
	}

	if strings.HasPrefix(normalized, "waiting") ||
		strings.HasPrefix(normalized, "mid") ||
		strings.HasPrefix(normalized, "queue") {
		return 2, true
	}

	if strings.HasPrefix(normalized, "charging") ||
		strings.HasPrefix(normalized, "discharging") {
		return 3, true
	}

	if strings.HasPrefix(normalized, "paused") ||
		strings.HasPrefix(normalized, "scheduled") {
		return 5, true
	}

	return 0, false
}

func normalizeSessionState(state string) string {
	normalized := strings.ToLower(strings.TrimSpace(state))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	return normalized
}
