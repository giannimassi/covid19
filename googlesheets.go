package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

// getGoogleSheetsClient returns a google sheet client
func getGoogleSheetsClient() (*sheets.Service, error) {
	usrCfgDir, _ := os.UserConfigDir()
	credentialsFile := filepath.Join(usrCfgDir, "credentials.json")
	b, err := ioutil.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to read client secret file: %w", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		return nil, fmt.Errorf("while parsing client secret file to config: %w", err)
	}
	auth, err := getOAuthClient(config)
	if err != nil {
		return nil, fmt.Errorf("while initializing OAuth client: %w", err)
	}

	client, err := sheets.New(auth)
	if err != nil {
		return nil, fmt.Errorf("while initializing Google Sheet client: %w", err)
	}

	return client, nil
}

// writeToGoogleSheets writes the provided values to the google sheet identified with the provided id
func writeToGoogleSheets(client *sheets.Service, id, sheet string, wrRange string, values [][]interface{}) error {
	vr := sheets.ValueRange{
		Values: values,
	}

	createSheet := func() error {
		req := sheets.Request{
			AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{
					Title: sheet,
				},
			},
		}

		rbb := &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{&req},
		}

		_, err := client.Spreadsheets.BatchUpdate(id, rbb).Context(context.Background()).Do()
		if err != nil {
			return err
		}
		return nil
	}

	for i := 0; i < 3; i++ {
		_, err := client.Spreadsheets.Values.Update(id, wrRange, &vr).ValueInputOption("RAW").Do()
		if err != nil {
			fmt.Println("Attempting to create sheet " + sheet)
			if err := createSheet(); err != nil && i == 2 {
				return fmt.Errorf("Unable to retrieve data from sheet. %w", err)
			}
			continue
		}
		break
	}
	return nil
}

// getOAuthClient retrieves a token, saves the token, then returns the generated client.
func getOAuthClient(config *oauth2.Config) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	usrCfgDir, _ := os.UserConfigDir()
	tokFile := filepath.Join(usrCfgDir, "covid19-sheet-token.json")
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok, err = getTokenFromWeb(config)
		if err != nil {
			return nil, fmt.Errorf("while getting token: %w", err)
		}
		if err := saveToken(tokFile, tok); err != nil {
			return nil, fmt.Errorf("while saving token: %w", err)
		}
	}
	return config.Client(context.Background(), tok), nil
}

// getTokenFromWeb request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("while reading authorization code: %w", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, fmt.Errorf("while retrieving token from web: %w", err)
	}
	return tok, nil
}

// tokenFromFile retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("Unable to cache oauth token: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("while saving token to disk: %w", err)
	}
	return nil
}
