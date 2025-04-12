package wallbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type DataCache struct {
	SQL struct {
		Lock                  int     `db:"lock"`
		ChargingEnable        int     `db:"charging_enable"`
		MaxChargingCurrent    int     `db:"max_charging_current"`
		HaloBrightness        int     `db:"halo_brightness"`
		CumulativeAddedEnergy float64 `db:"cumulative_added_energy"`
		AddedRange            float64 `db:"added_range"`
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
		ICPMaxCurrent                    float64 `redis:"telemetry.SENSOR_ICP_MAX_CURRENT"`
		InternalMeterCurrentL1           float64 `redis:"telemetry.SENSOR_INTERNAL_METER_CURRENT_L1"`
		InternalMeterCurrentL2           float64 `redis:"telemetry.SENSOR_INTERNAL_METER_CURRENT_L2"`
		InternalMeterCurrentL3           float64 `redis:"telemetry.SENSOR_INTERNAL_METER_CURRENT_L3"`
		MaxAvailableCurrent              float64 `redis:"telemetry.SENSOR_MAX_AVAILABLE_CURRENT"`
		UserCurrentProposal              float64 `redis:"telemetry.SENSOR_USER_CURRENT_PROPOSAL"`
		DynamicPowerSharingMaxCurrent    float64 `redis:"telemetry.SENSOR_DYNAMIC_POWER_SHARING_MAX_CURRENT"`

		InternalMeterVoltageL1           float64 `redis:"telemetry.SENSOR_INTERNAL_METER_VOLTAGE_L1"`
		InternalMeterVoltageL2           float64 `redis:"telemetry.SENSOR_INTERNAL_METER_VOLTAGE_L2"`
		InternalMeterVoltageL3           float64 `redis:"telemetry.SENSOR_INTERNAL_METER_VOLTAGE_L3"`
		InternalMeterVoltageFilterStatus float64 `redis:"telemetry.SENSOR_INTERNAL_METER_VOLTAGE_FILTER_STATUS"`
		ControlPilotHighVolts            float64 `redis:"telemetry.SENSOR_CONTROL_PILOT_HIGH_TENTHS_OF_VOLTS"`
		ControlPilotLowVolts             float64 `redis:"telemetry.SENSOR_CONTROL_PILOT_LOW_TENTHS_OF_VOLTS"`

		InternalMeterEnergy              float64 `redis:"telemetry.SENSOR_INTERNAL_METER_ENERGY"`
		EcosmartGreenEnergy              float64 `redis:"telemetry.SENSOR_ECOSMART_GREEN_ENERGY"`
		EcosmartEnergyTotal              float64 `redis:"telemetry.SENSOR_ECOSMART_ENERGY_TOTAL"`

		EcosmartMode                     float64 `redis:"telemetry.SENSOR_ECOSMART_MODE"`
		EcosmartStatus                   float64 `redis:"telemetry.SENSOR_ECOSMART_STATUS"`
		EcosmartCurrentProposal          float64 `redis:"telemetry.SENSOR_ECOSMART_CURRENT_PROPOSAL"`

		InternalMeterFrequency           float64 `redis:"telemetry.SENSOR_INTERNAL_METER_FREQUENCY"`

		ScheduleStatus                   float64 `redis:"telemetry.SENSOR_SCHEDULE_STATUS"`
		ScheduleCurrentProposal          float64 `redis:"telemetry.SENSOR_SCHEDULE_CURRENT_PROPOSAL"`
		PowerboostStatus                 float64 `redis:"telemetry.SENSOR_DCA_POWERBOOST_STATUS"`
		PowerboostProposalCurrent        float64 `redis:"telemetry.SENSOR_POWERBOOST_PROPOSAL_CURRENT"`
		
		// Additional fields referenced in getTelemetryEventEntities
		ChargingEnable                   float64 `redis:"telemetry.SENSOR_CHARGING_ENABLE"`
		ControlPilotDuty                 float64 `redis:"telemetry.SENSOR_CONTROL_PILOT_DUTY"`
		ControlPilotStatus               float64 `redis:"telemetry.SENSOR_CONTROL_PILOT_STATUS"`
		MaxChargingCurrent               float64 `redis:"telemetry.SENSOR_MAX_CHARGING_CURRENT"`
		MidStatus                        float64 `redis:"telemetry.SENSOR_MID_STATUS"`
		PowerSharingStatus               float64 `redis:"telemetry.SENSOR_POWER_SHARING_STATUS"`
		TempL1                           float64 `redis:"telemetry.SENSOR_TEMP_L1"`
		TempL2                           float64 `redis:"telemetry.SENSOR_TEMP_L2"`
		TempL3                           float64 `redis:"telemetry.SENSOR_TEMP_L3"`
		Welding                          float64 `redis:"telemetry.SENSOR_WELDING"`
		FirmwareError                    float64 `redis:"telemetry.SENSOR_FIRMWARE_ERROR"`
		PowerRelayManagementCommand      float64 `redis:"telemetry.SENSOR_POWER_RELAY_MANAGEMENT_COMMAND"`
	}
}

