# MQTT Bridge for Wallbox

This open-source project connects your Wallbox fully locally to Home Assistant, providing you with unparalleled speed and reliability.

> adds full telemetry support for firmware 6.7.x+ (control pilot, state machine, session energy, Power Boost, etc.) and tries to keep older firmware working automatically via legacy fallbacks.

> **Please note this is only tested against firmware 6.7.33 and 6.7.36 on the Pulsar Plus; it should work across 6.7.x as long as Wallbox does not introduce breaking changes, but some sensor readings may vary on other versions.**

## Features

- **Instant Sensor Data:** The Wallbox's internal state is polled every second and any updates are immediately pushed to the external MQTT broker.

- **Instant Control:** Quickly lock/unlock, pause/resume or change the max charging current, without involving the manufacturer's servers.

- **Always available:** As long as your local network is up and your Wallbox has power, you're in control! No need to rely on a third party to communicate with the device you own.

- **Home Assistant MQTT Auto Discovery:** Enjoy a hassle-free setup with Home Assistant MQTT Auto Discovery support. The integration effortlessly integrates with your existing Home Assistant environment.

<br/>
<p align="center">
   <img src="https://github.com/jagheterfredrik/wallbox-mqtt-bridge/assets/9987465/06488a5d-e6fe-4491-b11d-e7176792a7f5" height="507" />
</p>

## Getting Started

