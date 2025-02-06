package env

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type AppEnv = uint8

const (
	Dev AppEnv = iota
	Prod
)

type Env struct {
	DbUrl             string
	DbAuthToken       string
	AppEnv            AppEnv
	OauthClientId     string
	OauthClientSecret string
}

func GetEnv() (env *Env, err error) {
	dbUrl, dbUrlExists := os.LookupEnv("LIB_SQL_DB_URL")
	dbAuthToken, dbAuthTokenExists := os.LookupEnv("LIB_SQL_AUTH_TOKEN")
	appEnvStr, appEnvExists := os.LookupEnv("APP_ENV")
	oauthClientId, oauthClientIdExists := os.LookupEnv("OAUTH_CLIENT_ID")
	oauthClientSecret, oauthClientSecretExists := os.LookupEnv("OAUTH_CLIENT_SECRET")

	appEnv := Dev
	if appEnvExists && appEnvStr == "prod" {
		appEnv = Prod
	}

	if appEnv == Dev {
		if !oauthClientIdExists || !oauthClientSecretExists {
			err := godotenv.Load("./.env")
			if err != nil {
				log.Fatal("env variables not found and .env file not found")
			}

			oauthClientId, oauthClientIdExists = os.LookupEnv("OAUTH_CLIENT_ID")
			oauthClientSecret, oauthClientSecretExists = os.LookupEnv("OAUTH_CLIENT_SECRET")
			if !oauthClientIdExists || !oauthClientSecretExists {
				return nil, errors.New("env variables not found in .env file")
			}
		}
	} else if !dbUrlExists || !dbAuthTokenExists || !oauthClientIdExists || !oauthClientSecretExists {
		err := godotenv.Load("./.env")
		if err != nil {
			log.Fatal("env variables not found and .env file not found")
		}

		dbUrl, dbUrlExists = os.LookupEnv("LIB_SQL_DB_URL")
		dbAuthToken, dbAuthTokenExists = os.LookupEnv("LIB_SQL_AUTH_TOKEN")
		oauthClientId, oauthClientIdExists = os.LookupEnv("OAUTH_CLIENT_ID")
		oauthClientSecret, oauthClientSecretExists = os.LookupEnv("OAUTH_CLIENT_SECRET")
		if !dbUrlExists || !dbAuthTokenExists || !oauthClientIdExists || !oauthClientSecretExists {
			return nil, errors.New("env variables not found in .env file")
		}
	}

	return &Env{
		DbUrl:             dbUrl,
		DbAuthToken:       dbAuthToken,
		AppEnv:            appEnv,
		OauthClientId:     oauthClientId,
		OauthClientSecret: oauthClientSecret,
	}, nil
}
