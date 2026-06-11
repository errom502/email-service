package app

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/errom502/email-service/cmd/config"
	infrastructureNats "github.com/errom502/email-service/internal/infrastructure/nats"
	"github.com/errom502/email-service/internal/repository"
	redisSource "github.com/errom502/email-service/internal/source/redis"
	"github.com/errom502/email-service/internal/source/smtp"
	"github.com/errom502/email-service/internal/tools"
	"github.com/errom502/email-service/internal/usecase"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"
	"github.com/wneessen/go-mail"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Environment string

func (e Environment) String() string {
	return (string)(e)
}

const (
	EnvironmentDevelop Environment = "develop"
	EnvironmentTest    Environment = "test"
	EnvironmentProd    Environment = "prod"
)

var (
	// ErrAppStartup возвращается, если произошла ошибка при старте приложения.
	ErrAppStartup = errors.New("Application startup error")
	// ErrAppShutdownNormal возвращается, если приложение остановлено.
	ErrAppShutdownNormal = errors.New("Application is shutdown")
	// ErrAppShutdownWithError возвращается, если приложение остановлено с ошибкой.
	ErrAppShutdownWithError = errors.New("Application is shutdown with error")
)

type AppInfo struct {
	Name      string
	Instance  Environment
	BuildTime string
	Commit    string
	Release   string
}

var ApplicationInfo *AppInfo

type app struct {
	config            *config.Config
	serviceHTTPServer Server
	redisConn         *redis.Client
	natsConn          *nats.Conn
	natsJetStream     jetstream.JetStream
	smtpConn          *mail.Client
	stop              context.CancelFunc
	logger            tools.Logger
}

func NewApp(cfg *config.Config, appInfo *AppInfo, logger tools.Logger) *app {
	ApplicationInfo = appInfo
	return &app{
		config: cfg,
		logger: logger,
	}
}

func (a *app) Run(c context.Context) error {
	ctx, cancel := context.WithCancel(c)
	a.stop = cancel

	// AfterFunc-функция при завершении контекста для graceful shutdown
	context.AfterFunc(ctx, func() {
		defer func() {
			if e := recover(); e != nil {
				a.logger.Error("panic: context after-func: %w", zap.Error(fmt.Errorf("%s", e)))
			}
		}()
		err := a.GracefulShutdown(ctx)
		if err != nil {
			a.logger.Error("graceful shutdown error", zap.Error(err))
		}
	})

	switch a.config.Environment {
	case "prod":
		ApplicationInfo.Instance = EnvironmentProd
	case "test":
		ApplicationInfo.Instance = EnvironmentTest
	case "develop":
		fallthrough
	default:
		ApplicationInfo.Instance = EnvironmentDevelop
	}

	// Подключение к провайдерам

	smtpClientConn, err := a.connectToSmtpProvider(ctx)
	if err != nil {
		a.logger.Error(
			"app.Run: failed to connect to smtp provider",
			zap.Error(err),
		)
		return ErrAppStartup
	}
	a.smtpConn = smtpClientConn

	// Инициализация необходимых утилит

	hashTool, err := tools.NewHash(a.config.HMACSignature)
	if err != nil {
		a.logger.Error(
			"app.Run: failed to init hash tool",
			zap.Error(err),
		)
		return ErrAppStartup
	}

	linkTool, err := tools.NewLinkTool(a.config.BaseVerifyUrl)
	if err != nil {
		a.logger.Error(
			"app.Run: failed to init link tool",
			zap.Error(err),
		)
		return ErrAppStartup
	}

	messageTool, err := tools.NewMessageTool(a.config.SMTP.MessageTemplate)
	if err != nil {
		a.logger.Error(
			"app.Run: failed to init message tool",
			zap.Error(err),
		)
		return ErrAppStartup
	}

	eventIDTool := tools.NewGenerator()

	// Инициализация подключения к технологиям
	if err := a.initRedis(ctx); err != nil {
		a.logger.Error(
			"app.Run: failed to init redis",
			zap.Error(err),
		)
		return ErrAppStartup
	}

	if err := a.initNATS(); err != nil {
		a.logger.Error(
			"app.Run: failed to init NATS",
			zap.Error(err),
		)
	}

	// Инициализация продюсера для брокера NATS
	natsProducer := infrastructureNats.NewProducer(
		a.natsJetStream,
		a.config.NATS.DeliveryFailedSubject,
		a.logger,
	)

	// Инициализация source

	smtpSource, err := smtp.NewSmtpSource(
		smtpClientConn,
		a.config.SMTP.From,
		a.config.SMTP.Subject,
		a.config.SMTP.SendTimeoutMs,
	)
	if err != nil {
		a.logger.Error(
			"app.Run: failed to init smtp source",
			zap.Error(err),
		)
		return ErrAppStartup
	}

	cacheSource := redisSource.NewRedisCache(
		a.redisConn,
		time.Duration(a.config.Redis.IdempotencyTTL)*time.Second,
	)

	// Инициализация репозиториев

	cacheRepository := repository.NewCacheRepository(cacheSource)
	smtpRepository := repository.NewSmtpRepository(smtpSource)

	// Инициализация usecase

	emailUsecase := usecase.NewEmailUsecase(
		cacheRepository,
		smtpRepository,
		linkTool,
		hashTool,
		messageTool,
		eventIDTool,
		natsProducer,
		a.logger,
	)

	// Инициализация консьюмера брокера nats
	natsConsumer := infrastructureNats.NewNatsConsumer(
		a.natsJetStream,
		jetstream.ConsumerConfig{
			Durable:       a.config.NATS.ConsumerName,
			FilterSubject: a.config.NATS.VerificationCreatedSubject,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       time.Duration(a.config.NATS.AckWaitSec) * time.Second,
			MaxDeliver:    a.config.NATS.MaxDeliver,
		},
		a.config.NATS.VerificationStream,
		emailUsecase,
		time.Duration(a.config.NATS.FetchRetryWait)*time.Second,
		a.logger,
	)

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		defer a.stop()
		return natsConsumer.Run(egCtx)
	})

	if err := eg.Wait(); err != nil {
		a.logger.Error(
			"app.Run: consumer stopped",
			zap.Error(err),
		)
		return ErrAppShutdownWithError
	}

	return ErrAppShutdownNormal
}

