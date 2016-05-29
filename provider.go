package acme

import (
	"github.com/xenolf/lego/acme"
	"github.com/xenolf/lego/providers/dns/cloudflare"
	"github.com/xenolf/lego/providers/dns/digitalocean"
	"github.com/xenolf/lego/providers/dns/dnsimple"
	"github.com/xenolf/lego/providers/dns/dyn"
	"github.com/xenolf/lego/providers/dns/gandi"
	"github.com/xenolf/lego/providers/dns/googlecloud"
	"github.com/xenolf/lego/providers/dns/namecheap"
	"github.com/xenolf/lego/providers/dns/rfc2136"
	"github.com/xenolf/lego/providers/dns/route53"
	"github.com/xenolf/lego/providers/dns/vultr"
)

func newDNSProvider(dns string) (acme.ChallengeProvider, error) {
	switch dns {
	case "cloudflare":
		return cloudflare.NewDNSProvider()
	case "digitalocean":
		return digitalocean.NewDNSProvider()
	case "dnsimple":
		return dnsimple.NewDNSProvider()
	case "dyn":
		return dyn.NewDNSProvider()
	case "gandi":
		return gandi.NewDNSProvider()
	case "gcloud":
		return googlecloud.NewDNSProvider()
	case "manual":
		return acme.NewDNSProviderManual()
	case "namecheap":
		return namecheap.NewDNSProvider()
	case "route53":
		return route53.NewDNSProvider()
	case "rfc2136":
		return rfc2136.NewDNSProvider()
	case "vultr":
		return vultr.NewDNSProvider()
	default:
		panic("Unknown dns provider " + dns)
	}
}
