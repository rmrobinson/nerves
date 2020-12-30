package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"

	action "github.com/rmrobinson/google-smart-home-action-go"
	"github.com/rmrobinson/nerves/services/domotics/integrations/googlehome"
	"github.com/rmrobinson/nerves/services/domotics/integrations/googlehome/relay"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/api/homegraph/v1"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func main() {
	var (
		auth0Domain           = flag.String("auth0-domain", "", "The domain that Auth0 users will be coming from")
		letsEncryptHost       = flag.String("letsencrypt-host", "", "The host name that LetsEncrypt will generate the cert for")
		mockAgentUserID       = flag.String("mock-agent-user-id", "", "The HomeGraph account user ID to use for the mock provider")
		credsFile             = flag.String("creds-file", "", "The Google Service Account key file path")
		googleHomeServicePort = flag.Int("google-home-service-port", 20013, "The port the GoogleHome service is listening on")
	)
	flag.Parse()

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Setup our authentication validator
	auth := &auth0Authenticator{
		logger: logger,
		domain: *auth0Domain,
		client: &http.Client{},
		tokens: map[string]string{},
	}

	// Setup Google Assistant info
	ctx := context.Background()
	hgService, err := homegraph.NewService(ctx, option.WithCredentialsFile(*credsFile))
	if err != nil {
		logger.Fatal("err initializing homegraph",
			zap.Error(err),
		)
	}

	relayProvider := relay.NewProviderService(logger)

	googleActionService := action.NewService(logger, auth, relayProvider, hgService)

	relayAPI := relay.NewAPI(logger, relayProvider, googleActionService)

	// Register callback from Google
	http.HandleFunc(action.GoogleFulfillmentPath, googleActionService.GoogleFulfillmentHandler)

	if len(*mockAgentUserID) > 0 {
		mockLights := map[string]MockLightbulb{
			"123": {
				ID:         "123",
				Name:       "test light 1",
				IsOn:       false,
				Brightness: 40,
				Color: struct {
					Hue        float64
					Saturation float64
					Value      float64
				}{
					100,
					100,
					10,
				},
			},
			"456": {
				ID:         "456",
				Name:       "test light 2",
				IsOn:       false,
				Brightness: 40,
				Color: struct {
					Hue        float64
					Saturation float64
					Value      float64
				}{
					100,
					100,
					10,
				},
			},
		}
		mockReceiver := MockReceiver{
			ID:        "789",
			Name:      "test receiver",
			IsOn:      false,
			Volume:    20,
			Muted:     false,
			CurrInput: "input_1",
		}

		mp := NewMockProvider(logger, googleActionService, mockLights, mockReceiver, *mockAgentUserID)
		relayProvider.RegisterProvider(*mockAgentUserID, mp)
	}

	// Setup LetsEncrypt
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(*letsEncryptHost), //Your domain here
		Cache:      autocert.DirCache("certs"),               //Folder for storing certificates
	}

	googleCallbackServer := &http.Server{
		Addr: ":https",
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	logger.Info("listening for google callbacks",
		zap.String("local_addr", "0.0.0.0:443"),
	)

	// Start the HTTP listener used by LetsEncrypt to validate the domain
	go http.ListenAndServe(":http", certManager.HTTPHandler(nil))

	// Start the HTTPS listener used by Google to pass callbacks to the system
	go func() {
		err := googleCallbackServer.ListenAndServeTLS("", "")
		if err != nil {
			logger.Fatal("error listening for google callbacks",
				zap.Error(err),
			)
		}
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *googleHomeServicePort))
	if err != nil {
		logger.Fatal("error initializing grpc listener",
			zap.Error(err),
		)
	}
	defer lis.Close()
	logger.Info("listening for google home service",
		zap.String("local_addr", lis.Addr().String()),
	)

	grpcServer := grpc.NewServer()
	googlehome.RegisterGoogleHomeServiceServer(grpcServer, relayAPI)
	grpcServer.Serve(lis)
}
