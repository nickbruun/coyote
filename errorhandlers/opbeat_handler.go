package errorhandlers

import (
	"bytes"
	"net/http"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Test if a 

// Opbeat error handler.
type opbeatErrorHandler struct {
	appId string
	organizationId string
	secretToken string
}

func (h *opbeatErrorHandler) Handle(errMsg *Error) error {
	// Construct the payload.
	extra := make(map[string]string, len(errMsg.Environ))

	for k, v := range errMsg.Environ {
		if isAscii(k) && isAscii(v) {
			extra[k] = v
		}
	}

	payload := map[string]interface{}{
		"message": errMsg.Desc,
		"culprit": errMsg.QuotedCmd(),
		"extra": extra,
		"timestamp": errMsg.Timestamp,
	}

	if errMsg.Hostname != "" {
		payload["machine"] = map[string]string{
			"hostname": errMsg.Hostname,
		}
	}

	// Encode the payload and send the request.
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	fmt.Println(string(data))

	req, err := http.NewRequest("POST", fmt.Sprintf("https://intake.opbeat.com/api/v1/organizations/%s/apps/%s/errors/", h.organizationId, h.appId), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.secretToken))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(data))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	// Check the response.
	if resp.StatusCode < 200 || resp.StatusCode > 300 {
		respContentType := resp.Header.Get("Content-Type")

		if len(respContentType) >= len("application/json") && respContentType[:len("application/json")] == "application/json" {
			if respData, err := ioutil.ReadAll(resp.Body); err == nil {
				respJson := make(map[string]interface{})

				if err = json.Unmarshal(respData, &respJson); err == nil {
					if msg, ok := respJson["error_message"]; ok {
						return fmt.Errorf("error from Opbeat for status code %d: %s", resp.StatusCode, msg)
					}
				}
			}
		}

		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// New Opbeat error handler.
func NewOpbeatErrorHandler(appId, organizationId, secretToken string) (Handler, error) {
	if appId == "" {
		return nil, fmt.Errorf("app ID cannot be empty")
	}
	if organizationId == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}
	if secretToken == "" {
		return nil, fmt.Errorf("secret token cannot be empty")
	}

	return &opbeatErrorHandler{
		appId: appId,
		organizationId: organizationId,
		secretToken: secretToken,
	}, nil
}
