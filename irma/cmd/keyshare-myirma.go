package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/go-chi/chi"
	"github.com/go-errors/errors"
	irma "github.com/privacybydesign/irmago"
	"github.com/privacybydesign/irmago/internal/common"
	"github.com/privacybydesign/irmago/server"
	"github.com/privacybydesign/irmago/server/myirmaserver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var confKeyshareMyirma *myirmaserver.Configuration

var myirmadCmd = &cobra.Command{
	Use:   "myirma",
	Short: "Imra keyshare server myirma component",
	Run: func(command *cobra.Command, args []string) {
		configureMyirmad(command)

		// Determine full listening address.
		fullAddr := fmt.Sprintf("%s:%d", viper.GetString("listen-addr"), viper.GetInt("port"))

		// Load TLS configuration
		TLSConfig, err := kesyharedTLS(
			viper.GetString("tls-cert"),
			viper.GetString("tls-cert-file"),
			viper.GetString("tls-privkey"),
			viper.GetString("tls-privkey-file"))
		if err != nil {
			die("", err)
		}

		// Create main server
		myirmaServer, err := myirmaserver.New(confKeyshareMyirma)
		if err != nil {
			die("", err)
		}

		r := chi.NewRouter()
		r.Mount(viper.GetString("path-prefix"), myirmaServer.Handler())

		serv := &http.Server{
			Addr:      fullAddr,
			Handler:   r,
			TLSConfig: TLSConfig,
		}

		stopped := make(chan struct{})
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

		go func() {
			if TLSConfig != nil {
				err = serv.ListenAndServeTLS("", "")
			} else {
				err = serv.ListenAndServe()
			}
			confKeysharePhone.Logger.Debug("Server stopped")
			stopped <- struct{}{}
		}()

		for {
			select {
			case <-interrupt:
				confKeysharePhone.Logger.Debug("Caught interrupt")
				serv.Shutdown(context.Background())
				myirmaServer.Stop()
				confKeysharePhone.Logger.Debug("Sent stop signal to server")
			case <-stopped:
				confKeysharePhone.Logger.Info("Exiting")
				close(stopped)
				close(interrupt)
				return
			}
		}
	},
}

func init() {
	keyshareRoot.AddCommand(myirmadCmd)

	flags := myirmadCmd.Flags()
	flags.SortFlags = false

	flags.StringP("config", "c", "", "path to configuration file")
	flags.StringP("schemes-path", "s", irma.DefaultSchemesPath(), "path to irma_configuration")
	flags.String("schemes-assets-path", "", "if specified, copy schemes from here into --schemes-path")
	flags.Int("schemes-update", 60, "update IRMA schemes every x minutes (0 to disable)")
	flags.StringP("url", "u", "", "external URL to server to which the IRMA client connects, \":port\" being replaced by --port value")

	flags.IntP("port", "p", 8080, "port at which to listen")
	flags.StringP("listen-addr", "l", "", "address at which to listen (default 0.0.0.0)")
	flags.String("path-prefix", "/", "prefix to listen path")
	flags.Lookup("port").Header = `Server address and port to listen on`

	flags.String("db-type", myirmaserver.DatabaseTypePostgres, "Type of database to connect keyshare server to")
	flags.String("db", "", "Database server connection string")
	flags.Lookup("db-type").Header = `Database configuration`

	flags.StringArray("keyshare-attributes", nil, "Attributes allowed for login to myirma")
	flags.StringArray("email-attributes", nil, "Attributes allowed for adding email addresses")
	flags.Lookup("keyshare-attributes").Header = `Irma session configuration`

	flags.String("email-server", "", "Email server to use for sending email address confirmation emails")
	flags.String("email-hostname", "", "Hostname used in email server tls certificate (leave empty when mail server does not use tls)")
	flags.String("email-username", "", "Username to use when authenticating with email server")
	flags.String("email-password", "", "Password to use when authenticating with email server")
	flags.String("email-from", "", "Email address to use as sender address")
	flags.String("default-language", "en", "Default language, used as fallback when users prefered language is not available")
	flags.StringToString("login-email-subject", nil, "Translated subject lines for the registration email")
	flags.StringToString("login-email-template", nil, "Translated emails for the registration email")
	flags.StringToString("login-url", nil, "Base URL for the email verification link (localized)")
	flags.Lookup("email-server").Header = `Email configuration (leave empty to disable sending emails)`

	flags.String("tls-cert", "", "TLS certificate (chain)")
	flags.String("tls-cert-file", "", "path to TLS certificate (chain)")
	flags.String("tls-privkey", "", "TLS private key")
	flags.String("tls-privkey-file", "", "path to TLS private key")
	flags.Bool("no-tls", false, "Disable TLS")
	flags.Lookup("tls-cert").Header = `TLS configuration (leave empty to disable TLS)`

	flags.CountP("verbose", "v", "verbose (repeatable)")
	flags.BoolP("quiet", "q", false, "quiet")
	flags.Bool("log-json", false, "Log in JSON format")
	flags.Bool("production", false, "Production mode")
	flags.Lookup("verbose").Header = `Other options`
}

