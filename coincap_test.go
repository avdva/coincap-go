// Copyright 2017 Aleksandr Demakin. All rights reserved.

package coincap

import (
	"testing"
	"time"
)

func TestSubscribeGlobal(t *testing.T) {
	client := New()
	globalChan, stopChan := make(chan *Global), make(chan bool)
	go func() {
		var received int
		afterChan := time.After(time.Second * 30)
		select {
		case <-globalChan:
			received++
		case <-afterChan:
		}
		if received == 0 {
			t.Error("no messages in 30 sec")
		}
		stopChan <- true
	}()
	doneChan := make(chan struct{})
	go func() {
		if err := client.SubscribeGlobal(globalChan, stopChan); err != nil {
			t.Error(err)
		}
		doneChan <- struct{}{}
	}()
	select {
	case <-time.After(time.Second * 31):
		t.Error("subscribe did not return in 31 sec")
	case <-doneChan:
	}
}

func TestSubscribeTrades(t *testing.T) {
	client := New()
	tradeChan, stopChan := make(chan *Trade), make(chan bool)
	go func() {
		var received int
		afterChan := time.After(time.Second * 3)
		for loop := true; loop; {
			select {
			case <-tradeChan:
				if received == 0 { // re-subscribe after first message. after that we should receive some more.
					stopChan <- false
				}
				received++
			case <-afterChan:
				loop = false
			}
		}
		if received < 2 {
			t.Error("less than 2 messages in 3 sec")
		} else {
			t.Logf("%d messages received", received)
		}
		stopChan <- true
	}()
	doneChan := make(chan struct{})
	go func() {
		if err := client.SubscribeTrades(tradeChan, stopChan); err != nil {
			t.Error(err)
		}
		doneChan <- struct{}{}
	}()
	select {
	case <-time.After(time.Second * 4):
		t.Error("subscribe did not return in 4 sec")
	case <-doneChan:
	}
}
