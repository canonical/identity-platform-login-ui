package healthcheck

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	"go.opentelemetry.io/otel/codes"
)

type CheckFunction func(context.Context) (bool, error)

type CheckerInterface interface {
	Start()
	Stop()
	Status() bool
}

type Checker struct {
	f      CheckFunction
	ticker *time.Ticker
	status *atomic.Bool

	// goroutine control
	wg sync.WaitGroup

	shutdownCh chan bool

	tracer tracing.TracingInterface
	logger logging.LoggerInterface
}

func (c *Checker) Start() {
	c.wg.Add(1)
	go c.loop()
}

func (c *Checker) Stop() {
	// send shutdown
	c.shutdownCh <- true

	c.wg.Wait()
}

func (c *Checker) Status() bool {
	return c.status.Load()
}

func (c *Checker) set(ctx context.Context, status bool) {
	_, span := c.tracer.Start(context.Background(), "healthcheck.Checker.set")
	defer span.End()

	c.status.Store(status)
	span.SetStatus(codes.Ok, "")
}

func (c *Checker) loop() {
	for {

		select {
		case <-c.shutdownCh:
			_, span := c.tracer.Start(context.Background(), "healthcheck.Checker.loop")
			c.logger.Info("shutting down checker")
			span.SetStatus(codes.Ok, "")
			span.End()
			c.wg.Done()
			return
		case <-c.ticker.C:
			ctx, span := c.tracer.Start(context.Background(), "healthcheck.Checker.loop")
			status, err := c.f(ctx)

			if err != nil {
				c.logger.Error(err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				span.SetStatus(codes.Ok, "")
			}
			c.set(ctx, status)
			span.End()
		}
	}
}

func NewChecker(f CheckFunction, tracer tracing.TracingInterface, logger logging.LoggerInterface) *Checker {
	c := new(Checker)
	c.f = f
	c.shutdownCh = make(chan bool)
	c.ticker = time.NewTicker(10 * time.Second)
	c.status = new(atomic.Bool)

	c.tracer = tracer
	c.logger = logger

	return c
}
