package collector

import (
	"errors"
	"openess/internal/client"
	"openess/internal/commands"
	"openess/internal/log"
	"openess/internal/protocol"
	"time"
)

type Config struct {
	Enabled   bool
	Interval  string
	Registers map[string]string
}

type PolledRegister struct {
	Segment   *protocol.Segment
	Register  *protocol.Register
	LastValue *commands.RegValue
}

type PollState = map[string]*PolledRegister

type collectorTask struct {
	pollInterval time.Duration
	client       *client.Client
	state        PollState
	txState      chan PollState
	txConn       chan bool
}

type Collector struct {
	rxState chan PollState
	rxConn  chan bool
}

func StartCollector(client *client.Client, config Config) (*Collector, error) {
	client.WaitConnection()

	descriptor := client.GetDescriptor()

	if descriptor == nil {
		return nil, errors.New("descriptor is not loaded")
	}

	pollInterval, err := time.ParseDuration(config.Interval)
	if err != nil {
		return nil, err
	}

	values := make(map[string]*PolledRegister)
	ch := make(chan map[string]*PolledRegister)
	cch := make(chan bool)

	if !config.Enabled {
		this := Collector{
			rxState: ch,
		}

		log.PrInfo("collector: collector is disabled, doing nothing\n")

		return &this, nil
	}

	for exportId := range config.Registers {
		name := config.Registers[exportId]
		seg, reg := descriptor.FindRegister(name)

		if seg == nil || reg == nil {
			log.PrError("collector: failed to find register %s, skipping from polling\n", name)
			continue
		}

		entry := PolledRegister{
			Segment:   seg,
			Register:  reg,
			LastValue: nil,
		}

		values[exportId] = &entry
	}

	task := collectorTask{
		pollInterval: pollInterval,
		client:       client,
		state:        values,
		txState:      ch,
		txConn:       cch,
	}

	go task.pollLoop()

	collector := Collector{
		rxState: ch,
		rxConn:  cch,
	}

	return &collector, err
}

func (this *Collector) GetStateChan() chan PollState {
	return this.rxState
}

func (this *Collector) GetConnChan() chan bool {
	return this.rxConn
}

func (this *collectorTask) pollLoop() {
	timer := time.NewTimer(this.pollInterval)
	firstPoll := true

	for {
		<-timer.C
		timer.Reset(this.pollInterval)

		var isOffline = !this.client.IsConnected()
		if isOffline {
			this.txConn <- false
		}

		log.PrDebug("collector: waiting for connection\n")
		this.client.WaitConnection()

		if isOffline || firstPoll {
			this.txConn <- true
		}

		if firstPoll {
		    firstPoll = false
		}

		for exportId := range this.state {
			regState := this.state[exportId]
			log.PrDebug("collector: polling register %s\n", exportId)

			cmd := commands.NewRegReadDescr(regState.Segment, regState.Register)

			resp, err := client.SendCommand(this.client, cmd)
			if err != nil {
				log.PrError("collector: failed to read register: %s\n", err)
				continue
			}

			regState.LastValue = &resp.Value
		}

		select {
		case this.txState <- this.state:
			log.PrDebug("collector: sent updated state\n")
		default:
		}
	}
}
