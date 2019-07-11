package rpcclient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/pkg/errors"
)

type Client struct {
	url     string
	lastUID uint64

	Client *http.Client
}

func New(url string) *Client {
	return &Client{url, 1, nil}
}

func (c *Client) Call(method string, params interface{}, result interface{}) error {
	callID := strconv.FormatUint(atomic.AddUint64(&c.lastUID, 1), 10)

	call := struct {
		ID     string        `json:"id"`
		Method string        `json:"method"`
		Params []interface{} `json:"params"`
	}{
		callID,
		method,
		[]interface{}{params},
	}

	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(&call); err != nil {
		return errors.Wrap(err, "Error while encoding call body")
	}

	httpClient := c.Client

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	res, err := httpClient.Post(c.url, "application/json", &body)

	if err != nil {
		return errors.Wrap(err, "Error while sending request")
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.Errorf("Invalid HTTP reply status: %d", res.StatusCode)
	}

	reply := struct {
		ID     string
		Error  string
		Result json.RawMessage
	}{}

	if err := json.NewDecoder(res.Body).Decode(&reply); err != nil {
		return errors.Wrap(err, "Error while decoding reply body")
	}

	if reply.ID != callID {
		return errors.New("Reply ID does not match call ID")
	}

	if reply.Error != "" {
		return errors.Errorf("RPC error: %s", reply.Error)
	}

	if err := json.Unmarshal(reply.Result, &result); err != nil {
		return errors.Wrap(err, "Error while decoding reply result")
	}

	return nil
}