func (a *app) initRedis(ctx context.Context) error {
	client := redis.NewClient(
		&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", a.config.Redis.Host, a.config.Redis.Port),
			Username: a.config.Redis.Username,
			Password: a.config.Redis.Password,
			DB:       a.config.Redis.DB,
		},
	)

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("app.initRedis: %w", err)
	}

	a.redisConn = client

	return nil
}

func (a *app) initNATS() error {
	connection, err := nats.Connect(
		fmt.Sprintf("nats://%s:%d", a.config.NATS.Host, a.config.NATS.Port),

		nats.Name(ApplicationInfo.Name),
		nats.MaxReconnects(a.config.NATS.MaxReconnects),
		nats.ReconnectWait(time.Duration(a.config.NATS.ReconnectWait)*time.Millisecond),

		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			a.logger.Warn(
				"nats err disconnected",
				zap.Error(err),
			)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			a.logger.Info(
				"nats reconnected",
				zap.String("url", nc.ConnectedUrl()),
			)
		}),

		nats.ClosedHandler(func(nc *nats.Conn) {
			a.logger.Error("nats connection closed")
		}),
	)

	if err != nil {
		return fmt.Errorf("failed to connect to nats: %w", err)
	}

	js, err := jetstream.New(connection)
	if err != nil {
		return fmt.Errorf("can't create nats jetstream: %w", err)
	}

	a.natsConn = connection
	a.natsJetStream = js
	return nil
}

func (a *app) connectToSmtpProvider(c context.Context) (*mail.Client, error) {
	opts := []mail.Option{
		mail.WithPort(a.config.SMTP.Port),
		mail.WithHELO(ApplicationInfo.Name),
	}

	if a.config.DevMode {
		opts = append(
			opts,
			mail.WithTLSPolicy(mail.NoTLS),
		)
	} else {
		if a.config.SMTP.User == "" {
			return nil, errors.New("smtp user is empty")
		}

		if a.config.SMTP.Password == "" {
			return nil, errors.New("smtp password is empty")
		}
		opts = append(
			opts,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(a.config.SMTP.User),
			mail.WithPassword(a.config.SMTP.Password),
			mail.WithTLSPolicy(mail.TLSOpportunistic),
			mail.WithTLSConfig(&tls.Config{
				MinVersion: tls.VersionTLS12,
				ServerName: a.config.SMTP.Host,
			}),
		)
	}

	client, err := mail.NewClient(
		a.config.SMTP.Host,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("app.connectToSmtpProvider: %w", err)
	}

	// msg := mail.NewMsg()
	// msg.From(a.config.SMTP.From)
	// msg.To(a.config.SMTP.From)
	// msg.Subject("test")
	// msg.SetBodyString(mail.TypeTextPlain, "Test body")

	// ctx, cancel := context.WithTimeout(c, 10*time.Second)
	// defer cancel()

	// err = client.DialAndSendWithContext(ctx, msg)
	// if err != nil {
	// 	return nil, fmt.Errorf("app.connectToSmtpProvider: failed to dial: %w", err)
	// }

	return client, nil
}

func (a *app) startServiceServer(ctx context.Context) error {
	// TODO: Доделать health check, добавить в GracefulShutdown

	// defer func() {
	// 	if e := recover(); e != nil {
	// 		a.logger.Error("panic: service http start", zap.Error(fmt.Errorf("%s", e)))
	// 	}
	// 	serviceHttpBranch.Die()
	// }()
	// a.serviceHTTPServer = service.NewServer(serviceHttpBranch, &service.AppInfo{
	// 	Name:      ApplicationInfo.Name,
	// 	Instance:  ApplicationInfo.Instance.String(),
	// 	BuildTime: ApplicationInfo.BuildTime,
	// 	Commit:    ApplicationInfo.Commit,
	// 	Release:   ApplicationInfo.Release,
	// })
	// if err := a.serviceHTTPServer.Run(
	// 	context.WithValue(ctx, &service.ContextKeyServiceAddr, service.MakeContextStringValue(a.config.ServiceHTTPHost)), //nolint:staticcheck
	// ); err != nil {
	// 	return fmt.Errorf("service http server is shutdown: %w", err)
	// }
	// return nil
	return nil
}

// GracefulShutdown graceful shutdown приложения
func (a *app) GracefulShutdown(c context.Context) (err error) {
	if a.redisConn != nil {
		if redisErr := a.redisConn.Close(); redisErr != nil {
			err = errors.Join(err, fmt.Errorf("can't shutdown redis connection: %w", redisErr))
		}
	}

	if a.natsConn != nil {
		// При закрытии соединения с nats, jetstream закроется, т.к внутри содержится то же соединение
		a.natsConn.Close()
	}

	if a.smtpConn != nil {
		if smtpErr := a.smtpConn.Close(); smtpErr != nil {
			err = errors.Join(err, fmt.Errorf("can't shutdown smtp connection: %w", smtpErr))
		}
	}
	return
}