1. [Root your Wallbox](https://github.com/jagheterfredrik/wallbox-pwn)
2. Setup an MQTT Broker, if you don't already have one. Here's an example [installing it as a Home Assistant add-on](https://www.youtube.com/watch?v=dqTn-Gk4Qeo)
3. `ssh` to your Wallbox and run

```sh
curl -sSfL https://github.com/Leventionz/wallbox-mqtt-bridge/releases/download/bridgechannels-2025.12.06/install.sh > install.sh && bash install.sh
```

**If you intend to update on your charger, make sure no vehicle is connected to avoid any chance of this application being the cause of your charger rebooting mid-update.**

Note: To upgrade to new version, simply run the command from step 3 again.

## EVCC quickstart

- The installer now asks whether you want an EVCC helper file.
- Answer `y` and it will auto-detect your Wallbox serial (or prompt for it) and drop `~/mqtt-bridge/evcc-wallbox.yaml` containing the proper `meters`, `chargers`, and `loadpoints` sections.
- Copy that snippet into your EVCC config and adjust MQTT broker credentials on the EVCC side—topics already match the bridge’s Home Assistant entities.

## Firmware 6.7.x support

| Area | Behaviour on 6.7.x | Notes / fallback |
| --- | --- | --- |
| **Control pilot** | Telemetry control-pilot codes (161, 162, 177, 178, 193, 194, 195) drive `sensor.wallbox_control_pilot` **and** `binary_sensor.wallbox_cable_connected`. A companion `sensor.wallbox_control_pilot_state` converts those codes back to the familiar SAE/IEC letters (A/B/C). | Falls back to `state.ctrlPilot` on older firmware. |
| **State machine / status** | Telemetry `SENSOR_STATE_MACHINE` feeds `sensor.wallbox_state_machine`, `sensor.wallbox_status`, and the debug `sensor.wallbox_m2w_status`. Every code in the official Wallbox enum (Waiting, Scheduled, Paused, Charging, Locked, Updating, etc.) is mapped to a friendly string. | Falls back to the legacy `m2w/state` hashes and existing override tables automatically. |
| **OCPP visibility** | The bridge exposes `sensor.wallbox_ocpp_status` (codes 1–9 mapped to Available/Preparing/Charging/Suspended etc.), `binary_sensor.wallbox_ocpp_mismatch`, and `sensor.wallbox_ocpp_last_restart`. | `ocpp_status` now prefers the `StatusNotification` `status` values parsed from the `ocppwallbox` journald logs (Available/Preparing/Charging/SuspendedEV/…), then falls back to the Wallbox session events (`EVENT_SESSION_UPDATE`) and finally the telemetry `SENSOR_OCPP_STATUS` value. |
| **Session energy** | `sensor.wallbox_added_energy` now surfaces the current session Wh from MySQL (`active_session.energy_total`) whenever it is available, while `sensor.wallbox_cumulative_added_energy` remains the lifetime total. | When no active session total is available, it falls back to a telemetry baseline (Internal Meter Energy – baseline) or, on older firmware, to `scheduleEnergy`. |
| **S2 relay** | `sensor.wallbox_s2_open` is derived from control-pilot telemetry (S2 is “closed” only while telemetry reports a charging state). | Falls back to `state.S2open` where telemetry is unavailable. |
| **Charging enable** | `sensor.wallbox_charging_enable` mirrors the telemetry `SENSOR_CHARGING_ENABLE` flag so toggles are instantaneous. | Falls back to `wallbox_config.charging_enable` on older firmware. |
| **Power Boost** | When telemetry reports a PowerBoost session, the L1 sensors publish the telemetry proposal current/power; unused phases report `0`. If legacy `m2w` data exists (older firmware / multi-phase setups) it’s used automatically. | Assumes single-phase hardware unless telemetry supplies per-phase values. |
| **Other telemetry** | `charging_power*`, `charging_current*`, `temp_l*`, `status`, `control_pilot`, `state_machine`, `charging_enable`, `cable_connected`, and all debug telemetry entities emit live telemetry values out of the box. | Legacy data paths remain in place for <6.7.x devices. |

> If you update your Wallbox beyond 6.7.x, simply redeploy using the installer command above to keep the telemetry fixes in place. The bridge auto-detects telemetry and switches to legacy data when telemetry is missing.

## Key highlights (bridgechannels-2025.12.06)

- **Heal observability + graceful fallback** – OCPP self-heal now publishes action/detail/timestamp sensors (`ocpp_last_heal_action`, `ocpp_last_heal_detail`, `ocpp_last_heal_at`), prefers stop+start before restart, and only escalates to the Wallbox `reboot.sh` flow when needed. New OCPP enable/connected binary sensors surface backend connectivity.
- **Live OCPP visibility** – `sensor.wallbox_ocpp_status` follows `ocppwallbox` StatusNotification logs; if journald is unavailable it falls back to Wallbox session events and finally telemetry `SENSOR_OCPP_STATUS`.
- **Accurate energy + cable state** – `sensor.wallbox_added_energy` reads MySQL `active_session.energy_total`; `binary_sensor.wallbox_cable_connected` keys off telemetry control-pilot codes for true plug detection, with S2 relay and charging enable also driven from telemetry.
- **Cleaner telemetry + leaner HA entities** – resource metrics are mapped without log spam; the main HA sensor set stays focused while debug mode exposes the rest. Telemetry enums for schedule/ecosmart/powerboost/power sharing/MID/power relay command now render as friendly strings in Home Assistant.
- **EVCC helper + installer polish** – optional EVCC snippet, tolerant installer defaults (180 s mismatch, cooldowns), and Python/systemd resilience. Optional safeguard: reboot the Wallbox if control pilot error 14 persists (configurable timer).
- **Traceable builds** – version strings embed the release tag and commit (e.g., `bridgechannels-2025.12.06+<commit>`) for clear sw_version reporting in HA.

## Release notes (bridgechannels-2025.12.06)

- Added OCPP connectivity diagnostics: `binary_sensor.wallbox_ocpp_enabled` and `binary_sensor.wallbox_ocpp_connected` (uses Wallbox redis `wallbox:ocpp::online`).
- Added pilot error safeguard: optional reboot if control pilot stays in error state 14 for a configurable duration (default 5 minutes) via `pilot_error_reboot` / `pilot_error_seconds` in `install.sh`.
- Heal observability: OCPP self-heal publishes action/detail/timestamp sensors and prefers stop+start before restart, escalating to vendor `reboot.sh` only if needed. Removed the redundant `sensor.wallbox_ocpp_last_heal_error` (use mismatch + heal action/detail instead).
- Telemetry enums now render as friendly strings in HA for schedule/ecosmart/powerboost/power sharing/MID/power relay command/connectivity/control mode/connection type.
- Rebuilt armhf/arm64 binaries with embedded version tag/commit.

## OCPP self-healing & sensors

The installer (or `./bridge --config`) can auto-populate these settings:

```ini
[settings]
auto_restart_ocpp = true
ocpp_mismatch_seconds = 180           # how long the mismatch must persist
ocpp_restart_cooldown_seconds = 300   # wait time between restarts
ocpp_max_restarts = 3                 # how many service restarts before we stop or escalate
ocpp_full_reboot = false              # set to true to allow a full Wallbox reboot as a last resort
```

## Acknowledgments

The credits go out to jagheterfredrik (https://github.com/jagheterfredrik/wallbox-mqtt-bridge), who made the original MQTT Bridge for the Wallbox and jethrovo for his updated version supporting version v6.6.x.
A big shoutout to [@tronikos](https://github.com/tronikos) for their valuable contributions. This project wouldn't be the same without the collaborative spirit of the open-source community.
