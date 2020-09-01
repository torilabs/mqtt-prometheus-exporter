package cmd

import (
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/torilabs/mqtt-prometheus-exporter/mqtt"
	"github.com/torilabs/mqtt-prometheus-exporter/prometheus"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/torilabs/mqtt-prometheus-exporter/config"
	"github.com/torilabs/mqtt-prometheus-exporter/log"
	"github.com/torilabs/mqtt-prometheus-exporter/server"
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

		l, err := mqtt.NewListener(cfg.Mqtt)
		if err != nil {
			return err
		}
		defer l.Close()

		cl := prometheus.NewCollector(cfg.Cache.Timeout, cfg.Metrics)
		for _, m := range cfg.Metrics {
			mh := mqtt.NewMessageHandler(m, cl)
			if err := l.Subscribe(m.MqttTopic, mh); err != nil {
				return err
			}
		}

		prom.MustRegister(cl)
		startAdminServer()

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

func startAdminServer() {
	log.Logger.Infof("Starting admin server on port '%v'.", cfg.Server.Port)

	go func() {
		if err := server.ListenAndServe(cfg.Server); err != nil && err != http.ErrServerClosed {
			log.Logger.With(zap.Error(err)).Fatalf("Failed to start admin server.")
		}
	}()
}
