package wallbox

import "testing"

func TestParseOCPPStatusFromLogLine_StatusNotificationAvailable(t *testing.T) {
	line := `Nov 23 22:49:54 WB225619 ocppwallbox[13222]: OCPP_STACK|2025-11-23|22:49:54.647|INFO |13222|WebSocketJsonClient.cpp|63|dropMessages::Sending Request to CS:[2,"1115475570","StatusNotification",{"info": "","vendorId": "com.wallbox","vendorErrorCode": "","connectorId": 1,"errorCode": "NoError","status": "Available","timestamp": "2025-11-23T22:49:54Z"}]`

	status, ok := parseOCPPStatusFromLogLine(line)
	if !ok {
		t.Fatalf("expected to parse status from line, but got ok=false")
	}
	if status != "Available" {
		t.Fatalf("expected status %q, got %q", "Available", status)
	}

	code, found := LookupOCPPStatusCode(status)
	if !found {
		t.Fatalf("expected to find OCPP status code for %q", status)
	}
	if code != 1 {
		t.Fatalf("expected OCPP status code 1 (Available), got %d", code)
	}
}

func TestParseOCPPStatusFromLogLine_NonStatusNotification(t *testing.T) {
	line := `Nov 23 22:49:54 WB225619 ocppwallbox[13222]: some other log line without StatusNotification`

	if status, ok := parseOCPPStatusFromLogLine(line); ok {
		t.Fatalf("expected no status to be parsed, but got %q", status)
	}
}


