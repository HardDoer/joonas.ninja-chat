package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func httpRequest(method string, url string, requestOptions apiRequestOptions, successCallback responseFn) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	var req *http.Request
	var payload *bytes.Buffer
	var err error
	headers := requestOptions.headers

	if len(requestOptions.queryString) > 0 {
		url += requestOptions.queryString
	}
	if requestOptions.payload != nil {
		payload = bytes.NewBuffer(requestOptions.payload)
		req, err = http.NewRequest(method, url, payload)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		log.Print("apiRequest():", err)
		return nil, genericError()
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	apiResponse, err := client.Do(req)
	if err != nil {
		log.Print("apiRequest():", err)
		return nil, genericError()
	}
	defer apiResponse.Body.Close()
	responseBody, err := ioutil.ReadAll(apiResponse.Body)
	if err != nil {
		log.Print("apiRequest():", err)
		return nil, genericError()
	}
	if apiResponse != nil && apiResponse.Status != "200 OK" {
		errorResponse := errors.New("Error response: " + url + " " + apiResponse.Status)
		log.Print("apiRequest():", errorResponse)
		return nil, errorResponse
	}
	if successCallback != nil {
		return successCallback(responseBody), err
	}
	return responseBody, nil
}

func addBasicAuthHeaders(headers map[string]string, token string) {
	headers["Content-Type"] = "application/json"
	headers["Authorization"] =  `Basic `+ token
}

func apiRequest(method string, requestOptions apiRequestOptions, env string, successCallback responseFn) ([]byte, error) {
	addBasicAuthHeaders(requestOptions.headers, base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
	return httpRequest(method, os.Getenv(env), requestOptions, successCallback)
}

func gatewayApiRequest(method string, requestOptions apiRequestOptions, env string, successCallback responseFn) ([]byte, error){
	addBasicAuthHeaders(requestOptions.headers, base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("GATEWAY_KEY"))))
	return httpRequest(method, os.Getenv(env), requestOptions, successCallback)
}

func apiLoginRequest(email string, password string) (res gatewayDTO, err error) {
	var gatewayRes gatewayDTO
	chatloginRequest := chatLogin{Scope: "chat", GrantType: "client_credentials", Email: email, Password: password}
	jsonResponse, _ := json.Marshal(chatloginRequest)
	options := newApiRequestOptions(&apiRequestOptions{payload: jsonResponse})
	body, err := gatewayApiRequest("POST", options, "CHAT_LOGIN_URL", nil)
	if err != nil {
		log.Print("apiLoginRequest():", err)
		return gatewayRes, err
	}
	if err := json.Unmarshal(body, &gatewayRes); err != nil {
		log.Print("apiLoginRequest():", err)
	}
	return gatewayRes, err
}

func validateToken(token string) (validationRes tokenValidationRes, err error) {
	var tokenJson tokenValidationRes
	chatTokenRequest := gatewayDTO{Token: token}
	jsonResponse, _ := json.Marshal(chatTokenRequest)
	options := newApiRequestOptions(&apiRequestOptions{payload: jsonResponse})
	body, err := gatewayApiRequest("POST", options, "CHAT_TOKEN_URL", nil)
	if err != nil {
		log.Print("validateToken():", err)
		return tokenJson, err
	}
	if err := json.Unmarshal(body, &tokenJson); err != nil {
		log.Print("validateToken():", err)
	}
	return tokenJson, err
}