func configureMyirmad(cmd *cobra.Command) {
	dashReplacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(dashReplacer)
	viper.SetFileKeyReplacer(dashReplacer)
	viper.SetEnvPrefix("KEYSHARED")
	viper.AutomaticEnv()

	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		die("", err)
	}

	// Locate and read configuration file
	confpath := viper.GetString("config")
	if confpath != "" {
		dir, file := filepath.Dir(confpath), filepath.Base(confpath)
		viper.SetConfigName(strings.TrimSuffix(file, filepath.Ext(file)))
		viper.AddConfigPath(dir)
	} else {
		viper.SetConfigName("myirmad")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/myirmad/")
	}
	err := viper.ReadInConfig() // Hold error checking until we know how much of it to log

	// Create our logger instance
	logger = server.NewLogger(viper.GetInt("verbose"), viper.GetBool("quiet"), viper.GetBool("log-json"))

	// First log output: hello, development or production mode, log level
	mode := "development"
	if viper.GetBool("production") {
		mode = "production"
	}
	logger.WithFields(logrus.Fields{
		"version":   irma.Version,
		"mode":      mode,
		"verbosity": server.Verbosity(viper.GetInt("verbose")),
	}).Info("keyshared running")

	// Now we finally examine and log any error from viper.ReadInConfig()
	if err != nil {
		if _, notfound := err.(viper.ConfigFileNotFoundError); notfound {
			logger.Info("No configuration file found")
		} else {
			die("", errors.WrapPrefix(err, "Failed to unmarshal configuration file at "+viper.ConfigFileUsed(), 0))
		}
	} else {
		logger.Info("Config file: ", viper.ConfigFileUsed())
	}

	// If username/password are specified for the email server, build an authentication object.
	var emailAuth smtp.Auth
	if viper.GetString("email-username") != "" {
		emailAuth = smtp.PlainAuth("", viper.GetString("email-username"), viper.GetString("email-password"), viper.GetString("email-hostname"))
	}

	// And build the configuration
	confKeyshareMyirma = &myirmaserver.Configuration{
		SchemesPath:           viper.GetString("schemes-path"),
		SchemesAssetsPath:     viper.GetString("schemes-assets-path"),
		SchemesUpdateInterval: viper.GetInt("schemes-update"),
		DisableSchemesUpdate:  viper.GetInt("schemes-update") == 0,
		URL:                   string(regexp.MustCompile("(https?://[^/]*):port").ReplaceAll([]byte(viper.GetString("url")), []byte("$1:"+strconv.Itoa(viper.GetInt("port"))))),
		DisableTLS:            viper.GetBool("no-tls"),

		DbType:       myirmaserver.DatabaseType(viper.GetString("db-type")),
		DbConnstring: viper.GetString("db"),

		KeyshareAttributeNames: viper.GetStringSlice("keyshare-attributes"),
		EmailAttributeNames:    viper.GetStringSlice("email-attributes"),

		EmailServer:       viper.GetString("email-server"),
		EmailAuth:         emailAuth,
		EmailFrom:         viper.GetString("email-from"),
		DefaultLanguage:   viper.GetString("default-language"),
		LoginEmailSubject: viper.GetStringMapString("login-email-subject"),
		LoginEmailFiles:   viper.GetStringMapString("login-email-template"),
		LoginEmailBaseURL: viper.GetStringMapString("login-url"),

		Verbose:    viper.GetInt("verbose"),
		Quiet:      viper.GetBool("quiet"),
		LogJSON:    viper.GetBool("log-json"),
		Logger:     logger,
		Production: viper.GetBool("production"),
	}
}

func myirmadTLS(cert, certfile, key, keyfile string) (*tls.Config, error) {
	if cert == "" && certfile == "" && key == "" && keyfile == "" {
		return nil, nil
	}

	var certbts, keybts []byte
	var err error
	if certbts, err = common.ReadKey(cert, certfile); err != nil {
		return nil, err
	}
	if keybts, err = common.ReadKey(key, keyfile); err != nil {
		return nil, err
	}

	cer, err := tls.X509KeyPair(certbts, keybts)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:             []tls.Certificate{cer},
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}, nil
}
