package main

import (
	"log"
	"sync/atomic"
	"time"
)


func checkExpirations() {
	for {
		circuitInfoMapLock.Lock()
		for k, v := range circuitInfoMap {
			if time.Now().Compare(v.ExpTime) == 1 {
				log.Println("Deleted Circuit with ID:", k)
				delete(circuitInfoMap, k)
				atomic.AddInt32(&load, -1)
			}
		}
		circuitInfoMapLock.Unlock()
		time.Sleep(1 * time.Second)
	}
}