#!/bin/bash

# This script sends a sample JSON payload to the /webhook/account endpoint
# to test the server's webhook handling functionality.

echo "Sending test webhook to http://localhost:8080/webhook/account"

curl -X POST \
  http://localhost:8080/webhook/account \
  -H 'Content-Type: application/json' \
  -d '{ 
    "id": 98765, 
    "last_name": "Test", 
    "full_name": "Webhook Test", 
    "phone_number": "555-1234", 
    "email": "webhook@test.com", 
    "original_address": "123 Main St, Anytown, USA" 
  }'

echo -e "\n\nTest complete. Check the server output for confirmation."
