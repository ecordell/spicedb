package cmd

import (
	"context"
	"fmt"
	"time"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpczerolog "github.com/grpc-ecosystem/go-grpc-middleware/providers/zerolog/v2"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/authzed/spicedb/internal/auth"
	"github.com/authzed/spicedb/internal/dashboard"
	"github.com/authzed/spicedb/internal/datastore"
	"github.com/authzed/spicedb/internal/dispatch"
	combineddispatch "github.com/authzed/spicedb/internal/dispatch/combined"
	"github.com/authzed/spicedb/internal/gateway"
	"github.com/authzed/spicedb/internal/middleware/servicespecific"
	"github.com/authzed/spicedb/internal/namespace"
	"github.com/authzed/spicedb/internal/services"
	dispatchSvc "github.com/authzed/spicedb/internal/services/dispatch"
	v1alpha1svc "github.com/authzed/spicedb/internal/services/v1alpha1"
	logmw "github.com/authzed/spicedb/pkg/middleware/logging"
	"github.com/authzed/spicedb/pkg/middleware/requestid"
)

type ServerOption func(*ServerConfig)

//go:generate go run github.com/ecordell/optgen -output zz_generated.server_options.go . ServerConfig
type ServerConfig struct {
	// API config
	GRPCServer          GRPCServerConfig
	HTTPGateway         HTTPServerConfig
	PresharedKey        string
	ShutdownGracePeriod time.Duration

	// Datastore
	Datastore DatastoreConfig

	// Namespace cache
	NamespaceCacheExpiration time.Duration

	// Schema options
	SchemaPrefixesRequired bool

	// Dispatch options
	DispatchServer         GRPCServerConfig
	DispatchMaxDepth       uint32
	DispatchUpstreamAddr   string
	DispatchUpstreamCAPath string

	// API Behavior
	DisableV1SchemaAPI bool

	// Additional Services
	DashboardAPI HTTPServerConfig
	MetricsAPI   HTTPServerConfig

	// Middleware
	UnaryMiddleware             []grpc.UnaryServerInterceptor
	StreamingMiddleware         []grpc.StreamServerInterceptor
	DispatchUnaryMiddleware     []grpc.UnaryServerInterceptor
	DispatchStreamingMiddleware []grpc.StreamServerInterceptor
}

