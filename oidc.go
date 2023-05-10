package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/jwk"
)

type ActionsOIDCClient struct {
	TokenRequestURL string
	Audience        string
	RequestToken    string
}

type ActionsJWT struct {
	Count       int
	Value       string
	ParsedToken *jwt.Token
}

const jwksURL = `https://token.actions.githubusercontent.com/.well-known/jwks`

func getKey(token *jwt.Token) (interface{}, error) {

	set, err := jwk.Fetch(context.Background(), jwksURL)
	if err != nil {
		return nil, err
	}

	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("expecting JWT header to have string kid")
	}

	key, match := set.LookupKeyID(keyID)
	if match {

		var rawkey interface{}
		if err := key.Raw(&rawkey); err != nil {
			return nil, fmt.Errorf("failed to create public key")
		}
		return rawkey, nil
	}

	return nil, errors.New("no key found in jwksURL")
}

func getActionsJWTAndExp(client *ActionsOIDCClient, isDebugMode bool) (*ActionsJWT, float64, error) {
	actionsJWT, err := client.GetJWT()
	var exp float64
	if err != nil {
		return nil, exp, err
	}

	parser := new(jwt.Parser)
	parser.SkipClaimsValidation = true
	token, err := parser.Parse(actionsJWT.Value, getKey)
	if err != nil {
		return nil, exp, err
	}
	claims := token.Claims.(jwt.MapClaims)
	exp = claims["exp"].(float64)

	actionsJWT.Parse()
	printDebugMessageIfRequired(isDebugMode, "Parsed JWT:%s", actionsJWT.PrettyPrintClaims())

	return actionsJWT, exp, nil
}

func GetEnvironmentVariable(e string) (string, error) {
	value := os.Getenv(e)
	if value == "" {
		return "", fmt.Errorf("missing %s from environment", e)
	}
	return value, nil
}

func NewActionsOIDCClient(tokenURL string, audience string, token string) (*ActionsOIDCClient, error) {
	client := ActionsOIDCClient{
		TokenRequestURL: tokenURL,
		Audience:        audience,
		RequestToken:    token,
	}
	err := client.BuildTokenURL()
	return &client, err
}

func DefaultOIDCClient(audience string) (*ActionsOIDCClient, error) {
	tokenURL, err := GetEnvironmentVariable("ACTIONS_ID_TOKEN_REQUEST_URL")
	if err != nil {
		return nil, err
	}
	token, err := GetEnvironmentVariable("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	if err != nil {
		return nil, err
	}

	client, err := NewActionsOIDCClient(tokenURL, audience, token)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (client *ActionsOIDCClient) BuildTokenURL() error {
	parsed_url, err := url.Parse(client.TokenRequestURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	if client.Audience != "" {
		query := parsed_url.Query()
		query.Set("audience", client.Audience)
		parsed_url.RawQuery = query.Encode()
		client.TokenRequestURL = parsed_url.String()
	}
	return nil
}

func (client *ActionsOIDCClient) GetJWT() (*ActionsJWT, error) {
	request, err := http.NewRequest("GET", client.TokenRequestURL, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", "Bearer "+client.RequestToken)
	var httpClient http.Client
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 from jwt api: %s", http.StatusText((response.StatusCode)))
	}

	rawBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var jwt ActionsJWT
	err = json.Unmarshal(rawBody, &jwt)

	return &jwt, err
}

func (actionsJWT *ActionsJWT) Parse() {
	actionsJWT.ParsedToken, _ = jwt.Parse(actionsJWT.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte{}, nil
	})
}

func (actionsJWT *ActionsJWT) PrettyPrintClaims() string {
	if claims, ok := actionsJWT.ParsedToken.Claims.(jwt.MapClaims); ok {
		jsonClaims, err := json.MarshalIndent(claims, "", "  ")
		if err != nil {
			fmt.Println(fmt.Errorf("%w", err))
		}
		return string(jsonClaims)
	}
	return ""
}