type Wallbox struct {
	redisClient  *redis.Client
	sqlClient    *sqlx.DB
	Data         DataCache
	ChargerType  string `db:"charger_type"`
	pubsub       *redis.PubSub
	eventHandler func(channel string, message string)
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
		"    `latest_session`.`charged_range`) AS added_range " +
		"FROM `wallbox_config`," +
		"    `active_session`," +
		"    `power_outage_values`," +
		"    (SELECT * FROM `session` ORDER BY `id` DESC LIMIT 1) AS latest_session"
	w.sqlClient.Get(&w.Data.SQL, query)
}

func (w *Wallbox) refreshTelemetryData(ctx context.Context) {
	telemetryRes := w.redisClient.HMGet(ctx, "telemetry", getRedisFields(w.Data.RedisTelemetry)...)
	if telemetryRes.Err() != nil {
		log.Printf("Error fetching telemetry data: %v", telemetryRes.Err())
		return
	}

	if err := telemetryRes.Scan(&w.Data.RedisTelemetry); err != nil {
		log.Printf("Error scanning telemetry data: %v", err)
	}
}

func (w *Wallbox) SerialNumber() string {
	var serialNumber string
	w.sqlClient.Get(&serialNumber, "SELECT `serial_num` FROM charger_info")
	return serialNumber
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
	if w.Data.RedisM2W.ChargerStatus == 0 || w.Data.RedisM2W.ChargerStatus == 6 {
		return 0
	}
	return 1
}

func (w *Wallbox) EffectiveStatus() string {
	tmsStatus := w.Data.RedisM2W.ChargerStatus
	state := w.Data.RedisState.SessionState

	if override, ok := stateOverrides[state]; ok {
		tmsStatus = override
	}

	return wallboxStatusCodes[tmsStatus]
}

func (w *Wallbox) ControlPilotStatus() string {
	return fmt.Sprintf("%d: %s", w.Data.RedisState.ControlPilot, controlPilotStates[w.Data.RedisState.ControlPilot])
}

func (w *Wallbox) StateMachineState() string {
	return fmt.Sprintf("%d: %s", w.Data.RedisState.SessionState, stateMachineStates[w.Data.RedisState.SessionState])
}

func (w *Wallbox) SetEventHandler(handler func(channel string, message string)) {
	w.eventHandler = handler
}

func (w *Wallbox) StartRedisSubscriptions() {
	channels := []string{
		"/wbx/telemetry/events",
	}

	w.pubsub = w.redisClient.Subscribe(context.Background(), channels...)

	// Start goroutine to handle messages
	go func() {
		ch := w.pubsub.Channel()
		for msg := range ch {
			if msg.Channel == "/wbx/telemetry/events" {
				w.ProcessTelemetryEvent(msg.Payload)
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
			ID        string    `json:"id"`
			Metadata  []string  `json:"metadata"`
			Timestamp string    `json:"timestamp"`
			Value     float64   `json:"value"`
		} `json:"sensors"`
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
