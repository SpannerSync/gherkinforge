# @tier: nfr
# Non-functional requirements for the order domain.
# These scenarios are executed by Go benchmarks (testing.B) and fuzz targets.
# Run benchmarks: go test -bench=. ./tests/...
# Run fuzz:       go test -fuzz=FuzzCreateOrder -fuzztime=30s ./tests/...
@nfr
Feature: Order Throughput and Resilience
  As a platform operator
  I want order creation to meet throughput and resilience requirements
  So that peak garage traffic does not degrade service

  Scenario: Order creation throughput meets SLA
    Given a pre-seeded inventory of 1000 products
    When 500 concurrent order creation requests are submitted
    Then all requests complete within 2 seconds
    And zero orders are lost or duplicated

  Scenario: Domain logic is safe under arbitrary fuzz input
    Given a fuzz corpus of 100 random JSON payloads
    When each payload is submitted to the CreateOrder use case
    Then the use case never panics
    And invalid payloads return structured domain errors
