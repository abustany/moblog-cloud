package rpcclient

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/abustany/moblog-cloud/pkg/idgenerator"
)

type Client struct {
	url         string
	idGenerator *idgenerator.StringIdGenerator

	Client *http.Client
}

func New(url string) *Client {
	return &Client{url, &idgenerator.StringIdGenerator{}, nil}
}

func (c *Client) Call(method string, params interface{}, result interface{}) error {
	callID := c.idGenerator.Next()

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
