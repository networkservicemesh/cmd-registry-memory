// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !windows

// Package main defines a registry-memory application
package main

import (
	"context"
	"crypto/tls"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edwarnicke/grpcfd"

	"github.com/networkservicemesh/sdk/pkg/tools/opentelemetry"
	"github.com/networkservicemesh/sdk/pkg/tools/tracing"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/registry/chains/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/debug"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
)

// Config is configuration for cmd-registry-memory
type Config struct {
	ListenOn              []url.URL     `default:"unix:///listen.on.socket" desc:"url to listen on." split_words:"true"`
	ProxyRegistryURL      url.URL       `desc:"url to the proxy registry that handles this domain" split_words:"true"`
	ExpirePeriod          time.Duration `default:"1s" desc:"period to check expired NSEs" split_words:"true"`
	LogLevel              string        `default:"INFO" desc:"Log level" split_words:"true"`
	OpenTelemetryEndpoint string        `default:"otel-collector.observability.svc.cluster.local:4317" desc:"OpenTelemetry Collector Endpoint"`
}

func main() {
	// Setup context to catch signals
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		// More Linux signals here
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	// Setup logging
	log.EnableTracing(true)
	logrus.SetFormatter(&nested.Formatter{})
	ctx = log.WithLog(ctx, logruslogger.New(ctx, map[string]interface{}{"cmd": os.Args[0]}))

	// Debug self if necessary
	if err := debug.Self(); err != nil {
		log.FromContext(ctx).Infof("%s", err)
	}

	startTime := time.Now()

	// Get config from environment
	config := &Config{}
	if err := envconfig.Usage("registry_memory", config); err != nil {
		logrus.Fatal(err)
	}
	if err := envconfig.Process("registry_memory", config); err != nil {
		logrus.Fatalf("error processing config from env: %+v", err)
	}

	l, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logrus.Fatalf("invalid log level %s", config.LogLevel)
	}
	logrus.SetLevel(l)

	log.FromContext(ctx).Infof("Config: %#v", config)

	// Configure Open Telemetry
	if opentelemetry.IsEnabled() {
		collectorAddress := config.OpenTelemetryEndpoint
		spanExporter := opentelemetry.InitSpanExporter(ctx, collectorAddress)
		metricExporter := opentelemetry.InitMetricExporter(ctx, collectorAddress)
		o := opentelemetry.Init(ctx, spanExporter, metricExporter, "registry-memory")
		defer func() {
			if err = o.Close(); err != nil {
				log.FromContext(ctx).Error(err.Error())
			}
		}()
	}

	// Get a X509Source
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		logrus.Fatalf("error getting x509 source: %+v", err)
	}
	svid, err := source.GetX509SVID()
	if err != nil {
		logrus.Fatalf("error getting x509 svid: %+v", err)
	}
	logrus.Infof("SVID: %q", svid.ID)

	tlsClientConfig := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny())
	tlsClientConfig.MinVersion = tls.VersionTLS12
	tlsServerConfig := tlsconfig.MTLSServerConfig(source, source, tlsconfig.AuthorizeAny())
	tlsServerConfig.MinVersion = tls.VersionTLS12

	credsTLS := credentials.NewTLS(tlsServerConfig)
	// Create GRPC Server and register services
	serverOptions := append(tracing.WithTracing(), grpc.Creds(credsTLS))
	server := grpc.NewServer(serverOptions...)

	clientOptions := append(
		tracing.WithTracingDial(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithTransportCredentials(
			grpcfd.TransportCredentials(
				credentials.NewTLS(tlsClientConfig),
			),
		),
	)
	memory.NewServer(
		ctx,
		memory.WithExpireDuration(time.Minute),
		memory.WithProxyRegistryURL(&config.ProxyRegistryURL),
		memory.WithDialOptions(clientOptions...)).Register(server)

	for i := 0; i < len(config.ListenOn); i++ {
		srvErrCh := grpcutils.ListenAndServe(ctx, &config.ListenOn[i], server)
		exitOnErr(ctx, cancel, srvErrCh)
	}

	log.FromContext(ctx).Infof("Startup completed in %v", time.Since(startTime))
	<-ctx.Done()
}

func exitOnErr(ctx context.Context, cancel context.CancelFunc, errCh <-chan error) {
	// If we already have an error, log it and exit
	select {
	case err := <-errCh:
		log.FromContext(ctx).Fatal(err)
	default:
	}
	// Otherwise wait for an error in the background to log and cancel
	go func(ctx context.Context, errCh <-chan error) {
		err := <-errCh
		log.FromContext(ctx).Error(err)
		cancel()
	}(ctx, errCh)
}
