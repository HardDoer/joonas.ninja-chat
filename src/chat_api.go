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
)

func apiRequest(method string, requestOptions apiRequestOptions, env string, successCallback responseFn, expectedErrorCallback responseFn) ([]byte, error) {
	client := &http.Client{}
	url := os.Getenv(env)
	var req *http.Request
	var payload *bytes.Buffer
	var err error

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
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", `Basic `+
		base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
	apiResponse, err := client.Do(req)
	if err != nil {
		log.Print("apiRequest():", err)
		return nil, err
	}
	defer apiResponse.Body.Close()
	responseBody, err := ioutil.ReadAll(apiResponse.Body)
	if err != nil {
		log.Print("apiRequest():", err)
		return nil, err
	}
	if apiResponse != nil && apiResponse.Status != "200 OK" {
		errorResponse := errors.New("Error response: " + url + " " + apiResponse.Status)
		log.Print("apiRequest():", errorResponse)
		if expectedErrorCallback != nil {
			return expectedErrorCallback(responseBody), errorResponse
		}
		return nil, errorResponse
	}
	if successCallback != nil {
		return successCallback(responseBody), err
	}
	return responseBody, nil
}

func apiLoginRequest(email string, password string) (res gatewayDTO, err error) {
	var gatewayRes gatewayDTO
	chatloginRequest := chatLogin{Scope: "chat", GrantType: "client_credentials", Email: email, Password: password}
	jsonResponse, err := json.Marshal(chatloginRequest)
	if err != nil {
		log.Print("apiLoginRequest():", err)
		return gatewayRes, err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", os.Getenv("CHAT_LOGIN_URL"), bytes.NewBuffer(jsonResponse))
	if err != nil {
		log.Print("apiLoginRequest():", err)
		return gatewayRes, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", `Basic `+
		base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("GATEWAY_KEY"))))
	loginResponse, err := client.Do(req)
	if err != nil {
		log.Print("apiLoginRequest():", err)
		return gatewayRes, err
	}
	if loginResponse != nil && loginResponse.Status != "200 OK" {
		log.Print("apiLoginRequest():", "Error response "+loginResponse.Status)
		return gatewayRes, errors.New("Error response " + loginResponse.Status)
	}
	defer loginResponse.Body.Close()
	body, err := ioutil.ReadAll(loginResponse.Body)
	if err != nil {
		log.Print("apiLoginRequest():", err)
		return gatewayRes, err
	}
	err = json.Unmarshal(body, &gatewayRes)
	if err != nil {
		log.Print("apiLoginRequest():", err)
		return gatewayRes, err
	}
	return gatewayRes, nil
}

func validateToken(token string) (validationRes tokenValidationRes, err error) {
	chatTokenRequest := gatewayDTO{Token: token}
	jsonResponse, err := json.Marshal(chatTokenRequest)
	var tokenJson tokenValidationRes

	if err != nil {
		log.Print("validateToken():", err)
		return tokenJson, err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", os.Getenv("CHAT_TOKEN_URL"), bytes.NewBuffer(jsonResponse))
	if err != nil {
		log.Print("validateToken():", err)
		return tokenJson, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", `Basic `+
		base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("GATEWAY_KEY"))))
	tokenResponse, err := client.Do(req)
	if err != nil {
		log.Print("validateToken():", err)
		return tokenJson, err
	}
	if tokenResponse != nil && tokenResponse.Status != "200 OK" {
		log.Print("validateToken():", "Error response "+tokenResponse.Status)
		return tokenJson, errors.New("Error response " + tokenResponse.Status)
	}
	defer tokenResponse.Body.Close()
	body, err := ioutil.ReadAll(tokenResponse.Body)
	if err != nil {
		log.Print("getChatHistory():", err)
		return tokenJson, err
	}
	err = json.Unmarshal(body, &tokenJson)
	if err != nil {
		log.Print("getChatHistory():", err)
		return tokenJson, err
	}
	return tokenJson, nil
}
