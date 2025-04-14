package main

import (
	"time"
)

func checkExpirations() {
	for {
		circuitInfoMapLock.Lock()
		for k, v := range circuitInfoMap {
			if time.Now().Compare(v.ExpTime) == 1 {
				delete(circuitInfoMap, k)
			}
		}
		circuitInfoMapLock.Unlock()
		time.Sleep(5 * time.Second)
	}
}