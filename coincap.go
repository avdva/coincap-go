// Copyright 2017 Aleksandr Demakin. All rights reserved.

// Package coincap is a simple coincap.io API wrapper library.
// Note, that in many structs json.Number is used instead of string or float64.
// This is because for some unknown reason coincap may send
// same fields as numbers or as strings.
package coincap

import (
	"encoding/json"
	"net/http"
	"time"

	gosio "github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"
	"github.com/pkg/errors"
)

// HistoryInterval* consts are used in History() request.
const (
	HistoryIntervalAll     = ""
	HistoryInterval1Day    = "1day"
	HistoryInterval7Days   = "7day"
	HistoryInterval30Days  = "30day"
	HistoryInterval90Days  = "90day"
	HistoryInterval180Days = "180day"
	HistoryInterval365Days = "365day"
)

const (
	cAPIURL = "https://coincap.io/"
	cWsURL  = "coincap.io"
)

// Front is a reply for /front path.
type Front struct {
	Long          string
	Short         string
	Shapeshift    bool
	Price         json.Number
	Cap24hrChange json.Number
	Mktcap        json.Number
	Perc          json.Number
	Supply        json.Number
	USDVolume     json.Number
	Volume        json.Number
	VwapData      *json.Number
	VwapDataBTC   *json.Number
}

// TradeData is a piece of information about a trade.
type TradeData struct {
	ExchangeID string `json:"exchange_id"`
	MarketID   string `json:"market_id"`
	Price      json.Number
	Raw        struct {
		ID        string
		TimeStamp time.Time
		Quantity  json.Number
		Price     json.Number
		Total     json.Number
		FillType  string
		OrderType string
	}
	TimestampMs int64 `json:"timestamp_ms"`
	Volume      json.Number
}

// TradeMessage is a websocket message from 'trades' channel.
type TradeMessage struct {
	ExchangeID string `json:"exchange_id"`
	MarketID   string `json:"market_id"`
	Coin       string
	Msg        Front
}

// Trade contains trade message and an optional trade data.
type Trade struct {
	Msg  TradeMessage
	Data TradeData
}

// Global is a websocket message from 'global' channel, and a reply for /global path.
type Global struct {
	BTCPrice      json.Number
	BTCCap        json.Number
	AltCap        json.Number
	Dom           json.Number
	BitnodesCount json.Number
	TotalCap      json.Number
	VolumeAlt     json.Number
	VolumeBtc     json.Number
	VolumeTotal   json.Number
}

// Mapping is a reply for /map path.
type Mapping struct {
	Name    string
	Symbol  string
	Aliases []string
}

// Page is a reply for /page path.
type Page struct {
	Global
	ID           string
	Type         string
	IDD          string `json:"_id"`
	DisplayName  string `json:"display_name"`
	Status       string
	PriceUSD     json.Number `json:"price_usd"`
	PriceEUR     json.Number `json:"price_eur"`
	PriceBTC     json.Number `json:"price_btc"`
	PriceETH     json.Number `json:"price_eth"`
	PriceZEC     json.Number `json:"price_zec"`
	PriceLTC     json.Number `json:"price_ltc"`
	MarketCap    json.Number `json:"market_cap"`
	Cap24hChange json.Number `json:"cap24hrChange"`
	Supply       json.Number
	Volume       json.Number
	Price        json.Number
	VWAP24h      json.Number `json:"vwap_h24"`
}

// History is a reply for /history path.
type History struct {
	Price     [][2]json.Number
	MarketCap [][2]json.Number `json:"market_cap"`
	Volume    [][2]json.Number
}

// Client send API requests and parses responses.
// It also can be used for subscription on websocket.
type Client struct {
	cl *http.Client
}

// New returns new Client.
func New() *Client {
	return &Client{cl: &http.Client{}}
}

