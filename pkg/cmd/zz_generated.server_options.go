// Code generated by github.com/ecordell/optgen. DO NOT EDIT.
package cmd

import (
	grpc "google.golang.org/grpc"
	"time"
)

type ServerConfigOption func(s *ServerConfig)

// NewServerConfigWithOptions creates a new ServerConfig with the passed in options set
func NewServerConfigWithOptions(opts ...ServerConfigOption) *ServerConfig {
	s := &ServerConfig{}
	for _, o := range opts {
		o(s)
	}
	return s
}

// ServerConfigWithOptions configures an existing ServerConfig with the passed in options set
func ServerConfigWithOptions(s *ServerConfig, opts ...ServerConfigOption) *ServerConfig {
	for _, o := range opts {
		o(s)
	}
	return s
}

// WithGRPCServer returns an option that can set GRPCServer on a ServerConfig
func WithGRPCServer(gRPCServer GRPCServerConfig) ServerConfigOption {
	return func(s *ServerConfig) {
		s.GRPCServer = gRPCServer
	}
}

// WithHTTPGateway returns an option that can set HTTPGateway on a ServerConfig
func WithHTTPGateway(hTTPGateway HTTPServerConfig) ServerConfigOption {
	return func(s *ServerConfig) {
		s.HTTPGateway = hTTPGateway
	}
}

// WithPresharedKey returns an option that can set PresharedKey on a ServerConfig
func WithPresharedKey(presharedKey string) ServerConfigOption {
	return func(s *ServerConfig) {
		s.PresharedKey = presharedKey
	}
}

// WithShutdownGracePeriod returns an option that can set ShutdownGracePeriod on a ServerConfig
func WithShutdownGracePeriod(shutdownGracePeriod time.Duration) ServerConfigOption {
	return func(s *ServerConfig) {
		s.ShutdownGracePeriod = shutdownGracePeriod
	}
}

// WithDatastore returns an option that can set Datastore on a ServerConfig
func WithDatastore(datastore DatastoreConfig) ServerConfigOption {
	return func(s *ServerConfig) {
		s.Datastore = datastore
	}
}

// WithReadOnly returns an option that can set ReadOnly on a ServerConfig
func WithReadOnly(readOnly bool) ServerConfigOption {
	return func(s *ServerConfig) {
		s.ReadOnly = readOnly
	}
}

// WithBootstrapFiles returns an option that can append BootstrapFiless to ServerConfig.BootstrapFiles
func WithBootstrapFiles(bootstrapFiles string) ServerConfigOption {
	return func(s *ServerConfig) {
		s.BootstrapFiles = append(s.BootstrapFiles, bootstrapFiles)
	}
}

// SetBootstrapFiles returns an option that can set BootstrapFiles on a ServerConfig
func SetBootstrapFiles(bootstrapFiles []string) ServerConfigOption {
	return func(s *ServerConfig) {
		s.BootstrapFiles = bootstrapFiles
	}
}

// WithBootstrapOverwrite returns an option that can set BootstrapOverwrite on a ServerConfig
func WithBootstrapOverwrite(bootstrapOverwrite bool) ServerConfigOption {
	return func(s *ServerConfig) {
		s.BootstrapOverwrite = bootstrapOverwrite
	}
}

// WithRequestHedgingEnabled returns an option that can set RequestHedgingEnabled on a ServerConfig
func WithRequestHedgingEnabled(requestHedgingEnabled bool) ServerConfigOption {
	return func(s *ServerConfig) {
		s.RequestHedgingEnabled = requestHedgingEnabled
	}
}

// WithRequestHedgingInitialSlowValue returns an option that can set RequestHedgingInitialSlowValue on a ServerConfig
func WithRequestHedgingInitialSlowValue(requestHedgingInitialSlowValue time.Duration) ServerConfigOption {
	return func(s *ServerConfig) {
		s.RequestHedgingInitialSlowValue = requestHedgingInitialSlowValue
	}
}

// WithRequestHedgingMaxRequests returns an option that can set RequestHedgingMaxRequests on a ServerConfig
func WithRequestHedgingMaxRequests(requestHedgingMaxRequests uint64) ServerConfigOption {
	return func(s *ServerConfig) {
		s.RequestHedgingMaxRequests = requestHedgingMaxRequests
	}
}

