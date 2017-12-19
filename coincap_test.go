// Copyright 2017 Aleksandr Demakin. All rights reserved.

package coincap

import (
	"testing"
	"time"
)

func TestGlobal(t *testing.T) {
	client := New()
	gl, err := client.Global()
	if err != nil {
		t.Error(err)
		return
	}
	if val, err := gl.AltCap.Float64(); err != nil {
		t.Error(err)
	} else if val == 0 {
		t.Error("null value")
	}
	if val, err := gl.VolumeTotal.Float64(); err != nil {
		t.Error(err)
	} else if val == 0 {
		t.Error("null value")
	}
}

func TestCoins(t *testing.T) {
	client := New()
	if coins, err := client.Coins(); err != nil {
		t.Error(err)
	} else if len(coins) == 0 {
		t.Error("empty coins")
	}
}

func TestFrontXCP(t *testing.T) {
	client := New()
	if coins, err := client.CoinsXCP(); err != nil {
		t.Error(err)
	} else if len(coins) == 0 {
		t.Error("empty coins")
	}
}

func TestPageCoin(t *testing.T) {
	client := New()
	if page, err := client.Page("ETH"); err != nil {
		t.Error(err)
	} else if len(page.ID) == 0 {
		t.Error("empty reply")
	}
}

func TestHistory(t *testing.T) {
	client := New()
	if hist, err := client.History("BTC", HistoryIntervalAll); err != nil {
		t.Error(err)
	} else if len(hist.MarketCap)*len(hist.Price)*len(hist.Volume) == 0 {
		t.Error("empty reply")
	}
}

func TestSubscribeTrades(t *testing.T) {
	client := New()
	tradeChan, stopChan := make(chan *Trade), make(chan bool)
	go func() {
		var received int
		afterChan := time.After(time.Second * 4)
		for loop := true; loop && received < 10; {
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
