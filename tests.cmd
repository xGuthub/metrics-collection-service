metricstest.exe -test.v -test.run=^TestIteration1$ -binary-path=cmd/server/server
metricstest.exe -test.v -test.run=^TestIteration2[AB]$ -source-path=. -agent-binary-path=cmd/agent/agent