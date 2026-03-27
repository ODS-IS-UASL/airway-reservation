package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttClient struct {
	client mqtt.Client
	// conf   config.MQTTConfig
	conf *tls.Config
}

func NewMQTTClient() (mqttClient, error) {
	if os.Getenv("USE_MQTT") != "true" {
		fmt.Println("MQTT client creation is disabled (USE_MQTT=false)")
		return mqttClient{}, nil
	}

	clientID := fmt.Sprintf("uasl-service-%d", time.Now().UnixNano())
	cert, err := os.ReadFile(filepath.Join(".", "broker.crt"))
	if err != nil {
		return mqttClient{}, err
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(cert) {
		return mqttClient{}, fmt.Errorf("failed to parse certificate")
	}
	tlsConfig := &tls.Config{
		RootCAs:            certPool,
		InsecureSkipVerify: false,
	}
	opts := mqtt.NewClientOptions()

	opts.AddBroker(fmt.Sprintf("mqtts://%s:8883", os.Getenv("BROKER_ENDPOINT")))
	opts.SetClientID(clientID)
	opts.SetTLSConfig(tlsConfig)
	opts.SetCleanSession(true)
	opts.SetConnectTimeout(10 * time.Second)

	opts.SetKeepAlive(60 * time.Second)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Connection failed: %s\n", token.Error())
		return mqttClient{}, fmt.Errorf("failed to connect to MQTT broker: %s", token.Error())
	}
	if !client.IsConnected() {
		return mqttClient{}, fmt.Errorf("MQTT client is not connected after Connect()")
	}
	fmt.Println("MQTT connection established")
	return mqttClient{
		client: client,
		conf:   tlsConfig,
	}, nil
}

func (m *mqttClient) Publish(topic string, qos byte, retained bool, msg []byte) error {
	if os.Getenv("USE_MQTT") != "true" {
		fmt.Println("MQTT publishing is disabled (USE_MQTT=false)")
		return nil
	}

	token := m.client.Publish(topic, qos, retained, msg)
	token.Wait()
	if token.Error() != nil {
		fmt.Printf("Failed to publish message: %s\n", token.Error())
		return token.Error()
	}
	fmt.Printf("Message published to topic: %s\n", topic)
	return nil
}
