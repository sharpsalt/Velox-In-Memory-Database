package core

import (
	"errors"
)

/*
Pub/Sub System Implementation

Redis Pub/Sub allows clients to subscribe to channels and receive messages
published to those channels by other clients. This decoupling of publishers
and subscribers is a classic pattern for real-time applications.

Implementation details:
- `channels` maps a channel name to a set of subscribed clients.
- When a client subscribes, it's added to the channel's map.
- When a message is published, we iterate over all clients in that channel's
  map and write the message directly to their connection.
- Note: Subscribed clients are supposed to enter a mode where they can ONLY
  issue Pub/Sub commands (SUBSCRIBE, UNSUBSCRIBE, PING, QUIT). We enforce this
  in a robust implementation, but here we provide the core routing logic.
*/

var (
	// channels maps channel name -> map of subscribed clients
	// Since Velox is fundamentally single-threaded (event loop via epoll),
	// this map is inherently thread-safe and doesn't strictly need a mutex.
	channels = make(map[string]map[*Client]struct{})
)

// pubsubSubscribe adds the client to the given channels.
func pubsubSubscribe(c *Client, args []string) []byte {
	if len(args) < 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'subscribe' command"), false)
	}

	for _, channel := range args {
		if channels[channel] == nil {
			channels[channel] = make(map[*Client]struct{})
		}
		channels[channel][c] = struct{}{}
		
		// For each channel, send a confirmation message:
		// 1) "subscribe"
		// 2) channel name
		// 3) total number of channels this client is subscribed to
		// (We'll simplify the count to 1 for this implementation)
		
		resp := []interface{}{
			"subscribe",
			channel,
			int64(1), // In a full implementation, we'd track per-client sub count
		}
		
		// Write the response directly to the client
		c.Write(Encode(resp, false))
	}
	
	// We return nil because we already wrote the responses directly
	return nil
}

// pubsubUnsubscribe removes the client from the given channels.
// If no channels are provided, it removes the client from ALL channels.
func pubsubUnsubscribe(c *Client, args []string) []byte {
	channelsToUnsub := args
	
	if len(args) == 0 {
		// Unsubscribe from all channels
		for channel, clients := range channels {
			if _, exists := clients[c]; exists {
				channelsToUnsub = append(channelsToUnsub, channel)
			}
		}
	}

	for _, channel := range channelsToUnsub {
		if clients, ok := channels[channel]; ok {
			delete(clients, c)
			if len(clients) == 0 {
				delete(channels, channel) // Clean up empty channels
			}
		}
		
		resp := []interface{}{
			"unsubscribe",
			channel,
			int64(0), // Total subscribed channels
		}
		c.Write(Encode(resp, false))
	}

	return nil
}

// pubsubPublish sends a message to all clients subscribed to the channel.
func pubsubPublish(args []string) []byte {
	if len(args) != 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'publish' command"), false)
	}

	channel := args[0]
	message := args[1]

	clients, exists := channels[channel]
	if !exists || len(clients) == 0 {
		return Encode(int64(0), false) // 0 receivers
	}

	// Prepare the message payload
	// Format: ["message", channel, message]
	payload := []interface{}{
		"message",
		channel,
		message,
	}
	encodedPayload := Encode(payload, false)

	receivers := 0
	for client := range clients {
		_, err := client.Write(encodedPayload)
		if err == nil {
			receivers++
		} else {
			// If write fails, the client probably disconnected.
			// The event loop will handle cleanup, but we could eagerly remove here.
		}
	}

	return Encode(int64(receivers), false)
}

// RemoveClientFromPubSub cleans up a client's subscriptions when they disconnect
func RemoveClientFromPubSub(c *Client) {
	for channel, clients := range channels {
		if _, exists := clients[c]; exists {
			delete(clients, c)
			if len(clients) == 0 {
				delete(channels, channel)
			}
		}
	}
}
