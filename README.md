# Load testing tool

Performing requests to HTTP services by different methods with support of POST request payload

## Example

```shell
ltt --url="https://some-host.local" -concurrent=1000 -httpMethod=POST -postData="param1=value&param2=value"
```

## Linux tuning

Increase amount of sockets able to open:

```
sysctl net.ipv4.ip_local_port_range="15000 61000"
sysctl net.ipv4.tcp_fin_timeout=30
```

The `net.ipv4.tcp_fin_timeout` defines the minimum time these sockets will stay in TIME_WAIT state (unusable after being used once).
Now maximum number of outbound sockets a host can create from a particular IP address is (61000-15000)/30=1533

Allow reuse of sockets:

```shell
sysctl net.ipv4.tcp_tw_reuse=1
```

This allows fast cycling of sockets in time_wait state and re-using them.
