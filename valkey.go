package eltest

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/redis/go-redis/v9"
)

const (
	// ValkeyRepo is the upstream Valkey image. Valkey is a community
	// fork of Redis kept under the BSD 3-clause license and exposes
	// the same RESP protocol, so applications targeting Redis work
	// unchanged.
	ValkeyRepo = "valkey/valkey"

	Valkey8_0 = "8.0"
	Valkey8_1 = "8.1"
)

// NewValkey bootstraps a Valkey container shared across tests in the
// process (or unique to the test if BootstrapService is used directly).
func NewValkey(t T, tag string) *Valkey {
	v, err := Bootstrap("valkey-"+tag, &Valkey{
		tag: tag,
	})
	Must(t, err, "bootstrap valkey %s", tag)

	return v
}

type Valkey struct {
	tag string
	res *dockertest.Resource
}

// ValkeyEnvironment carries the addresses tests should use to reach
// the Valkey instance, both from the host (Addr) and from another
// container on the eltest network (ContainerAddr).
type ValkeyEnvironment struct {
	Addr          string
	ContainerAddr string
}

func (v *Valkey) Environment() ValkeyEnvironment {
	return ValkeyEnvironment{
		Addr:          v.getAddr(),
		ContainerAddr: v.getContainerAddr(),
	}
}

// Client returns a fresh go-redis client connected to the Valkey
// instance from the host. Tests that need redis interactions should
// call this for each test (or each goroutine) — clients are cheap and
// returning a shared one would let tests step on each other.
func (v *Valkey) Client() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: v.getAddr(),
	})
}

func (v *Valkey) getAddr() string {
	return fmt.Sprintf("localhost:%s", v.res.GetPort("6379/tcp"))
}

func (v *Valkey) getContainerAddr() string {
	hostname := strings.TrimPrefix(v.res.Container.Name, "/")

	return fmt.Sprintf("%s:6379", hostname)
}

func (v *Valkey) SetUp(pool *dockertest.Pool, network *dockertest.Network) error {
	res, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: ValkeyRepo,
		Tag:        v.tag,
		NetworkID:  network.Network.ID,
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		return fmt.Errorf("failed to run valkey container: %w", err)
	}

	v.res = res

	// Cap container lifetime even if in-process cleanup fails.
	_ = res.Expire(3600)

	err = pool.Retry(func() error {
		client := redis.NewClient(&redis.Options{
			Addr: v.getAddr(),
		})

		defer func() {
			_ = client.Close()
		}()

		if err := client.Ping(context.Background()).Err(); err != nil {
			log.Println(err.Error())

			return fmt.Errorf("ping valkey: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to connect to valkey: %w", err)
	}

	return nil
}

func (v *Valkey) Purge(pool *dockertest.Pool) error {
	if v.res == nil {
		return nil
	}

	err := pool.Purge(v.res)
	if err != nil {
		return fmt.Errorf("failed to purge valkey container: %w", err)
	}

	return nil
}
