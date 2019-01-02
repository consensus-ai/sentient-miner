package main

import "testing"

func TestExcludedDevices(t *testing.T) {
	testSet := []struct {
		deviceID        int
		excludedDevices string
		excluded        bool
	}{
		{
			deviceID:        1,
			excludedDevices: "",
			excluded:        false,
		},
		{
			deviceID:        2,
			excludedDevices: "2",
			excluded:        true,
		},
		{
			deviceID:        2,
			excludedDevices: "3,2",
			excluded:        true,
		},
		{
			deviceID:        1,
			excludedDevices: "2,3",
			excluded:        false,
		},
		{
			deviceID:        1,
			excludedDevices: "0",
			excluded:        false,
		},
	}
	for _, test := range testSet {
		result := deviceExcludedForMining(test.deviceID, test.excludedDevices)
		if result != test.excluded {
			t.Error(test)
		}
	}
}
