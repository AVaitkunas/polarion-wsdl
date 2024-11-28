package polarion_wsdl

import (
	"github.com/AVaitkunas/polarion-wsdl/session_ws"
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// 3 struct above is used for loginRaw only

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
		log.Printf("error marshaling login request envelope %v", err)
		return "", err
	}

	responseBodyBytes := makeLoginRequest(
		httpClient,
		sessionEndpoint,
		"logInWithToken",
		string(envelopeBytes),
	)

	responseEnvelope := loginWithTokenEnvelope{}
	err = xml.Unmarshal(responseBodyBytes, &responseEnvelope)
	if err != nil {
		log.Printf("failed to unmarshal response xml %v", err)
		return "", err
	}

	return responseEnvelope.Header.SessionID, nil
}

func makeLoginRequest(client *http.Client, url, action, payload string) []byte {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(payload)))
	if err != nil {
		log.Printf("error on creating request object %v", err.Error())
		return []byte{}
	}
	req.Header.Set("SOAPAction", fmt.Sprintf("urn:%s", action))

	res, err := client.Do(req)
	if err != nil {
		log.Printf("error on dispatching request %v", err.Error())
		return []byte{}
	}

	responseBodyBytes, _ := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	return responseBodyBytes
}
