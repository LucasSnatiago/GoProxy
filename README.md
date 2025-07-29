# GoProxy [![Go](https://github.com/LucasSnatiago/GoProxy/actions/workflows/go.yml/badge.svg)](https://github.com/LucasSnatiago/GoProxy/actions/workflows/go.yml)

GoProxy is a fast, configurable HTTP/HTTPS and SOCKS5 proxy written in Go. It supports Proxy Auto-Configuration (PAC/WPAD), dynamic upstream selection and detailed request logging.

## Features

- **HTTP/HTTPS Proxy** (including `CONNECT` for TLS/TCP tunneling)
- **SOCKS5 Proxy** via [go-socks5](https://github.com/things-go/go-socks5)
- **PAC/WPAD Support** via [gopac](https://github.com/jackwakefield/gopac)
- Honors `PROXY`, `SOCKS5` and `DIRECT` directives in your PAC file
- Per-request logging: method, target host, and chosen upstream proxy
- Efficient bidirectional tunneling with `io.Copy` and zero-copy splice
- Simple CLI flags for configuration
- Easy extension points for HTTP caching, ad-blocking, pprof metrics, etc.

## Requirements

- Go 1.18+
- Network access to your PAC/WPAD URL

## Installation

```bash
git clone https://github.com/LucasSnatiago/GoProxy.git
cd GoProxy
go build -o GoProxy
```

Or just go to the [release section](https://github.com/LucasSnatiago/GoProxy/releases/)

# Usage

```bash
./goproxy
```

# License

MIT License
