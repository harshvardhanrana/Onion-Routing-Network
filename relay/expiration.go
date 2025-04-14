package main

import (
	"log"
	"time"
)

func checkExpirations() {
	for {
		circuitInfoMapLock.Lock()
		for k, v := range circuitInfoMap {
			if time.Now().Compare(v.ExpTime) == 1 {
				log.Println("Deleted Circuit with ID:", k)
				delete(circuitInfoMap, k)
			}
		}
		circuitInfoMapLock.Unlock()
		time.Sleep(1 * time.Second)
	}
}