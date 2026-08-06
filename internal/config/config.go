package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type SigningMethod = string

const (
	HS256 = "HS256"
	ES256 = "ES256"
	RS256 = "RS256"
)

type Config struct {
	Env        string     `yaml:"env" env-default:"production"`
	Log        logger     `yaml:"logger"`
	Server     server     `yaml:"server"`
	GRPCClient gRPCClient `yaml:"grpc_client"`
	Storage    storage    `yaml:"storage"`
	JWT        jWT        `yaml:"jwt"`
	Security   security   `yaml:"security"`
}

type logger struct {
	Type  string `yaml:"type" env-default:"zap"`
	Level string `yaml:"level" env-default:"production"`
}

type server struct {
	Port             string        `yaml:"port" env-default:"8080"`
	ShutdownDuration time.Duration `yaml:"shutdown_duration" env-default:"10s"`
}
type gRPCClient struct {
	Host           string        `yaml:"host" env-default:"localhost"`
	Port           string        `yaml:"port" env-default:"50051"`
	JobMaxDuration time.Duration `yaml:"job_max_duration" env-default:"60s"`
}

type storage struct {
	JobTTL time.Duration `yaml:"job_ttl" env-default:"5m"`
}

type security struct {
	HashCost  int `yaml:"hash_cost" env-default:"10"`
	SecretKey string
}

type jWT struct {
	JWTSigningMethod SigningMethod `yaml:"jwt_signing_method" env-default:"HS256"`
	TokenTTL         time.Duration `yaml:"token_ttl" env-default:"1h"`
	Prefix           string        `yaml:"prefix" env-default:"Bearer"`
}

func Load() (*Config, error) {
	cfgPath := fetchConfigPath()

	if cfgPath == "" {
		return nil, fmt.Errorf("config path is empty")
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config path does not exist")
	}

	var cfg Config
	if err := cleanenv.ReadConfig(cfgPath, &cfg); err != nil {
		return nil, fmt.Errorf("can't read config: %w", err)
	}

	secret, ok := os.LookupEnv("JWT_SECRET")
	if !ok {
		return nil, fmt.Errorf("JWT secret string is denied")
	}
	if secret == "" {
		return nil, fmt.Errorf("JWT secret is empty")
	}

	cfg.Security.SecretKey = secret

	return &cfg, nil
}

func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
