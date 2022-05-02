#!/bin/sh -eux

mkdir -p github.com/lucas-clemente
cd github.com/lucas-clemente

rm -rf quic-go
git clone --quiet https://github.com/lucas-clemente/quic-go
cd quic-go
git fetch --quiet origin refs/pull/3402/head:http3-graceful-shutdown refs/pull/3397/head:http3-do-not-embed-http-server
git merge -n --no-edit http3-do-not-embed-http-server
git merge -n --no-edit -X theirs http3-graceful-shutdown
git apply <<EOF
diff --git a/http3/server.go b/http3/server.go
index 3fe73318..e988ef6b 100644
--- a/http3/server.go
+++ b/http3/server.go
@@ -251,10 +251,6 @@ func (s *Server) Serve(conn net.PacketConn) error {
 // and use it to construct a http3-friendly QUIC listener.
 // Closing the server does close the listener.
 func (s *Server) ServeListener(ln quic.EarlyListener) error {
-	if s.Server == nil {
-		return errors.New("use of http3.Server without http.Server")
-	}
-
 	if s.getClosed() {
 		return http.ErrServerClosed
 	}
@@ -285,10 +281,7 @@ func (s *Server) serveConn(tlsConf *tls.Config, conn net.PacketConn) error {
 		return errServerWithoutTLSConfig
 	}
 
-	s.mutex.Lock()
-	closed := s.closed
-	s.mutex.Unlock()
-	if closed {
+	if s.getClosed() {
 		return http.ErrServerClosed
 	}
 
EOF
rm -rf .git

find . -type f \( \( \! -name '*.go' -o -name '*_test.go' \) -a \! -name go.mod -a \! -name go.sum -a \! -name LICENSE \) -delete
find . -type f -name '*.go' -exec sed -i 's:github.com/lucas-clemente/quic-go:go.pact.im/x/third_party/github.com/lucas-clemente/quic-go:g' '{}' \;

go mod edit -module go.pact.im/x/third_party/github.com/lucas-clemente/quic-go
go work sync
go mod tidy
rm -f ../../../../../go.work.sum
