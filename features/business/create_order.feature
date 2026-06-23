# @tier: business
# This file is the "Golden Packet" specification.
# It is readable by product owners and executable by godog.
# All monetary values are expressed in pence (int64) to prevent floating-point errors.
@business
Feature: Order Management
  As a customer
  I want to create an order
  So that I can purchase items securely

  Background: System Initialization
    Given the system contains the following valid product inventory:
      | product_id | sku      | unit_pence | stock |
      | P-1001     | WIDGET-X | 2999       | 50    |
      | P-1002     | GADGET-Y | 8950       | 15    |

  Scenario: Successfully creating an order emits a domain event
    Given a customer aggregate initialized with ID "CUST-998"
    When the customer submits a create order command with the following payload:
      """json
      {
        "customer_id": "CUST-998",
        "items": [
          {"product_id": "P-1001", "quantity": 2}
        ]
      }
      """
    Then the order aggregate should be successfully created
    And an "order.created" domain event is published to the broker
    And the total order value in pence should be 5998

  Scenario: Order creation fails when no items are provided
    Given a customer aggregate initialized with ID "CUST-001"
    When the customer submits a create order command with the following payload:
      """json
      {
        "customer_id": "CUST-001",
        "items": []
      }
      """
    Then the order creation should fail with "order must contain at least one item"

  # Zero Trust Pillar 4 — mutation-proof scenario.
  # Every Then clause asserts a precise mathematical value.
  # A mutant that flips TotalPence calculation will fail the 17898 assertion.
  # A mutant that drops the event publish will fail the event assertion.
  # A mutant that clears the ID will fail the non-empty assertion.
  Scenario: Multi-item order total is mathematically verifiable
    Given a customer aggregate initialized with ID "CUST-777"
    When the customer submits a create order command with the following payload:
      """json
      {
        "customer_id": "CUST-777",
        "items": [
          {"product_id": "P-1001", "quantity": 3},
          {"product_id": "P-1002", "quantity": 1}
        ]
      }
      """
    Then the order aggregate should be successfully created
    And the order ID should not be empty
    And an "order.created" domain event is published to the broker
    And the total order value in pence should be 17947
