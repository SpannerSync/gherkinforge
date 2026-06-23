# @tier: integration
# Validates that infrastructure adapters correctly persist and retrieve domain objects.
# These scenarios require a real database (testcontainers-go spins PostgreSQL).
# Run: go test -tags=integration ./tests/integration/...
@integration
Feature: Order Persistence
  As an infrastructure adapter
  I want to save and retrieve orders from the database
  So that order state survives process restarts

  Background: Database is ready
    Given a clean PostgreSQL database is available

  Scenario: Saved order is retrievable by ID
    Given an order with ID "ORD-100" for customer "CUST-500" and total 2999 pence
    When the order is saved to the repository
    Then finding order "ORD-100" returns the same order with total 2999 pence

  Scenario: Finding a non-existent order returns not-found error
    When finding order "ORD-DOES-NOT-EXIST" from the repository
    Then the result is an "order not found" error
