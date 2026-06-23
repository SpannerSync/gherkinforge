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