// Complete validates the config and fills out defaults.
// if there is no error, a completedServerConfig (with limited options for
// mutation) is returned.
func (c *ServerConfig) Complete() (RunnableServer, error) {
	if len(c.PresharedKey) < 1 {
		return nil, fmt.Errorf("a preshared key must be provided to authenticate API requests")
	}

	ds, err := NewDatastore(c.Datastore.ToOption())
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	nsm, err := namespace.NewCachingNamespaceManager(ds, c.NamespaceCacheExpiration, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace manager: %w", err)
	}

	grpcprom.EnableHandlingTimeHistogram(grpcprom.WithHistogramBuckets(
		[]float64{.006, .010, .018, .024, .032, .042, .056, .075, .100, .178, .316, .562, 1.000},
	))

	if len(c.UnaryMiddleware) == 0 && len(c.StreamingMiddleware) == 0 {
		c.UnaryMiddleware, c.StreamingMiddleware = defaultMiddleware(log.Logger, c.PresharedKey)
	}

	if len(c.DispatchUnaryMiddleware) == 0 && len(c.DispatchStreamingMiddleware) == 0 {
		c.DispatchUnaryMiddleware, c.DispatchStreamingMiddleware = defaultMiddleware(log.Logger, c.PresharedKey)
	}

	dispatcher, err := combineddispatch.NewDispatcher(nsm, ds,
		combineddispatch.UpstreamAddr(c.DispatchUpstreamAddr),
		combineddispatch.UpstreamCAPath(c.DispatchUpstreamCAPath),
		combineddispatch.GrpcPresharedKey(c.PresharedKey),
		combineddispatch.GrpcDialOpts(
			grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"consistent-hashring"}`),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create dispatcher: %w", err)
	}

	cachingClusterDispatch, err := combineddispatch.NewClusterDispatcher(dispatcher, nsm, ds)
	if err != nil {
		return nil, fmt.Errorf("failed to configure cluster dispatch: %w", err)
	}
	dispatchGrpcServer, err := c.DispatchServer.Complete(zerolog.InfoLevel,
		func(server *grpc.Server) {
			dispatchSvc.RegisterGrpcServices(server, cachingClusterDispatch)
		},
		grpc.ChainUnaryInterceptor(c.DispatchUnaryMiddleware...),
		grpc.ChainStreamInterceptor(c.DispatchStreamingMiddleware...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create dispatch gRPC server: %w", err)
	}

	prefixRequiredOption := v1alpha1svc.PrefixRequired
	if !c.SchemaPrefixesRequired {
		prefixRequiredOption = v1alpha1svc.PrefixNotRequired
	}

	v1SchemaServiceOption := services.V1SchemaServiceEnabled
	if !c.DisableV1SchemaAPI {
		v1SchemaServiceOption = services.V1SchemaServiceDisabled
	}

	// delay setting middleware until Run()
	grpcServer, err := c.GRPCServer.Complete(zerolog.InfoLevel,
		func(server *grpc.Server) {
			services.RegisterGrpcServices(
				server,
				ds,
				nsm,
				dispatcher,
				c.DispatchMaxDepth,
				prefixRequiredOption,
				v1SchemaServiceOption,
			)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC server: %w", err)
	}

	// Configure the gateway to serve HTTP
	gatewayHandler, err := gateway.NewHandler(
		context.TODO(),
		c.GRPCServer.Address,
		c.GRPCServer.TLSCertPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rest gateway: %w", err)
	}
	gatewayServer, err := c.HTTPGateway.Complete(zerolog.InfoLevel, gatewayHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rest gateway: %w", err)
	}

	metricsServer, err := c.MetricsAPI.Complete(zerolog.InfoLevel, MetricsHandler())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics server: %w", err)
	}

	dashboardServer, err := c.DashboardAPI.Complete(zerolog.InfoLevel, dashboard.NewHandler(
		c.GRPCServer.Address,
		c.GRPCServer.TLSKeyPath != "" || c.GRPCServer.TLSCertPath != "",
		c.Datastore.Engine,
		ds,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dashboard server: %w", err)
	}

	return &completedServerConfig{
		gRPCServer:          grpcServer.(*completedGRPCServer),
		dispatchGRPCServer:  dispatchGrpcServer.(*completedGRPCServer),
		gatewayServer:       gatewayServer.(*completedHTTPServer),
		metricsServer:       metricsServer.(*completedHTTPServer),
		dashboardServer:     dashboardServer.(*completedHTTPServer),
		ds:                  ds,
		nsm:                 nsm,
		dispatcher:          dispatcher,
		unaryMiddleware:     c.UnaryMiddleware,
		streamingMiddleware: c.StreamingMiddleware,
	}, nil
}

// RunnableServer is a spicedb service set ready to run
type RunnableServer interface {
	Run(ctx context.Context) error
	Dispatcher() dispatch.Dispatcher
	Datastore() datastore.Datastore
	Middleware() ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor)
	SetMiddleware(unaryInterceptors []grpc.UnaryServerInterceptor, streamingInterceptors []grpc.StreamServerInterceptor) RunnableServer
}

// completedServerConfig holds the full configuration to run a spicedb server,
// but is assumed have already been validated via `Complete()` on ServerConfig.
// It offers limited options for mutation before Run() starts the services.
type completedServerConfig struct {
	gRPCServer         *completedGRPCServer
	dispatchGRPCServer *completedGRPCServer
	gatewayServer      *completedHTTPServer
	metricsServer      *completedHTTPServer
	dashboardServer    *completedHTTPServer

	ds                  datastore.Datastore
	dispatcher          dispatch.Dispatcher
	nsm                 namespace.Manager
	unaryMiddleware     []grpc.UnaryServerInterceptor
	streamingMiddleware []grpc.StreamServerInterceptor
}

func (c *completedServerConfig) Middleware() ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	return c.unaryMiddleware, c.streamingMiddleware
}

func (c *completedServerConfig) SetMiddleware(unaryInterceptors []grpc.UnaryServerInterceptor, streamingInterceptors []grpc.StreamServerInterceptor) RunnableServer {
	c.unaryMiddleware = unaryInterceptors
	c.streamingMiddleware = streamingInterceptors
	return c
}

func (c *completedServerConfig) Datastore() datastore.Datastore {
	return c.ds
}

func (c *completedServerConfig) Dispatcher() dispatch.Dispatcher {
	return c.dispatcher
}

func (c *completedServerConfig) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	stopOnCancel := func(stopFn func()) func() error {
		return func() error {
			<-ctx.Done()
			stopFn()
			return nil
		}
	}

	grpcServer := c.gRPCServer.WithOpts(grpc.ChainUnaryInterceptor(c.unaryMiddleware...), grpc.ChainStreamInterceptor(c.streamingMiddleware...))
	g.Go(grpcServer.Listen)
	g.Go(stopOnCancel(grpcServer.GracefulStop))

	g.Go(c.dispatchGRPCServer.Listen)
	g.Go(stopOnCancel(c.dispatchGRPCServer.GracefulStop))

	g.Go(c.gatewayServer.ListenAndServe)
	g.Go(stopOnCancel(c.gatewayServer.Close))

	g.Go(c.metricsServer.ListenAndServe)
	g.Go(stopOnCancel(c.metricsServer.Close))

	g.Go(c.dashboardServer.ListenAndServe)
	g.Go(stopOnCancel(c.dashboardServer.Close))

	if err := g.Wait(); err != nil {
		log.Warn().Err(err).Msg("error shutting down servers")
	}

	return nil
}

func defaultMiddleware(logger zerolog.Logger, presharedKey string) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	return []grpc.UnaryServerInterceptor{
			requestid.UnaryServerInterceptor(requestid.GenerateIfMissing(true)),
			logmw.UnaryServerInterceptor(logmw.ExtractMetadataField("x-request-id", "requestID")),
			grpclog.UnaryServerInterceptor(grpczerolog.InterceptorLogger(logger)),
			otelgrpc.UnaryServerInterceptor(),
			grpcauth.UnaryServerInterceptor(auth.RequirePresharedKey(presharedKey)),
			grpcprom.UnaryServerInterceptor,
			servicespecific.UnaryServerInterceptor,
		}, []grpc.StreamServerInterceptor{
			requestid.StreamServerInterceptor(requestid.GenerateIfMissing(true)),
			logmw.StreamServerInterceptor(logmw.ExtractMetadataField("x-request-id", "requestID")),
			grpclog.StreamServerInterceptor(grpczerolog.InterceptorLogger(logger)),
			otelgrpc.StreamServerInterceptor(),
			grpcauth.StreamServerInterceptor(auth.RequirePresharedKey(presharedKey)),
			grpcprom.StreamServerInterceptor,
			servicespecific.StreamServerInterceptor,
		}
}