// WithRequestHedgingQuantile returns an option that can set RequestHedgingQuantile on a ServerConfig
func WithRequestHedgingQuantile(requestHedgingQuantile float64) ServerConfigOption {
	return func(s *ServerConfig) {
		s.RequestHedgingQuantile = requestHedgingQuantile
	}
}

// WithNamespaceCacheExpiration returns an option that can set NamespaceCacheExpiration on a ServerConfig
func WithNamespaceCacheExpiration(namespaceCacheExpiration time.Duration) ServerConfigOption {
	return func(s *ServerConfig) {
		s.NamespaceCacheExpiration = namespaceCacheExpiration
	}
}

// WithSchemaPrefixesRequired returns an option that can set SchemaPrefixesRequired on a ServerConfig
func WithSchemaPrefixesRequired(schemaPrefixesRequired bool) ServerConfigOption {
	return func(s *ServerConfig) {
		s.SchemaPrefixesRequired = schemaPrefixesRequired
	}
}

// WithDispatchServer returns an option that can set DispatchServer on a ServerConfig
func WithDispatchServer(dispatchServer GRPCServerConfig) ServerConfigOption {
	return func(s *ServerConfig) {
		s.DispatchServer = dispatchServer
	}
}

// WithDispatchMaxDepth returns an option that can set DispatchMaxDepth on a ServerConfig
func WithDispatchMaxDepth(dispatchMaxDepth uint32) ServerConfigOption {
	return func(s *ServerConfig) {
		s.DispatchMaxDepth = dispatchMaxDepth
	}
}

// WithDispatchUpstreamAddr returns an option that can set DispatchUpstreamAddr on a ServerConfig
func WithDispatchUpstreamAddr(dispatchUpstreamAddr string) ServerConfigOption {
	return func(s *ServerConfig) {
		s.DispatchUpstreamAddr = dispatchUpstreamAddr
	}
}

// WithDispatchUpstreamCAPath returns an option that can set DispatchUpstreamCAPath on a ServerConfig
func WithDispatchUpstreamCAPath(dispatchUpstreamCAPath string) ServerConfigOption {
	return func(s *ServerConfig) {
		s.DispatchUpstreamCAPath = dispatchUpstreamCAPath
	}
}

// WithDisableV1SchemaAPI returns an option that can set DisableV1SchemaAPI on a ServerConfig
func WithDisableV1SchemaAPI(disableV1SchemaAPI bool) ServerConfigOption {
	return func(s *ServerConfig) {
		s.DisableV1SchemaAPI = disableV1SchemaAPI
	}
}

// WithDashboardAPI returns an option that can set DashboardAPI on a ServerConfig
func WithDashboardAPI(dashboardAPI HTTPServerConfig) ServerConfigOption {
	return func(s *ServerConfig) {
		s.DashboardAPI = dashboardAPI
	}
}

// WithMetricsAPI returns an option that can set MetricsAPI on a ServerConfig
func WithMetricsAPI(metricsAPI HTTPServerConfig) ServerConfigOption {
	return func(s *ServerConfig) {
		s.MetricsAPI = metricsAPI
	}
}

// WithUnaryMiddleware returns an option that can append UnaryMiddlewares to ServerConfig.UnaryMiddleware
func WithUnaryMiddleware(unaryMiddleware grpc.UnaryServerInterceptor) ServerConfigOption {
	return func(s *ServerConfig) {
		s.UnaryMiddleware = append(s.UnaryMiddleware, unaryMiddleware)
	}
}

// SetUnaryMiddleware returns an option that can set UnaryMiddleware on a ServerConfig
func SetUnaryMiddleware(unaryMiddleware []grpc.UnaryServerInterceptor) ServerConfigOption {
	return func(s *ServerConfig) {
		s.UnaryMiddleware = unaryMiddleware
	}
}

// WithStreamingMiddleware returns an option that can append StreamingMiddlewares to ServerConfig.StreamingMiddleware
func WithStreamingMiddleware(streamingMiddleware grpc.StreamServerInterceptor) ServerConfigOption {
	return func(s *ServerConfig) {
		s.StreamingMiddleware = append(s.StreamingMiddleware, streamingMiddleware)
	}
}

// SetStreamingMiddleware returns an option that can set StreamingMiddleware on a ServerConfig
func SetStreamingMiddleware(streamingMiddleware []grpc.StreamServerInterceptor) ServerConfigOption {
	return func(s *ServerConfig) {
		s.StreamingMiddleware = streamingMiddleware
	}
}
