package watcher

import (
	"time"

	"github.com/geodro/lerd/internal/dns"
)

// WatchDNS polls DNS health for the given TLD every interval. When resolution
// is broken it waits for lerd-dns to be ready and re-applies the resolver
// configuration, replicating the DNS repair done by lerd start.
func WatchDNS(interval time.Duration, tld string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		ok, _ := dns.Check(tld)
		if ok {
			continue
		}

		logger.Warn("DNS resolution broken, repairing", "tld", tld)

		if err := dns.WaitReady(10 * time.Second); err != nil {
			logger.Error("lerd-dns not ready", "err", err)
			continue
		}

		if err := dns.ConfigureResolver(); err != nil {
			logger.Error("DNS repair failed", "err", err)
		} else {
			logger.Info("DNS resolution restored", "tld", tld)
		}
	}
}
