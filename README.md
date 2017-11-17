# `minio-proxy-go`

A proxy to [minio](https://minio.io/) which bypasses its access tokens.
This is useful when you want to serve Minio content directly without user login.
You can still require users to be logged in and only then show them minio content
via [nginx internal redirects](http://nginx.org/en/docs/http/ngx_http_core_module.html#internal).
