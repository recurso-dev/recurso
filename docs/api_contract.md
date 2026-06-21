# Recurso: API Contract (OpenAPI 3.1)

This document defines the REST API surface for Recurso.

## Overview
- **Base URL**: `https://api.recurso.com/v1`
- **Authentication**: Usage of `Authorization: Basic <api_key>:` header.
- **Content-Type**: `application/json`

## API Specification (YAML)

```yaml
openapi: 3.1.0
info:
  title: Recurso API
  version: 1.0.0
  description: The Revenue Operating System API.
servers:
  - url: https://api.recurso.com/v1
    description: Production Server

components:
  securitySchemes:
    basicAuth:
      type: http
      scheme: basic
      description: Use your API Key as the username. Password can be empty.
  
  schemas:
    Plan:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        code:
          type: string
          description: Unique identifier (e.g. "gold-monthly")
        interval_unit:
          type: string
          enum: [day, week, month, year]
        interval_count:
          type: integer
        amount:
          type: integer
          description: Price in lowest currency unit (e.g. cents)
        currency:
          type: string
          minLength: 3
          maxLength: 3

    Customer:
      type: object
      properties:
        id:
          type: string
          format: uuid
        email:
          type: string
          format: email
        name:
          type: string
        billing_address:
          $ref: '#/components/schemas/Address'
        tax_info:
          $ref: '#/components/schemas/TaxInfo'

    Address:
      type: object
      properties:
        line1: { type: string }
        city: { type: string }
        state: { type: string }
        zip: { type: string }
        country: { type: string, minLength: 2, maxLength: 2 }

    TaxInfo:
      type: object
      properties:
        gstin: { type: string }
        vat_number: { type: string }

    Subscription:
      type: object
      properties:
        id: { type: string, format: uuid }
        customer_id: { type: string, format: uuid }
        plan_id: { type: string, format: uuid }
        status:
          type: string
          enum: [trialing, active, past_due, paused, canceled, unpaid]
        current_period_end:
          type: string
          format: date-time
    
    Error:
      type: object
      properties:
        code: { type: string }
        message: { type: string }

security:
  - basicAuth: []

paths:
  # PLANS
  /plans:
    post:
      summary: Create a Plan
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Plan'
      responses:
        '201':
          description: Plan created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Plan'
    get:
      summary: List Plans
      responses:
        '200':
          description: List of plans
          content:
            application/json:
              schema:
                type: array
                items: 
                  $ref: '#/components/schemas/Plan'

  /plans/{id}:
    get:
      summary: Retrieve a Plan
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: Plan details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Plan'

  # CUSTOMERS
  /customers:
    post:
      summary: Create a Customer
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Customer'
      responses:
        '201':
          description: Customer created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Customer'

  /customers/{id}:
    get:
      summary: Retrieve a Customer
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: Customer details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Customer'

  # SUBSCRIPTIONS
  /subscriptions:
    post:
      summary: Create a Subscription
      description: This endpoints calculates proration if needed, charges the payment method, and creates the subscription.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                customer_id: { type: string, format: uuid }
                plan_id: { type: string, format: uuid }
                start_date: { type: string, format: date-time }
      responses:
        '201':
          description: Subscription created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Subscription'
```
