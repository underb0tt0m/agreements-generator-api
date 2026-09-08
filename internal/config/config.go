package config

import (
	"errors"
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
	ExecMod    string     `yaml:"execution_mode"`
	GRPCClient gRPCClient `yaml:"grpc_client"`
	Storage    storage    `yaml:"storage"`
	Redis      redis      `yaml:"redis"`
	RabbitMQ   rabbitMQ   `yaml:"rabbit_mq"`
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
	JobTTL   time.Duration `yaml:"job_ttl" env-default:"5m"`
	Type     string        `yaml:"type" env-default:"postgres"`
	Driver   string        `yaml:"driver" env-default:"postgres"`
	Host     string        `yaml:"host" env-default:"localhost"`
	Port     string        `yaml:"port" env-default:"5432"`
	Database string        `yaml:"database" env-default:"docx_generator"`
}

type redis struct {
	Host         string        `yaml:"host" env-default:"localhost"`
	Port         string        `yaml:"port" env-default:"6379"`
	Db           int           `yaml:"db" env-default:"0"`
	JobStatusTTL time.Duration `yaml:"job_status_ttl" env-default:"10m"`
}

type rabbitMQ struct {
	Host     string `yaml:"host" env-default:"localhost"`
	Port     int    `yaml:"port" env-default:"5672"`
	Username string `yaml:"username"`
	Vhost    string `yaml:"vhost"`
	Queue    string `yaml:"queue"`
}

type security struct {
	HashCost         int `yaml:"hash_cost" env-default:"10"`
	MinPasswordLen   int `yaml:"min_password_len" env-default:"8"`
	SecretKey        string
	DBUser           string
	DBPassword       string
	RedisPassword    string
	RabbitMQPassword string
}

type jWT struct {
	JWTSigningMethod SigningMethod `yaml:"jwt_signing_method" env-default:"HS256"`
	TokenTTL         time.Duration `yaml:"token_ttl" env-default:"1h"`
	Prefix           string        `yaml:"prefix" env-default:"Bearer"`
}

func Load() (*Config, error) {
	cfgPath := fetchConfigPath()

	if cfgPath == "" {
		return nil, errors.New("config path is empty")
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return nil, errors.New("config path does not exist")
	}

	var cfg Config
	if err := cleanenv.ReadConfig(cfgPath, &cfg); err != nil {
		return nil, fmt.Errorf("can't read config: %w", err)
	}

	if err := loadEnvVars(
		map[string]*string{
			"JWT_SECRET":        &cfg.Security.SecretKey,
			"DB_USER":           &cfg.Security.DBUser,
			"DB_PASSWORD":       &cfg.Security.DBPassword,
			"REDIS_PASSWORD":    &cfg.Security.RedisPassword,
			"RABBITMQ_PASSWORD": &cfg.Security.RabbitMQPassword,
		},
	); err != nil {
		return nil, err
	}

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

func loadEnvVars(m map[string]*string) error {
	for name, cfgValue := range m {
		envValue, ok := os.LookupEnv(name)
		if !ok {
			return fmt.Errorf("%v is denied", name)
		}
		if envValue == "" {
			return fmt.Errorf("%v is empty", name)
		}
		*cfgValue = envValue
	}
	return nil
}
