package main

import (
	"openess/internal/client"
	"openess/internal/collector"
	"openess/internal/export"
	"openess/internal/log"
	"os"
)

func BackgroundMain(args Args) {
	config, err := LoadConfig(args.ConfPath)
	if err != nil {
		log.PrError("openess: failed to read config %s: %s", args.ConfPath, err)
		os.Exit(1)
	}

	clientConfig := client.Config{
		DeviceAddr: config.DeviceAddr,
		LocalPort:  config.BindPort,
		ProtoPath:  config.ProtoPath,
		Protocol:   config.Protocol,
	}

    if args.DeviceAddr != nil {
	    clientConfig.DeviceAddr = *args.DeviceAddr
	}

	log.Init(args.LogLevel)

	var cli = client.StartClient(clientConfig)
	cli.WaitConnection()

	collector, err := collector.StartCollector(cli, config.Collector)
	if err != nil {
		log.PrError("openess: failed to init collector: %s\n", err)
		os.Exit(1)
	}

	export.StartMqttExporter(config.Export, collector)

	select{}
}
