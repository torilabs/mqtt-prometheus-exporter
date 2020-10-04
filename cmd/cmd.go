package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/etherlabsio/healthcheck"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/torilabs/mqtt-prometheus-exporter/config"
	"github.com/torilabs/mqtt-prometheus-exporter/log"
	"github.com/torilabs/mqtt-prometheus-exporter/mqtt"
	"github.com/torilabs/mqtt-prometheus-exporter/prometheus"
	"go.uber.org/zap"
)

var (
	cfgPath string
	cfg     config.Configuration
)

// Execute run root command (main entry-point).
func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:               "mqtt-prometheus-exporter",
	DisableAutoGenTag: true,
	Short:             "MQTT exporter for Prometheus.",
	Long:              "MQTT Prometheus Exporter exports MQTT topics in Prometheus format.",
	SilenceErrors:     true,
	SilenceUsage:      true,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		if err = viper.BindPFlags(cmd.Flags()); err != nil {
			return
		}
		viper.SetConfigFile(cfgPath)

		if cfg, err = config.Parse(); err != nil {
			return err
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := log.Setup(cfg); err != nil {
			return err
		}
		defer log.Logger.Sync()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		l, err := mqtt.NewListener(cfg.MQTT)
		if err != nil {
			return err
		}
		defer l.Close()

		cl := prometheus.NewCollector(cfg.Cache.Expiration, cfg.Metrics)
		for _, m := range cfg.Metrics {
			mh := mqtt.NewMessageHandler(m, cl)
			if err := l.Subscribe(m.MqttTopic, mh); err != nil {
				return err
			}
		}

		if err := prom.Register(cl); err != nil {
			return err
		}
		startServer()

		// wait for program to terminate
		<-sigs

		// shutdown
		log.Logger.Info("Shutting down the service.")

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "./config.yaml", "Path to the config file.")
}

func startServer() {
	log.Logger.Infof("Starting admin server on port '%v'.", cfg.Server.Port)

	go func() {
		http.Handle("/healthcheck", healthcheck.Handler())
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), nil); err != nil && err != http.ErrServerClosed {
			log.Logger.With(zap.Error(err)).Fatalf("Failed to start admin server.")
		}
	}()
}
