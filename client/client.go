package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/rbaylon/srvcman/database"
)

type Token struct {
	Name string
	Jwt  string
}

func GetToken(username string, password string) (*string, error) {
	var (
		api_url = database.GetEnvVariable("API_URL")
	)
	api_auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	url := api_url + "login"
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", api_auth))
	res, err := client.Do(req)
	if err != nil || res.StatusCode != 200 {
		return nil, fmt.Errorf("login failed")
	}
	defer res.Body.Close()
	responseData, ioerr := io.ReadAll(res.Body)
	if ioerr != nil {
		return nil, ioerr
	}
	log.Println("API login successful")
	var t Token
	json.Unmarshal(responseData, &t)
	return &t.Jwt, nil
}

func Rester(module_uri string, method string, data any, r *http.Request) (error, int) {
	var (
		api_url = database.GetEnvVariable("API_URL")
		key     = []byte(database.GetEnvVariable("APP_SESSION_KEY"))
		store   = sessions.NewCookieStore(key)
	)
	session, _ := store.Get(r, database.GetEnvVariable("APP_NAME"))
	client := &http.Client{}
	req, _ := http.NewRequest(method, api_url+module_uri, nil)
	if method == "POST" || method == "PUT" || method == "DELETE" {
		jsondata, jderr := json.Marshal(data)
		if jderr != nil {
			return jderr, 501
		}
		preq, _ := http.NewRequest(method, api_url+module_uri, bytes.NewBuffer(jsondata))
		req = preq
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", session.Values["jwt"]))
	res, rerror := client.Do(req)
	if rerror != nil {
		return rerror, 501
	}
	defer res.Body.Close()
	if res.StatusCode == 200 || res.StatusCode == 201 {
		responseData, ioerr := io.ReadAll(res.Body)
		if ioerr != nil {
			return ioerr, 501
		}
		jerr := json.Unmarshal(responseData, data)
		if jerr != nil {
			return jerr, 501
		}
		return nil, res.StatusCode
	}
	return fmt.Errorf("API error: %s", res.Status), res.StatusCode
}
