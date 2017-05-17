package metricemitter_test

import (
	"net"
	"time"

	v2 "code.cloudfoundry.org/scalable-syslog/internal/api/loggregator/v2"
	"code.cloudfoundry.org/scalable-syslog/internal/metricemitter"
	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Emitter Client", func() {
	It("reconnects if the connection is lost", func() {
		grpcServer := newgRPCServer()

		client, err := metricemitter.NewClient(
			grpcServer.addr,
			metricemitter.WithGRPCDialOptions(grpc.WithInsecure()),
			metricemitter.WithPulseInterval(50*time.Millisecond),
		)
		Expect(err).ToNot(HaveOccurred())

		client.NewCounterMetric("some-name")
		Eventually(grpcServer.senders).Should(HaveLen(1))
		Eventually(func() int {
			return len(grpcServer.envelopes)
		}).Should(BeNumerically(">", 1))

		grpcServer.stop()

		envelopeCount := len(grpcServer.envelopes)
		Consistently(grpcServer.envelopes).Should(HaveLen(envelopeCount))

		grpcServer = newgRPCServerWithAddr(grpcServer.addr)
		defer grpcServer.stop()

		Eventually(func() int {
			return len(grpcServer.envelopes)
		}, 3).Should(BeNumerically(">", envelopeCount))
	})

	It("emits a zero value on an interval", func() {
		grpcServer := newgRPCServer()
		defer grpcServer.stop()

		client, err := metricemitter.NewClient(
			grpcServer.addr,
			metricemitter.WithGRPCDialOptions(grpc.WithInsecure()),
			metricemitter.WithPulseInterval(50*time.Millisecond),
			metricemitter.WithSourceID("a-source"),
		)
		Expect(err).ToNot(HaveOccurred())

		client.NewCounterMetric("some-name")
		Eventually(grpcServer.senders).Should(HaveLen(1))

		var env *v2.Envelope
		Consistently(func() uint64 {
			Eventually(grpcServer.envelopes).Should(Receive(&env))
			Expect(env.SourceId).To(Equal("a-source"))
			Expect(env.Timestamp).To(BeNumerically(">", 0))

			counter := env.GetCounter()
			Expect(counter.Name).To(Equal("some-name"))

			return env.GetCounter().GetDelta()
		}).Should(Equal(uint64(0)))
	})

	It("always combines the tags from the client and the metric", func() {
		grpcServer := newgRPCServer()
		defer grpcServer.stop()

		client, err := metricemitter.NewClient(
			grpcServer.addr,
			metricemitter.WithGRPCDialOptions(grpc.WithInsecure()),
			metricemitter.WithPulseInterval(50*time.Millisecond),
			metricemitter.WithSourceID("a-source"),
			metricemitter.WithOrigin("a-origin"),
			metricemitter.WithDeployment("a-deployment", "a-job", "a-index"),
		)
		Expect(err).ToNot(HaveOccurred())

		client.NewCounterMetric("some-name",
			metricemitter.WithVersion(2, 0),
			metricemitter.WithTags(map[string]string{
				"unicorn": "another-unicorn",
			}),
		)
		Eventually(grpcServer.senders).Should(HaveLen(1))

		var env *v2.Envelope
		text := func(s string) *v2.Value {
			return &v2.Value{Data: &v2.Value_Text{Text: s}}
		}

		Eventually(grpcServer.envelopes).Should(Receive(&env))
		Expect(env.Tags).To(Equal(map[string]*v2.Value{
			//client tags
			"origin":     text("a-origin"),
			"deployment": text("a-deployment"),
			"job":        text("a-job"),
			"index":      text("a-index"),
			//metric tags
			"metric_version": text("2.0"),
			"unicorn":        text("another-unicorn"),
		}))
	})

	Context("with a counter metric", func() {
		It("emits that value, followed by zero values", func() {
			grpcServer := newgRPCServer()
			defer grpcServer.stop()

			client, err := metricemitter.NewClient(
				grpcServer.addr,
				metricemitter.WithGRPCDialOptions(grpc.WithInsecure()),
				metricemitter.WithPulseInterval(50*time.Millisecond),
			)
			Expect(err).ToNot(HaveOccurred())

			metric := client.NewCounterMetric("some-name")
			Eventually(grpcServer.senders).Should(HaveLen(1))

			metric.Increment(5)

			var env *v2.Envelope
			Eventually(func() uint64 {
				Eventually(grpcServer.envelopes).Should(Receive(&env))
				return env.GetCounter().GetDelta()
			}).Should(Equal(uint64(5)))

			Eventually(grpcServer.envelopes).Should(Receive(&env))
			Expect(env.GetCounter().GetDelta()).To(Equal(uint64(0)))
		})
	})

	Context("with a guage metric", func() {
		It("emits the gauge value on interval with tags", func() {
			grpcServer := newgRPCServer()
			defer grpcServer.stop()

			client, err := metricemitter.NewClient(
				grpcServer.addr,
				metricemitter.WithGRPCDialOptions(grpc.WithInsecure()),
				metricemitter.WithPulseInterval(50*time.Millisecond),
			)
			Expect(err).ToNot(HaveOccurred())

			metric := client.NewGaugeMetric(
				"some-name",
				"some-unit",
				metricemitter.WithTags(map[string]string{
					"some-tag": "some-value",
				}),
			)
			Eventually(grpcServer.senders).Should(HaveLen(1))

			metric.Set(5)

			var env *v2.Envelope
			Eventually(func() map[string]*v2.GaugeValue {
				Eventually(grpcServer.envelopes).Should(Receive(&env))
				return env.GetGauge().Metrics
			}).Should(Equal(map[string]*v2.GaugeValue{
				"some-name": &v2.GaugeValue{
					Unit:  "some-unit",
					Value: float64(5),
				},
			}))

			Eventually(grpcServer.envelopes).Should(Receive(&env))
			Expect(env.GetGauge().Metrics).To(Equal(map[string]*v2.GaugeValue{
				"some-name": &v2.GaugeValue{
					Unit:  "some-unit",
					Value: 5.0,
				},
			}))
			Expect(env.Tags).To(Equal(map[string]*v2.Value{
				"some-tag": &v2.Value{
					Data: &v2.Value_Text{
						Text: "some-value",
					},
				},
			}))
		})
	})
})

func newgRPCServer() *SpyIngressServer {
	return newgRPCServerWithAddr("127.0.0.1:0")
}

func newgRPCServerWithAddr(addr string) *SpyIngressServer {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	spyIngressServer := &SpyIngressServer{
		addr:      lis.Addr().String(),
		senders:   make(chan v2.Ingress_SenderServer, 100),
		envelopes: make(chan *v2.Envelope, 100),
		server:    s,
	}

	v2.RegisterIngressServer(s, spyIngressServer)
	go s.Serve(lis)

	return spyIngressServer
}

type SpyIngressServer struct {
	addr      string
	senders   chan v2.Ingress_SenderServer
	server    *grpc.Server
	envelopes chan *v2.Envelope
}

func (s *SpyIngressServer) Sender(sender v2.Ingress_SenderServer) error {
	s.senders <- sender

	for {
		e, err := sender.Recv()
		if err != nil {
			return err
		}
		s.envelopes <- e
	}

	return nil
}

func (s *SpyIngressServer) stop() {
	s.server.Stop()
}