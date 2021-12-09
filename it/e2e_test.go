// +build integration

package it_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/suite"
	"github.com/torilabs/mqtt-prometheus-exporter/cmd"
)

type e2eTestSuite struct {
	suite.Suite
	port     int
	mqttPort int
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, &e2eTestSuite{
		port:     8079,
		mqttPort: 1883,
	})
}

func (s *e2eTestSuite) SetupSuite() {
	os.Args = append(os.Args, "--config=./it-config.yaml")

	go func() {
		if err := cmd.Execute(); err != nil {
			s.Fail("application start", err)
		}
	}()
	time.Sleep(4 * time.Second)
}

func (s *e2eTestSuite) Test_EndToEnd_Healthcheck() {
	healthcheckBody := s.httpResponseBody("healthcheck")

	s.Equal(`{"status":"OK"}`, healthcheckBody)
}

func (s *e2eTestSuite) Test_EndToEnd_Metrics() {
	opts := pahomqtt.NewClientOptions()
	opts.SetClientID(fmt.Sprintf("%s%d", "e2e-test-", rand.Int31()))
	opts.AddBroker(fmt.Sprintf("localhost:%d", s.mqttPort))
	mqttClient := pahomqtt.NewClient(opts)
	token := mqttClient.Connect()
	if ok := token.WaitTimeout(opts.PingTimeout); !ok {
		s.Fail("MQTT connection timed out", opts.PingTimeout)
	}
	if !mqttClient.IsConnected() {
		s.Fail("MQTT connection unsuccessful to brokers", opts.Servers)
	}
	defer mqttClient.Disconnect(0)

	mqttClient.Publish("/home/owen/memory", 1, true, "13")
	time.Sleep(time.Second)

	metricsBody := s.httpResponseBody("metrics")

	s.Contains(metricsBody, `# HELP iot_memory free memory of a device`)
	s.Contains(metricsBody, `# TYPE iot_memory gauge`)
	s.Contains(metricsBody, `iot_memory{device="owen",device2="home",mylabel="label value",topic="/home/owen/memory"} 13`)
}

func (s *e2eTestSuite) httpResponseBody(path string) string {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/%s", s.port, path), strings.NewReader(""))
	s.NoError(err)

	client := http.Client{}
	response, err := client.Do(req)
	s.NoError(err)
	s.Equal(http.StatusOK, response.StatusCode)

	byteBody, err := ioutil.ReadAll(response.Body)
	s.NoError(err)
	response.Body.Close()

	return strings.Trim(string(byteBody), "\n")
}
