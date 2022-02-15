package gateway

import (
	"context"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/MagalixTechnologies/channel"
	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/uuid-go"
)

const (
	maxConnRetries   = 10
	maxPingRetries   = 3
	backOffTimeout   = 1 * time.Second
	watchdogInterval = 3 * time.Second
)

type Gateway struct {
	options    channel.ChannelOptions
	client     *channel.Client
	accountID  uuid.UUID
	clusterID  uuid.UUID
	secret     []byte
	suspended  bool
	connected  bool
	authorized bool
	lock       *sync.Mutex
}

func New(gatewayURL url.URL, accountID, clusterID uuid.UUID, secret []byte) *Gateway {
	options := channel.ChannelOptions{
		ProtoHandshake: time.Second,
		ProtoWrite:     time.Second,
		ProtoRead:      2 * time.Second,
		ProtoReconnect: time.Second,
	}

	gateway := Gateway{
		options:   options,
		accountID: accountID,
		clusterID: clusterID,
		secret:    secret,
		lock:      &sync.Mutex{},
	}

	gateway.client = channel.NewClient(gatewayURL, options)
	return &gateway
}

func (g *Gateway) suspend() {
	withLock(g.lock, func() {
		g.suspended = true
		g.client.SetHooks(nil, nil)
	})
	logger.Info("cluster is suspended, will not try to connect to the server again")
}

func (g *Gateway) onConnect() error {
	if g.suspended {
		logger.Error("ignore cluster connection because it is marked as suspended")
		return nil
	}

	withLock(g.lock, func() { g.connected = true })
	logger.Info("connection to server is established")

	err := withBackOff(maxConnRetries, backOffTimeout, func() error {
		err := g.hello()
		if err != nil {
			logger.Errorw("failed to say hello to server", "error", err)
			return err
		}

		err = g.authorize()
		if err != nil {
			logger.Errorw("failed to authorize cluster", "error", err)
			return err
		}

		return nil
	})

	if err != nil {
		if err, ok := err.(FatalError); ok {
			g.suspend()
			return err
		}
		os.Exit(122)
	}

	withLock(g.lock, func() { g.authorized = true })
	logger.Info("cluster is connected and authorized successfully")

	return nil
}

func (g *Gateway) onDisconnect() {
	logger.Error("connection to server is closed")
	withLock(g.lock, func() {
		g.connected = false
		g.authorized = false
	})
}

func (g *Gateway) IsActive() bool {
	return !g.suspended && g.authorized && g.connected
}

func (g *Gateway) watchdog(ctx context.Context) {
	ticker := time.NewTicker(watchdogInterval)
	for !g.suspended {
		select {
		case <-ticker.C:
			var connected bool
			if g.client.IsConnected() && g.authorized {
				if err := withBackOff(maxPingRetries, backOffTimeout, g.ping); err == nil {
					connected = true
				}
			}
			withLock(g.lock, func() { g.connected = connected })
			logger.Infow("wachdog status:", "connected", connected, "authorized", g.authorized)

		case <-ctx.Done():
			break
		}
	}
	logger.Info("terminating watchdog ...")
}

func (g *Gateway) Start(ctx context.Context) {
	onConnectHandler := g.onConnect
	onDisconnectHandler := g.onDisconnect
	g.client.SetHooks(&onConnectHandler, &onDisconnectHandler)

	go func() { g.watchdog(ctx) }()

	g.client.Listen()
}
