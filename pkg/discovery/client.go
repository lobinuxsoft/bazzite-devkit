package discovery

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
	"github.com/lobinuxsoft/capydeploy/pkg/protocol"
)

// Client discovers agents on the local network via mDNS.
type Client struct {
	mu       sync.RWMutex
	agents   map[string]*DiscoveredAgent
	eventsCh chan DiscoveryEvent
	timeout  time.Duration
}

// NewClient creates a new mDNS discovery client.
func NewClient() *Client {
	return &Client{
		agents:   make(map[string]*DiscoveredAgent),
		eventsCh: make(chan DiscoveryEvent, 16),
		timeout:  time.Duration(DefaultTTL) * time.Second,
	}
}

// SetTimeout sets the stale agent timeout.
func (c *Client) SetTimeout(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeout = d
}

// Events returns a channel of discovery events.
func (c *Client) Events() <-chan DiscoveryEvent {
	return c.eventsCh
}

// Discover performs a one-time mDNS query and returns discovered agents.
func (c *Client) Discover(ctx context.Context, timeout time.Duration) ([]*DiscoveredAgent, error) {
	entriesCh := make(chan *mdns.ServiceEntry, 16)

	// Start lookup in background
	go func() {
		params := mdns.DefaultParams(ServiceName)
		params.Entries = entriesCh
		params.Timeout = timeout
		params.WantUnicastResponse = true
		_ = mdns.Query(params)
		close(entriesCh)
	}()

	var agents []*DiscoveredAgent

	for {
		select {
		case entry, ok := <-entriesCh:
			if !ok {
				return agents, nil
			}
			if agent := c.processEntry(entry); agent != nil {
				agents = append(agents, agent)
			}
		case <-ctx.Done():
			return agents, ctx.Err()
		}
	}
}

// StartContinuousDiscovery begins continuous agent discovery.
func (c *Client) StartContinuousDiscovery(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial discovery
	c.Discover(ctx, 3*time.Second)

	for {
		select {
		case <-ticker.C:
			c.Discover(ctx, 3*time.Second)
			c.pruneStaleAgents()
		case <-ctx.Done():
			return
		}
	}
}

// processEntry converts an mDNS entry to a DiscoveredAgent.
func (c *Client) processEntry(entry *mdns.ServiceEntry) *DiscoveredAgent {
	if entry == nil {
		return nil
	}

	// Parse TXT records
	info := protocol.AgentInfo{}
	for _, txt := range entry.InfoFields {
		switch {
		case len(txt) > 3 && txt[:3] == "id=":
			info.ID = txt[3:]
		case len(txt) > 5 && txt[:5] == "name=":
			info.Name = txt[5:]
		case len(txt) > 9 && txt[:9] == "platform=":
			info.Platform = txt[9:]
		case len(txt) > 8 && txt[:8] == "version=":
			info.Version = txt[8:]
		}
	}

	// Use instance name as ID if not in TXT
	if info.ID == "" {
		info.ID = entry.Name
	}
	if info.Name == "" {
		info.Name = entry.Host
	}

	// Collect IPs
	var ips []net.IP
	if entry.AddrV4 != nil {
		ips = append(ips, entry.AddrV4)
	}
	if entry.AddrV6 != nil {
		ips = append(ips, entry.AddrV6)
	}

	now := time.Now()
	agent := &DiscoveredAgent{
		Info:         info,
		Host:         entry.Host,
		Port:         entry.Port,
		IPs:          ips,
		DiscoveredAt: now,
		LastSeen:     now,
	}

	// Update or add agent
	c.mu.Lock()
	existing, exists := c.agents[info.ID]
	if exists {
		existing.LastSeen = now
		existing.IPs = ips
		existing.Port = entry.Port
		agent = existing
		c.mu.Unlock()
		c.emitEvent(DiscoveryEvent{Type: EventUpdated, Agent: agent})
	} else {
		c.agents[info.ID] = agent
		c.mu.Unlock()
		c.emitEvent(DiscoveryEvent{Type: EventDiscovered, Agent: agent})
	}

	return agent
}

// pruneStaleAgents removes agents that haven't been seen recently.
func (c *Client) pruneStaleAgents() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, agent := range c.agents {
		if agent.IsStale(c.timeout) {
			delete(c.agents, id)
			c.emitEvent(DiscoveryEvent{Type: EventLost, Agent: agent})
		}
	}
}

// emitEvent sends an event non-blocking.
func (c *Client) emitEvent(event DiscoveryEvent) {
	select {
	case c.eventsCh <- event:
	default:
		// Channel full, skip event
	}
}

// GetAgents returns all currently known agents.
func (c *Client) GetAgents() []*DiscoveredAgent {
	c.mu.RLock()
	defer c.mu.RUnlock()

	agents := make([]*DiscoveredAgent, 0, len(c.agents))
	for _, agent := range c.agents {
		agents = append(agents, agent)
	}
	return agents
}

// GetAgent returns a specific agent by ID.
func (c *Client) GetAgent(id string) *DiscoveredAgent {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.agents[id]
}

// RemoveAgent removes an agent from tracking.
func (c *Client) RemoveAgent(id string) {
	c.mu.Lock()
	agent, exists := c.agents[id]
	if exists {
		delete(c.agents, id)
	}
	c.mu.Unlock()

	if exists {
		c.emitEvent(DiscoveryEvent{Type: EventLost, Agent: agent})
	}
}

// Clear removes all tracked agents.
func (c *Client) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.agents = make(map[string]*DiscoveredAgent)
}

// Close closes the client and its event channel.
func (c *Client) Close() {
	close(c.eventsCh)
}
