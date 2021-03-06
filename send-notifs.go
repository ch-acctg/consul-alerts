package main

import (
	"bytes"

	"encoding/json"
	"os/exec"

	log "github.com/AcalephStorage/consul-alerts/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/AcalephStorage/consul-alerts/notifier"
)

// NotifEngine handles notifications.
//
// To start NotifEngine:
//     notifEngine := startNotifEngine()
//
// Tp stop NotifEngine (for cleanup):
//     notifEngine.stop()
//
type NotifEngine struct {
	inChan    chan notifier.Messages
	closeChan chan struct{}
}

func (n *NotifEngine) start() {
	cleanup := false
	for !cleanup {
		select {
		case messages := <-n.inChan:
			n.sendBuiltin(messages)
			n.sendCustom(messages)
		case <-n.closeChan:
			cleanup = true
		}
	}
}

func (n *NotifEngine) stop() {
	close(n.closeChan)
}

func (n *NotifEngine) queueMessages(messages notifier.Messages) {
	n.inChan <- messages
	log.Println("messages sent for notification")
}

func (n *NotifEngine) sendBuiltin(messages notifier.Messages) {
	for _, n := range builtinNotifiers() {
		n.Notify(messages)
	}
}

func (n *NotifEngine) sendCustom(messages notifier.Messages) {
	for _, notifCmd := range consulClient.CustomNotifiers() {
		data, err := json.Marshal(&messages)
		if err != nil {
			log.Println("Unable to read messages: ", err)
			return
		}

		input := bytes.NewReader(data)
		output := new(bytes.Buffer)
		cmd := exec.Command(notifCmd)
		cmd.Stdin = input
		cmd.Stdout = output
		cmd.Stderr = output

		if err := cmd.Run(); err != nil {
			log.Println("error running notifier: ", err)
		} else {
			log.Println(">>> notification sent to:", notifCmd)
		}
		log.Println(output)
	}
}

func startNotifEngine() *NotifEngine {
	notifEngine := &NotifEngine{
		inChan:    make(chan notifier.Messages),
		closeChan: make(chan struct{}),
	}
	go notifEngine.start()
	return notifEngine
}
