package export

import (
	"fmt"
	"openess/internal/collector"
	"openess/internal/log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Config struct {
	Broker   string
	ClientId *string
	User     *string
	Password *string
}

type mqttExporterTask struct {
	options   *mqtt.ClientOptions
	client    *mqtt.Client
	collector *collector.Collector
}

func (task *mqttExporterTask) connect() error {
	mqttClient := mqtt.NewClient(task.options)

	tok := mqttClient.Connect()
	tok.Wait()

	if tok.Error() != nil {
		return tok.Error()
	}

	task.client = &mqttClient

	return nil
}

func (task *mqttExporterTask) publishStatus(connState bool) mqtt.Token {
	log.PrInfo("export:mqtt: publishing connection state: %v\n", connState)

	status := "offline"
	if connState {
		status = "online"
	}

	tok := (*task.client).Publish("openess/status", 0, false, status)
	tok.Wait()

	return tok
}

func (task *mqttExporterTask) eventLoop() {
	c := task.collector.GetStateChan()
	conn := task.collector.GetConnChan()

	for {
		var backoff time.Duration = time.Second * 2

		for {
			err := task.connect()
			if err != nil {
				log.PrError("export:mqtt: failed to connect: %s, will try again in %.d sec\n", err, int(backoff.Seconds()))
				time.Sleep(backoff)
				if backoff.Minutes() < 5 {
					backoff *= 2
				}
				continue
			}

			break
		}

		log.PrInfo("export:mqtt: connected to broker\n")

	publish_loop:
		for {
			var tok mqtt.Token

			select {
			case state := <-c:
			    tok = task.publishStatus(true)

				if tok.Error()!= nil {
					break
				}

				for n, v := range state {
					var valStr string

					if v.LastValue != nil {
						valStr = v.LastValue.ToStringRaw()
					} else {
						valStr = ""
					}

					log.PrInfo("export:mqtt: publishing register: %s = %s\n", n, valStr)

					tok = (*task.client).Publish(fmt.Sprintf("openess/register/%s", n), 0, false, valStr)
					tok.Wait()

					if tok.Error()!= nil {
					    break
					}
				}
			case connState := <-conn:
			    tok = task.publishStatus(connState)
			}

			if tok.Error() != nil {
				(*task.client).Disconnect(500)
				task.client = nil
				break publish_loop
			}
		}
	}
}

func StartMqttExporter(config Config, col *collector.Collector) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Broker)

	if config.ClientId != nil {
		opts.SetClientID(*config.ClientId)
	}
	if config.User != nil {
		opts.SetUsername(*config.User)
	}
	if config.Password != nil {
		opts.SetPassword(*config.Password)
	}

	cli := &mqttExporterTask{
		options:   opts,
		client:    nil,
		collector: col,
	}

	go cli.eventLoop()
}
