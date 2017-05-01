package ingress

import (
	"io"
	"log"

	v2 "code.cloudfoundry.org/scalable-syslog/internal/api/loggregator/v2"

	"google.golang.org/grpc"
)

// Connector connects to loggregator egress API
type Connector struct {
	opts     []grpc.DialOption
	balancer *Balancer
}

// NewConnector returns a new Connector
func NewConnector(balancer *Balancer, opts ...grpc.DialOption) *Connector {
	return &Connector{
		balancer: balancer,
		opts:     opts,
	}
}

// Connect connects to a loggregator egress API
func (c *Connector) Connect() (io.Closer, v2.EgressClient, error) {
	hp, err := c.balancer.NextHostPort()
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpc.Dial(hp, c.opts...)
	if err != nil {
		return nil, nil, err
	}

	client := v2.NewEgressClient(conn)
	log.Println("Created new connection to loggregator egress API")

	return conn, client, nil
}
