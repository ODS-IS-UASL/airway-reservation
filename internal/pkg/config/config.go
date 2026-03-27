package config

import (
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServerConfig       ServerConfig
	SyncerServerConfig SyncerServerConfig
	JWTConfig          JWTConfig
	KafkaConfig        KafkaConfig
	PostgresConfig     PostgresConfig
	DynamoConfig       DynamoConfig
	MQTTConfig         MQTTConfig
	S3Config           S3Config
	CognitoConfig      CognitoConfig
}

type ServerConfig struct {
	Environment string `envconfig:"APP_ENV" default:"local"`
	RESTPort    int    `envconfig:"REST_PORT" required:"false" default:"8080"`
	GiGRPCPort  int    `envconfig:"GI_GRPC_PORT" required:"false" default:"8084"`
	OpGRPCPort  int    `envconfig:"OP_GRPC_PORT" required:"false" default:"8085"`
}

type SyncerServerConfig struct {
	Host string `envconfig:"SYNCER_SERVER_HOST" default:"syncer"`
	Port int    `envconfig:"SYNCER_GRPC_PORT" default:"8011"`
}

type JWTConfig struct {
	JWTSignature         string `envconfig:"JWT_SIGNATURE" required:"false" default:""`
	TokenDuration        int    `envconfig:"JWT_TOKEN_DURATION" required:"false" default:"1440"`
	RefreshTokenDuration int    `envconfig:"JWT_REFRESH_TOKEN_DURATION" required:"false" default:"43200"`
}

type PostgresConfig struct {
	DB        string `envconfig:"POSTGRES_DB" required:"false" default:"local_account"`
	User      string `envconfig:"POSTGRES_USER" required:"false" default:"postgres"`
	Password  string `envconfig:"POSTGRES_PASSWORD" required:"false"  default:"passw0rd!"`
	Host      string `envconfig:"POSTGRES_HOST" required:"false" default:"0.0.0.0"`
	Port      int    `envconfig:"POSTGRES_PORT" required:"false" default:"5432"`
	SSLMode   bool   `envconfig:"POSTGRES_SSL_MODE" required:"false" default:"false"`
	DebugMode bool   `envconfig:"POSTGRES_DEBUG_MODE" required:"false" default:"true"`
	TimeZone  string `envconfig:"POSTGRES_TIME_ZONE" required:"false" default:"Asia/Tokyo"`
}

type KafkaConfig struct {
	Brokers      []string `envconfig:"KAFKA_BROKERS" required:"false" default:"0.0.0.0:9092"`
	SASLUser     string   `envconfig:"KAFKA_SASL_USER" required:"false"`
	SASLPassword string   `envconfig:"KAFKA_SASL_PASSWORD" required:"false"`
}

type DynamoConfig struct {
	Region   string `envconfig:"AWS_REGION" required:"false" default:"ap-northeast-1"`
	Endpoint string `envconfig:"DYNAMO_ENDPOINT" required:"false"`
	Timeout  int    `envconfig:"DYNAMO_TIMEOUT" required:"false" default:"10"`
}

type MQTTConfig struct {
	AccessKey string `envconfig:"MQTT_ACCESS_KEY" json:"access_key"`
	SecretKey string `envconfig:"MQTT_SECRET_KEY" json:"secret_key"`
	Region    string `envconfig:"MQTT_REGION" json:"region"`
	Endpoint  string `envconfig:"MQTT_ENDPOINT" json:"end_point"`
	Timeout   int    `envconfig:"DYNAMO_TIMEOUT" required:"false" default:"3"`
}

type S3Config struct {
	Region string `envconfig:"AWS_REGION" required:"false" default:"ap-northeast-1"`
}

type CognitoConfig struct {
	SystemUserPoolID    string `envconfig:"SYSTEM_USER_POOL_ID" required:"false"`
	OperationUserPoolID string `envconfig:"OPERATION_USER_POOL_ID" required:"false"`
}

var c *Config

func Load() *Config {
	var cnf Config
	if c != nil {
		return c
	}
	err := envconfig.Process("", &cnf)
	if err != nil {
		panic(err)
	}
	c = &cnf
	return c
}

func FetchKafkaConfig() (*KafkaConfig, error) {
	var cnf KafkaConfig
	err := envconfig.Process("", &cnf)
	if err != nil {
		return nil, err
	}
	return &cnf, nil
}

func CreateKafkaConfig() *KafkaConfig {
	var cnf KafkaConfig
	cnf.Brokers = strings.Split(os.Getenv("BOOTSTRAP_SERVERS"), ",")
	return &cnf
}

func GetConfig() *Config {
	return c
}

func (c *Config) IsLocal() bool {
	return c.ServerConfig.Environment == "local"
}

func (c *Config) IsDev() bool {
	return c.ServerConfig.Environment == "dev"
}

func (c *Config) IsStage() bool {
	return c.ServerConfig.Environment == "stg"
}

func (c *Config) IsProd() bool {
	return c.ServerConfig.Environment == "prd"
}
