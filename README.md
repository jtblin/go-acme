# go-acme

Add [Let's Encrypt](https://letsencrypt.org/) (ACME) support to generate and renew SSL certificates to go servers 
using the DNS provider challenge so that it can be used for internal servers.

The library is  built upon [lego](https://github.com/xenolf/lego). It will generate the certificates and 
store them in a pluggable storage backend. It will renew the certificates automatically 7 days 
before they expire.

If the certificates are found in the storage backend, they will be reused, which prevents from hitting
[Let’s Encrypt rate limits](https://community.letsencrypt.org/t/rate-limits-for-lets-encrypt/6769) of
20 certificates per domain per week. It is recommended to use a distributed storage backend to avoid
this issue (currently only `s3` is implemented).

For local development, it can generate self signed certificates instead of calling Let's Encrypt.

## Usage

Example with a standard http server:

```
	ACME := &acme.ACME{
		BackendName: "s3",
		Email:       "user@gmail.com",
		DNSProvider: "route53",
		Domain:      &types.Domain{Main: "foo.my-domain.io"},
	}
	tlsConfig := &tls.Config{}
	if err := ACME.CreateConfig(tlsConfig); err != nil {
		panic(err)
	}
	listener, err := tls.Listen("tcp", ":443", tlsConfig)
	if err != nil {
		panic("Listener: " + err.Error())
	}
	
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	
	// To enable http2, we need http.Server to have reference to tlsConfig
	// https://github.com/golang/go/issues/14374
	server := &http.Server{
		Addr:      ":443",
		Handler:   mux,
		TLSConfig: tlsConfig,
	}
	server.Serve(listener)
```

See [examples](examples/) for complete example implementation.

### ACME config

* `BackendName`: the name of the storage backend e.g. fs, s3 (default `fs`), see below for environment variables
* `Domain`: struct containing the main domain name and optional SANs (Subject Alternate Names)
* `CAServer`: optional CA server url (default to `https://acme-v01.api.letsencrypt.org/directory`)
* `DNSProvider`: mandatory DNS provider name e.g. `route53`. 
* `Email`: email address to register the account
* `SelfSigned`: set to true if you want to generate self signed certificates instead of Let's Encrypt ones

## DNS providers

All DNS providers offered by [lego](https://github.com/xenolf/lego) at the time of publishing
are supported. Environment variables need to be set depending on provider as per [lego](https://github.com/xenolf/lego).

## Storage backends

Pluggable storage backends are supported, and only need to implement the [backend.Interface](backend/backend.go).
Currently the following backend are supported:

### fs

This backend stores the account details and certificate on the filesystem. 
The following environment variables can be set:

* `STORAGE_DIR`: set the directory to store the account and certificate information (default to current directory).
The information will be saved to a `domain.name.json` file.

### s3

This backend stores the account details and certificate on the filesystem. 
The following environment variables can to be set:

* `AWS_BUCKET`: set the bucket to store the account and certificate information.
The information will be saved to a `name/domain/cert.json` file e.g. `bucket/io/domain/label/cert.json`.
* `AWS_REGION`: set the region for the bucket.
* `AWS_ENCRYPTION_KEY`: set the encryption key for s3 server side encryption (optional).
* `AWS_ENCRYPTION_ALG`: set the encryption algorithm for s3 server side encryption e.g. `AES256` (optional).

# Disclaimer

This project is in an alpha state, and therefore should be considered as unreliable and the API is likely to 
have breaking changes in the future.

# Credits, reference and similar projects

* [traefik](https://github.com/containous/traefik) is a reverse proxy and load balancer that supports several backends 
e.g. etcd, kubernetes, etc. and allow generating certificates automatically. The `go-acme` library is based on 
`traefik`'s original code.
* [acmewrapper](https://github.com/dkumor/acmewrapper) allows generating certificate using the HTTP/TLS challenge. 
So not appropriate for internal services with no public internet access. Only offers a filesystem storage backend.
* [caddy](https://github.com/mholt/caddy) server is another go reverse proxy with support 
for Let's Encrypt certificates.
* [Generate and Use Free TLS Certificates with Lego](https://blog.gopheracademy.com/advent-2015/generate-free-tls-certs-with-lego/)

# Author

Jerome Touffe-Blin, [@jtblin](https://twitter.com/jtblin), [About me](http://about.me/jtblin)

# License

go-acme is copyright 2015 Jerome Touffe-Blin and contributors. 
It is licensed under the BSD license. See the include LICENSE file for details.
