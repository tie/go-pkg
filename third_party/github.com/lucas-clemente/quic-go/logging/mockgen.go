package logging

//go:generate sh -c "mockgen -package logging -self_package go.pact.im/x/third_party/github.com/lucas-clemente/quic-go/logging -destination mock_connection_tracer_test.go go.pact.im/x/third_party/github.com/lucas-clemente/quic-go/logging ConnectionTracer"
//go:generate sh -c "mockgen -package logging -self_package go.pact.im/x/third_party/github.com/lucas-clemente/quic-go/logging -destination mock_tracer_test.go go.pact.im/x/third_party/github.com/lucas-clemente/quic-go/logging Tracer"
