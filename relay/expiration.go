package main

import (
	"log"
	"sync/atomic"
	"time"
)

const EXPIRATION_TIME = 5

func checkExpirations() {
	for {
		circuitInfoMapLock.Lock()
		for k, v := range circuitInfoMap {
			if time.Now().Compare(v.ExpTime) == EXPIRATION_TIME {
				log.Println("Deleted Circuit with ID:", k)
				delete(circuitInfoMap, k)
				atomic.AddInt32(&load, -1)
			}
		}
		circuitInfoMapLock.Unlock()
		time.Sleep(1 * time.Second)
	}
}