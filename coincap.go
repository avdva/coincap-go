// Copyright 2017 Aleksandr Demakin. All rights reserved.

// Package coincap is a simple coincap.io API wrapper library.
// Note, that in many structs json.Number is used instead of string or float64.
// This is because for some unknown reason coincap may send
// same fields as numbers or as strings.
package coincap

import (
	"encoding/json"
	"net/http"

	gosio "github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"
	"github.com/pkg/errors"
)

// HistoryInterval* consts are used in History() request.
const (
	HistoryInterval1Day    = "1day"
	HistoryInterval7Days   = "7day"
	HistoryInterval30Days  = "30day"
	HistoryInterval90Days  = "90day"
	HistoryInterval180Days = "180day"
	HistoryInterval365Days = "365day"
)

const (
	cAPIURL = "http://coincap.io/"
	cWsURL  = "socket.coincap.io"
)

type market struct {
	Time                 int64
	Shapeshift           bool
	Published            bool
	IsBuy                bool
	Short                string
	Long                 string
	Position24           json.Number
	Position             json.Number
	Perc                 json.Number
	Volume               json.Number
	USDVolume            json.Number
	Cap24hrChange        json.Number
	Mktcap               json.Number
	Supply               json.Number
	Cap24hrChangePercent json.Number
	VwapData             *json.Number
	VwapDataBTC          *json.Number
}

// Front is a reply for /front path.
type Front struct {
	market
	Price json.Number
}

// Trade is a websocket message from 'trades' channel.
type Trade struct {
	Coin string
	Msg  Front
}

// Global is a websocket message from 'global' channel,
// and a reply for /global path.
type Global struct {
	BTCPrice      json.Number
	BTCCap        json.Number
	AltCap        json.Number
	Dom           json.Number
	BitnodesCount json.Number
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
	priceAndCap
	market
	USDPrice      json.Number
	HomeURL       string
	ExplorerURL   string
	Twitter       string
	DiscussionURL string
	Mineable      bool
	Premined      bool
	PreminedSig   bool
}

type priceAndCap struct {
	Price      [][2]json.Number
	Market_Cap [][2]json.Number
}

// History is a reply for /history path.
type History struct {
	priceAndCap
	Volume [][2]json.Number
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

// SubscribeGlobal subscribes for websocket messages on 'global' channel.
// All incoming messages are sent to 'dataChan'.
// If there are errors during subscription, it returns an error immediately.
// Otherwise, it blocks, waiting for an error, or stop signal.
// If an error occures, it will be returned as the result.
// Send to/close of 'stopChan' will close connection, and nil will be returned.
func (c *Client) SubscribeGlobal(dataChan chan<- *Global, stopChan <-chan struct{}) error {
	type wrapper struct {
		Message Global
	}
	return c.subscribe("global", func(ch *gosio.Channel, gm *wrapper) {
		dataChan <- &gm.Message
	}, stopChan)
}

// SubscribeTrades subscribes for websocket messages on 'trades' channel.
// All incoming messages are sent to 'dataChan'.
// If there are errors during subscription, it returns an error immediately.
// Otherwise, it blocks, waiting for an error, or stop signal.
// If an error occures, it will be returned as the result.
// Send to/close of 'stopChan' will close connection, and nil will be returned.
func (c *Client) SubscribeTrades(dataChan chan<- *Trade, stopChan <-chan struct{}) error {
	type wrapper struct {
		Message Trade
	}
	return c.subscribe("trades", func(ch *gosio.Channel, tm *wrapper) {
		dataChan <- &tm.Message
	}, stopChan)
}

func (c *Client) subscribe(method string, handler interface{}, stopChan <-chan struct{}) error {
	client, err := gosio.Dial(gosio.GetUrl(cWsURL, 80, false), transport.GetDefaultWebsocketTransport())
	if err != nil {
		return errors.Wrap(err, "coincap: ws dial error")
	}
	defer client.Close()
	errCh := make(chan error, 2)
	err = client.On(gosio.OnDisconnection, func(ch *gosio.Channel) {
		errCh <- errors.Errorf("websocket disconnected on channel %s", ch.Id())
	})
	if err != nil {
		return errors.Wrap(err, "failed to setup disconnect handler")
	}
	err = client.On(gosio.OnError, func(ch *gosio.Channel) {
		errCh <- errors.Errorf("websocket error on channel %s", ch.Id())
	})
	if err != nil {
		return errors.Wrap(err, "failed to setup error handler")
	}
	if err = client.On(method, handler); err != nil {
		return errors.Wrap(err, "failed to setup message handler")
	}
	select {
	case err := <-errCh:
		return err
	case <-stopChan:
		return nil
	}
}
