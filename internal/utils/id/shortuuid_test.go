package id

import (
	"testing"
)

func TestShortID(t *testing.T) {
	id := ShortID()
	t.Log(id)
}

func TestDeviceId(t *testing.T) {

	// GetMachineId
	appId := "SC_SiDur7xmUAXY6pUtCYvsZz"
	machineId := DeviceId(appId)
	t.Logf("DeviceId(%s) = %s", appId, machineId)
	machineId2 := DeviceId(appId)
	t.Logf("DeviceId(%s) = %s", appId, machineId2)
	if machineId != machineId2 {
		t.Errorf("DeviceId(%s) = %s, want %s", appId, machineId2, machineId)
	} else {
		t.Logf("DeviceId(%s) = %s, want %s", appId, machineId2, machineId)
	}

}
