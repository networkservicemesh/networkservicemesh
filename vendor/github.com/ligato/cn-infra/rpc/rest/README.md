# HTTPmux

The `REST Plugin` is a infrastructure Plugin which allows app plugins 
to handle HTTP requests (see the diagram below) in this sequence:
1. httpmux starts the HTTP server in its own goroutine
2. Plugins register their handlers with `REST Plugin`.
   To service HTTP requests, a plugin must first implement a handler
   function and register it at a given URL path using
   the `RegisterHTTPHandler` method. `REST Plugin` uses an HTTP request
   multiplexer from the `gorilla/mux` package to register
   the HTTP handlers by the specified URL path.
3. HTTP server routes HTTP requests to their respective registered handlers
   using the `gorilla/mux` multiplexer.

![http](../../docs/imgs/http.png)

**Configuration**

- the server's port can be defined using commandline flag `http-port` or 
  via the environment variable HTTP_PORT.

**Example**

The following example demonstrates the usage of the `REST Plugin` plugin
API:
```
// httpExampleHandler returns a very simple HTTP request handler.
func httpExampleHandler(formatter *render.Render) http.HandlerFunc {

    // An example HTTP request handler which prints out attributes of 
    // a trivial Go structure in JSON format.
    return func(w http.ResponseWriter, req *http.Request) {
        formatter.JSON(w, http.StatusOK, struct{ Example string }{"This is an example"})
    }
}

// Register our HTTP request handler as a GET method serving at 
// the URL path "/example".
httpmux.RegisterHTTPHandler("/example", httpExampleHandler, "GET")
```

Once the handler is registered with `REST Plugin` and the agent is running, 
you can use `curl` to verify that it is operating properly:
```
$ curl -X GET http://localhost:9191/example
{
  "Example": "This is an example"
}
```


## Security

REST plugin allows to optionally configure following security features:
- server certificate (HTTPS)
- Basic HTTP Authentication - username & password
- client certificates

All of them are disabled by default and can be enabled by config file:

```yaml
endpoint: 127.0.0.1:9292
server-cert-file: server.crt
server-key-file: server.key
client-cert-files:
  - "ca.crt"
client-basic-auth:
  - "user:pass"
  - "foo:bar"
```

If `server-cert-file` and `server-key-file` are defined the server requires HTTPS instead
of HTTP for all its endpoints.

`client-cert-files` the list of the root certificate authorities that server uses to validate
client certificates. If the list is not empty only client who provide a valid certificate
is allowed to access the server.

`client-basic-auth` allows to define user password pairs that are allowed to access the
server. The config option defines a static list of allowed user. If the list is not empty default
staticAuthenticator is instantiated. Alternatively, you can implement custom authenticator and inject it
into the plugin (e.g.: if you want to read credentials from ETCD).


***Example***

In order to generated self-signed certificates you can use the following commands:

```bash
#generate key for "Our Certificate Authority"
openssl genrsa -out ca.key 2048

#generate certificate for CA
openssl req -new -nodes -x509 -key ca.key -out ca.crt  -subj '/CN=CA'

#generate certificate for the server assume that server will be accessed by 127.0.0.1
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj '/CN=127.0.0.1'
openssl x509 -req -extensions client_server_ssl -extfile openssl_ext.conf -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

#generate client certificate
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr -subj '/CN=client'
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 360

```

Once the security features are enabled, the endpoint can be accessed by the following commands:

- **HTTPS**
where `ca.pem` is a certificate authority where server certificate should be validated (in case of self-signed certificates)
  ```
  curl --cacert ca.crt  https://127.0.0.1:9292/log/list
  ```

- **HTTPS + client cert** where `client.crt` is a valid client certificate.
  ```
  curl --cacert ca.crt  --cert client.crt --key client.key  https://127.0.0.1:9292/log/list
  ```

- **HTTPS + basic auth** where `user:pass` is a valid username password pair.
  ```
  curl --cacert ca.crt  -u user:pass  https://127.0.0.1:9292/log/list
  ```

- **HTTPS + client cert + basic auth**
  ```
  curl --cacert ca.crt  --cert client.crt --key client.key -u user:pass  https://127.0.0.1:9292/log/list
  ```
