package polarion_wsdl

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/AVaitkunas/polarion-wsdl/session_ws"
	"io"
	"net/http"
)

type loginWithTokenEnvelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Header  *loginResponseHeader
	Body    loginWithTokenBody
}

type loginResponseHeader struct {
	XMLName   xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`
	SessionID string   `xml:"http://ws.polarion.com/session sessionID"`
}

type loginWithTokenBody struct {
	XMLName        xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	LogInWithToken session_ws.LogInWithToken
}

// raw login using custom requests to get session ID
func loginWithTokenRaw(httpClient *http.Client, sessionEndpoint, username, token string) (string, error) {
	loginRequestEnvelope := loginWithTokenEnvelope{
		Body: loginWithTokenBody{
			LogInWithToken: session_ws.LogInWithToken{
				Username:  username,
				Token:     token,
				Mechanism: "AccessToken",
			},
		},
	}
	envelopeBytes, err := xml.MarshalIndent(loginRequestEnvelope, "", "  ")
	if err != nil {

		return "", fmt.Errorf("failed to marshal login request envelope %v", err)
	}

	responseBodyBytes, err := makeLoginRequest(
		httpClient,
		sessionEndpoint,
		"logInWithToken",
		string(envelopeBytes),
	)

	if err != nil {
		return "", err
	}

	responseEnvelope := loginWithTokenEnvelope{}
	err = xml.Unmarshal(responseBodyBytes, &responseEnvelope)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal login response xml %v", err)
	}

	if responseEnvelope.Header == nil {
		return "", fmt.Errorf("failed to login to Polarion response envelope header is nil")
	}

	return responseEnvelope.Header.SessionID, nil
}

func makeLoginRequest(client *http.Client, url, action, payload string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(payload)))
	if err != nil {
		return []byte{}, fmt.Errorf("failed to create login request object %v", err)
	}
	req.Header.Set("SOAPAction", fmt.Sprintf("urn:%s", action))

	res, err := client.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to make login request %v", err)
	}
	if res.StatusCode != 200 {
		return []byte{}, fmt.Errorf("failed to make login request: response status: %d", res.StatusCode)
	}

	responseBodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read login response body %v", err)
	}
	defer res.Body.Close()

	return responseBodyBytes, nil
}
