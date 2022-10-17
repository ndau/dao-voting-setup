package configuration

import (
	"context"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/ndau/dao-voting-setup/models"
	configure "github.com/ndau/go-config"
	logger "github.com/ndau/go-logger"
)

const (
	dbURL = "NDAU_CONNECTION_STRING"
)

// LoadConfig ...
func LoadConfig(ctx context.Context, cfg configure.Config, log logger.Logger) (*models.Config, error) {
	var ret models.Config
	log.Info("Get config from local file")
	envCfg := cfg.GetStringMap("env")
	err := mapstructure.Decode(envCfg, &ret)
	if err != nil {
		panic(err)
	}

	if err := loadEnvConfig(cfg.GetStringMap("env"), &ret); err != nil {
		panic(err)
	}
	return &ret, nil
}

func loadEnvConfig(dm map[string]interface{}, cfg *models.Config) error {
	//DB access
	val, ok := dm[dbURL]
	if !ok {
		val, ok = dm[strings.ToLower(dbURL)]
		if !ok {
			return fmt.Errorf("no field '%s' in the secret", dbURL)
		}
	}

	db, ok := val.(string)
	if !ok {
		return fmt.Errorf("field '%s' in the secret is not a string but a '%T'", dbURL, val)
	}
	cfg.ConnectionString = db

	return nil
}
