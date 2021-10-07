package events

import (
	"sync"

	"github.com/cskr/pubsub"

	"github.com/openservicemesh/osm/pkg/announcements"
)

const (
	// Default number of events a subscriber channel will buffer
	defaultAnnouncementChannelSize = 512
)

var (
	// Globally accessible instance, through singleton pattern using getPubSubInstance()
	pubSubInstance *pubsub.PubSub

	// pubSubOnce is used to ensure PubSub object creation happens just once
	pubSubOnce sync.Once
)

// Subscribe is the Subscribe implementation for PubSub
func Subscribe(aTypes ...announcements.Kind) chan interface{} {
	var subTypes []string
	for _, v := range aTypes {
		subTypes = append(subTypes, string(v))
	}

	c := getPubSubInstance().Sub(subTypes...)
	log.Trace().Msgf("subscribing to events %v on channel %v", aTypes, c)
	return c
}

// Publish is the Publish implementation for PubSub
func Publish(message PubSubMessage) {
	log.Trace().Msgf("publishing event kind %s", message.Kind)
	getPubSubInstance().Pub(message, message.Kind.String())
}

// Unsub is the Unsub implementation for PubSub.
// It is synchronized, upon exit the channel is guaranteed to be both
// unsubbed to all topics and closed.
// This is a necessary step to guarantee garbage collection
func Unsub(unsubChan chan interface{}) {
	log.Trace().Msgf("unsubscribing from events on channel %v", unsubChan)

	// implementation has several requirements (including different goroutine context)
	// https://github.com/cskr/pubsub/blob/v1.0.2/pubsub.go#L102

	inst := getPubSubInstance()
	syncCh := make(chan struct{})
	go func() {
		// This will close the channel on the pubsub backend
		// https://github.com/cskr/pubsub/blob/v1.0.2/pubsub.go#L264
		inst.Unsub(unsubChan)

		for range unsubChan {
			// Drain channel, read til close
		}
		syncCh <- struct{}{}
	}()

	<-syncCh
}

// getPubSubInstance returns a unique, global scope PubSub interface instance
// Note that spawning the instance is not thread-safe. First call should happen on
// a single-routine context to avoid races.
func getPubSubInstance() *pubsub.PubSub {
	pubSubOnce.Do(func() {
		pubSubInstance = pubsub.New(defaultAnnouncementChannelSize)
	})
	return pubSubInstance
}