// Coins requests /coins path.
func (c *Client) Coins() ([]string, error) {
	var result []string
	if err := c.get("coins", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CoinsXCP requests coins/xcp path
func (c *Client) CoinsXCP() ([]string, error) {
	var result []string
	if err := c.get("coins/xcp", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CoinsXCPAll requests coins/xcp/all path.
func (c *Client) CoinsXCPAll() ([]string, error) {
	var result []string
	if err := c.get("coins/xcp/all", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Map requests /map path.
func (c *Client) Map() ([]Mapping, error) {
	var result []Mapping
	if err := c.get("map", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Global requests /global path.
func (c *Client) Global() (Global, error) {
	var result Global
	err := c.get("global", &result)
	return result, err
}

// Front requests /front path.
func (c *Client) Front() ([]Front, error) {
	var result []Front
	if err := c.get("front", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// FrontXCP requests front/xcp path.
func (c *Client) FrontXCP() ([]Front, error) {
	var result []Front
	if err := c.get("front/xcp", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Page requests /page path for given symbol.
func (c *Client) Page(symb string) (*Page, error) {
	var result Page
	if err := c.get("page/"+symb, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// History requests /history path for given symbol.
//	interval can be either empty (returns all history on a coin),
//	or one of the HistoryInterval* consts.
func (c *Client) History(symb, interval string) (*History, error) {
	var result History
	if len(interval) > 0 {
		interval += "/"
	}
	if err := c.get("history/"+interval+symb, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) get(url string, value interface{}) error {
	resp, err := c.cl.Get(cAPIURL + url)
	if err != nil {
		return errors.Wrap(err, "http request error")
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(value); err != nil {
		return errors.Wrap(err, "failed to decode request")
	}
	return nil
}

// SubscribeTrades subscribes for websocket messages on 'trades' channel.
// All incoming messages are sent to 'dataChan'.
// If there are errors during subscription, it returns an error immediately.
// Otherwise, it blocks, waiting for an error, or stop signal.
// If an error occures, it will be returned as the result.
//	dataChan - a channel for Trade messages.
//	stopChan - a channel to cancel or reset ws subscribtion.
//		close it or send 'true' to stop subscribtion.
//		send 'false' to reconnect. May be useful, if updates stalled.
func (c *Client) SubscribeTrades(dataChan chan<- *Trade, stopChan <-chan bool) error {
	type wrapper struct {
		Message TradeMessage
		Trade   struct {
			Data TradeData
		}
	}
	return c.subscribe("trades", func(ch *gosio.Channel, tm wrapper) {
		dataChan <- &Trade{Msg: tm.Message, Data: tm.Trade.Data}
	}, stopChan)
}

func (c *Client) subscribe(method string, handler interface{}, stopChan <-chan bool) error {
	makeClient := func(errCh chan error) (*gosio.Client, error) {
		client, err := gosio.Dial(gosio.GetUrl(cWsURL, 443, true), transport.GetDefaultWebsocketTransport())
		if err != nil {
			return nil, errors.Wrap(err, "coincap: ws dial error")
		}
		defer func() {
			if err != nil {
				client.Close()
			}
		}()
		err = client.On(gosio.OnDisconnection, func(ch *gosio.Channel) {
			errCh <- errors.Errorf("websocket disconnected on channel %s", ch.Id())
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to setup disconnect handler")
		}
		err = client.On(gosio.OnError, func(ch *gosio.Channel) {
			errCh <- errors.Errorf("websocket error on channel %s", ch.Id())
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to setup error handler")
		}
		if err = client.On(method, handler); err != nil {
			return nil, errors.Wrap(err, "failed to setup message handler")
		}
		return client, nil
	}
	doConnect := func() (bool, error) {
		errCh := make(chan error, 2)
		client, err := makeClient(errCh)
		if err != nil {
			return false, err
		}
		defer client.Close()
		select {
		case err := <-errCh:
			return false, err
		case val, ok := <-stopChan:
			return ok && !val, nil
		}
	}
	for {
		if goon, err := doConnect(); !goon {
			return err
		}
	}
}
