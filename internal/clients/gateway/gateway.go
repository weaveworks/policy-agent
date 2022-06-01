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
	maxConnRetries            = 10
	maxPingRetries            = 3
	backOffTimeout            = 1 * time.Second
	watchdogInterval          = 30 * time.Second
	WebsocketTimeoutHandshake = time.Second
	WebsocketTimeoutRead      = 5 * time.Second
	WebsocketTimeoutWrite     = time.Second
	WebsocketTimeoutReconnect = time.Second
)

// Gateway connects and authorizes to the SaaS gateway
type Gateway struct {
	client           *channel.Client
	accountID        uuid.UUID
	clusterID        uuid.UUID
	secret           []byte
	k8sServerVersion string
	clusterProvider  string
	agentPermissions string
	buildVersion     string
	suspended        bool
	connected        bool
	authorized       bool
	lock             *sync.Mutex
}

// NewGateway returns and configures a gateway instance
func NewGateway(
	gatewayURL url.URL,
	accountID,
	clusterID uuid.UUID,
	secret []byte,
	k8sServerVersion string,
	clusterProvider string,
	agentPermissions string,
	buildVersion string,
) *Gateway {
	options := channel.ChannelOptions{
		ProtoHandshake: WebsocketTimeoutHandshake,
		ProtoRead:      WebsocketTimeoutRead,
		ProtoWrite:     WebsocketTimeoutWrite,
		ProtoReconnect: WebsocketTimeoutReconnect,
	}
	gateway := Gateway{
		accountID:        accountID,
		clusterID:        clusterID,
		secret:           secret,
		k8sServerVersion: k8sServerVersion,
		clusterProvider:  clusterProvider,
		agentPermissions: agentPermissions,
		buildVersion:     buildVersion,
		lock:             &sync.Mutex{},
	}

	gateway.client = channel.NewClient(gatewayURL, options)
	return &gateway
}

// suspend stops attempted connection
func (g *Gateway) suspend() {
	withLock(g.lock, func() {
		g.suspended = true
		g.client.SetHooks(nil, nil)
	})
	logger.Info("cluster is suspended, will not try to connect to the server again")
}

// onConnect triggers authentication and will suspend the worker if it fails
// It is a hook that is called when websocket connection is started
func (g *Gateway) onConnect() error {
	if g.suspended {
		logger.Info("ignore cluster connection because it is marked as suspended")
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
		if err, ok := err.(ConnectionError); ok {
			g.suspend()
			return err
		}
		os.Exit(122)
	}

	withLock(g.lock, func() { g.authorized = true })
	logger.Info("cluster is connected and authorized successfully")

	return nil
}

// onDisconnect called when a connection is disconnected
func (g *Gateway) onDisconnect() {
	logger.Error("connection to server is closed")
	withLock(g.lock, func() {
		g.connected = false
		g.authorized = false
	})
}

// IsActive indicates if the worker is connected and is ready to send data to the SaaS gateway
func (g *Gateway) IsActive() bool {
	return !g.suspended && g.authorized && g.connected
}

// WaitActive waits until the gateway is active and ready to be used
func (g *Gateway) WaitActive(ctx context.Context, wait time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	for {
		if g.IsActive() {
			return true
		}

		if ctx.Err() != nil {
			return false
		}
	}
}

// watchdog sends periodic pings to the Saas gateway as a connection healthcheck
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

			if g.connected != connected {
				withLock(g.lock, func() { g.connected = connected })
			}

			logger.Debugw("wachdog status:", "connected", connected, "authorized", g.authorized)

		case <-ctx.Done():
			return
		}
	}
	logger.Info("terminating watchdog ...")
}

// Start sets up the websocket connection
func (g *Gateway) Start(ctx context.Context) error {
	onConnectHandler := g.onConnect
	onDisconnectHandler := g.onDisconnect
	g.client.SetHooks(&onConnectHandler, &onDisconnectHandler)

	go func() { g.watchdog(ctx) }()

	// @TODO allow channel library to be managed using contexts and refactor for unit testing
	g.client.Listen()
	return nil
}
