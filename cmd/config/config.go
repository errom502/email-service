package config

import (
	"fmt"
	"log"

	"github.com/jessevdk/go-flags"
)

type Config struct {
	// LogLevel уровень логирования
	LogLevel string `long:"log-level" description:"Log level: panic, fatal, warn, info, debug" env:"LOG_LEVEL" default:"warn"`
	// AppName наименование сервиса
	AppName string `long:"appname" env:"APP_NAME" default:"email-service"`
	// Environment окружение
	Environment string `env:"ENVIRONMENT" description:"App environment (develop, test, prod)" default:"develop"`
	// DevMode режим отладки
	DevMode bool `long:"dev-mode" env:"DEV_MODE" description:"Developer mode"`
	MaxCpu  int  `long:"max-cpu" env:"MAX_CPU" description:"Max cpu usage (GOMAXPROC)" default:"0"`

	// TODO:
	// ServiceHTTPHost хост-адрес для входящих HTTP-подключений healthz, readyz, info
	//ServiceHTTPHost string `long:"service-host" env:"SERVICE_HOST" description:"Host for service HTTP-server (ex. 127.0.0.1:8080)" default:":8080"`

	// Redis конфигурация для подключения к Redis серверу
	Redis *Redis
	// NATS конфигурация для подключения к NATS серверу
	NATS *NATS
	// SMTP конфигурация для подключения к SMTP серверу
	SMTP *SMTP

	// HMACSignature секрет для HMAC-SHA256.
	// Передаётся как base64-encoded 32-byte key.
	// После декодирования длина должна быть ровно 32 байта.
	HMACSignature string `long:"hmac-signature" env:"HMAC_SIGNATURE" required:"true"`

	// BaseVerifyUrl базовый URL для построения ссылки подтверждения регистрации. Например, https://example.com
	BaseVerifyUrl string `long:"base-verify-url" env:"BASE_VERIFY_URL" required:"true"`
}

// NATS конфигурация для подключения к NATS серверу
type NATS struct {
	// Host адрес NATS сервера
	Host string `long:"nats-host" env:"NATS_HOST" description:"NATS host" required:"true"`
	// Port порт NATS сервера
	Port int `long:"nats-port" env:"NATS_PORT" description:"NATS port" required:"true"`

	// VerificationStream имя стрима для верификаций
	VerificationStream string `long:"nats-stream" env:"NATS_STREAM" default:"VERIFICATION"`
	// ConsumerName имя потребителя для подписки на события
	ConsumerName string `long:"nats-consumer-name" env:"NATS_CONSUMER_NAME" default:"email-service"`
	// AckWaitSec время ожидания подтверждения в секундах
	AckWaitSec int `long:"nats-ack-wait-sec" env:"NATS_ACK_WAIT_SEC" default:"30"`
	// MaxDeliver максимальное количество попыток доставки сообщения
	MaxDeliver int `long:"nats-max-deliver" env:"NATS_MAX_DELIVER" default:"10"`

	// VerificationCreatedSubject тема для подписки на события создания верификации
	VerificationCreatedSubject string `long:"nats-verification-created-subject" env:"NATS_VERIFICATION_CREATED_SUBJECT" description:"NATS verification created subject" default:"verification.email.created"`
	// DeliveryFailedSubject тема для отправки событий о неудачной доставке письма
	DeliveryFailedSubject string `long:"nats-delivery-failed-subject" env:"NATS_DELIVERY_FAILED_SUBJECT" description:"NATS delivery failed subject" default:"verification.email.delivery_failed"`

	// MaxReconnects количество попыток переподключения к NATS серверу
	MaxReconnects int `long:"nats-max-reconnects" env:"NATS_MAX_RECONNECTS" description:"NATS max reconnects" default:"10"`
	// ReconnectWait время ожидания между попытками переподключения к NATS серверу в секундах
	ReconnectWait int `long:"nats-reconnect-wait" env:"NATS_RECONNECT_WAIT" description:"NATS reconnect wait time in seconds" default:"5000"`
	// FetchRetryWait время ожидания в секундах перед повторной попыткой чтения сообщений из JetStream
	FetchRetryWait int `long:"nats-fetch-retry-wait" env:"NATS_FETCH_RETRY_WAIT" default:"10"`
}

// Redis конфигурация для подключения к Redis серверу
type Redis struct {
	// Host адрес Redis сервера
	Host string `long:"redis-host" env:"REDIS_HOST" description:"Redis host" required:"true"`
	// Port порт Redis сервера
	Port int `long:"redis-port" env:"REDIS_PORT" description:"Redis port" required:"true"`
	// Username имя пользователя для подключения к Redis серверу
	Username string `long:"redis-user" env:"REDIS_USER" description:"Redis user" default:""`
	// Password пароль для подключения к Redis серверу
	Password string `long:"redis-pass" env:"REDIS_PASS" description:"Redis password" default:""`
	// DB номер базы данных Redis
	DB int `long:"redis-db" env:"REDIS_DB" description:"Redis database" required:"true"`
	// IdempotencyTTL время жизни идемпотентности в Redis
	IdempotencyTTL int `long:"redis-idempotency-ttl-sec" env:"REDIS_IDEMPOTENCY_TTL_SEC" default:"86400"`
}

// SMTP конфигурация для подключения к SMTP серверу
type SMTP struct {
	// Host адрес SMTP сервера
	Host string `long:"smtp-host" env:"SMTP_HOST"`
	// Port порт SMTP сервера
	Port int `long:"smtp-port" env:"SMTP_PORT" default:"587"`
	// User имя пользователя для подключения к SMTP серверу
	User string `long:"smtp-user" env:"SMTP_USER"`
	// Password пароль для подключения к SMTP серверу
	Password string `long:"smtp-password" env:"SMTP_PASSWORD"`
	// From адрес отправителя писем
	From string `long:"smtp-from" env:"SMTP_FROM"`
	// Subject тема письма
	Subject string `long:"smtp-subject" env:"SMTP_SUBJECT" default:"Подтверждение регистрации"`
	// MessageTemplate шаблон письма. В шаблоне должен быть плейсхолдер {{verification_link}} для ссылки на подтверждение регистрации.
	// Пример:
	/*
		Здравствуйте.
		Для подтверждения регистрации перейдите по ссылке:
		{{verification_link}}
	*/
	MessageTemplate string `long:"smtp-message-template" env:"SMTP_MESSAGE_TEMPLATE" default:"Перейдите по ссылке:<br><a href='http://{{verification_link}}'>http://{{verification_link}}</a>"`
	// SendTimeoutMs время ожидания отправки письма в миллисекундах
	SendTimeoutMs int `long:"smtp-send-timeout-ms" env:"SMTP_SEND_TIMEOUT_MS" default:"5000"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	parser := flags.NewParser(&cfg, flags.Default|flags.IgnoreUnknown)
	_, err := parser.Parse()
	if err != nil {
		parser.WriteHelp(log.Writer())
		return nil, fmt.Errorf("config parse failed: %w", err)
	}

	return &cfg, nil
}
