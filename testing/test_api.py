#!/usr/bin/env python3
"""
Test script for the mock BadgerMaps API server
This script demonstrates how to make HTTP requests to the mock server
"""

import requests
import json
import sys

# Base URL for the mock server
BASE_URL = "http://localhost:8080/api/2"

def test_profile():
    """Test the profile endpoint"""
    print("Testing GET /api/2/profile/")
    response = requests.get(f"{BASE_URL}/profile/")
    print(f"Status: {response.status_code}")
    if response.status_code == 200:
        data = response.json()
        print(f"User: {data.get('first_name')} {data.get('last_name')}")
        print(f"Email: {data.get('email')}")
        print(f"Company: {data.get('company', {}).get('name')}")
    print()

def test_customers_list():
    """Test the customers list endpoint"""
    print("Testing GET /api/2/customers/")
    response = requests.get(f"{BASE_URL}/customers/")
    print(f"Status: {response.status_code}")
    if response.status_code == 200:
        customers = response.json()
        print(f"Found {len(customers)} customers:")
        for customer in customers[:3]:  # Show first 3
            print(f"  - {customer.get('first_name')} {customer.get('last_name')} (ID: {customer.get('id')})")
    print()

def test_customer_detail(customer_id=1001):
    """Test the customer detail endpoint"""
    print(f"Testing GET /api/2/customers/{customer_id}/")
    response = requests.get(f"{BASE_URL}/customers/{customer_id}/")
    print(f"Status: {response.status_code}")
    if response.status_code == 200:
        customer = response.json()
        print(f"Customer: {customer.get('first_name')} {customer.get('last_name')}")
        print(f"Email: {customer.get('email')}")
        print(f"Phone: {customer.get('phone_number')}")
        print(f"Locations: {len(customer.get('locations', []))}")
    print()

def test_customer_update(customer_id=1001):
    """Test the customer update endpoint"""
    print(f"Testing PATCH /api/2/customers/{customer_id}/")
    update_data = {
        "first_name": "John",
        "last_name": "Smith",
        "email": "john.smith.updated@example.com",
        "custom_text": "Updated via API test"
    }
    response = requests.patch(f"{BASE_URL}/customers/{customer_id}/", json=update_data)
    print(f"Status: {response.status_code}")
    if response.status_code == 200:
        result = response.json()
        print("Update successful!")
        print(f"Updated email: {result.get('email')}")
    print()

def test_checkins_list(customer_id=1001):
    """Test the check-ins list endpoint"""
    print(f"Testing GET /api/2/appointments/?customer_id={customer_id}")
    response = requests.get(f"{BASE_URL}/appointments/", params={"customer_id": customer_id})
    print(f"Status: {response.status_code}")
    if response.status_code == 200:
        checkins = response.json()
        print(f"Found {len(checkins)} check-ins:")
        for checkin in checkins[:3]:  # Show first 3
            print(f"  - {checkin.get('type')} on {checkin.get('log_datetime')}")
    print()

def test_checkin_create():
    """Test creating a new check-in"""
    print("Testing POST /api/2/appointments/")
    checkin_data = {
        "customer": 1001,
        "log_datetime": "2024-01-15T16:00:00Z",
        "type": "visit",
        "comments": "Test check-in created via API",
        "extra_fields": '{"duration": "30", "test": true}',
        "created_by": "test@example.com"
    }
    response = requests.post(f"{BASE_URL}/appointments/", json=checkin_data)
    print(f"Status: {response.status_code}")
    if response.status_code == 201:
        result = response.json()
        print("Check-in created successfully!")
        print(f"Check-in ID: {result.get('id')}")
        print(f"Type: {result.get('type')}")
    print()

def test_routes_list():
    """Test the routes list endpoint"""
    print("Testing GET /api/2/routes/")
    response = requests.get(f"{BASE_URL}/routes/")
    print(f"Status: {response.status_code}")
    if response.status_code == 200:
        routes = response.json()
        print(f"Found {len(routes)} routes:")
        for route in routes[:3]:  # Show first 3
            print(f"  - {route.get('name')} on {route.get('route_date')}")
    print()

def test_route_detail(route_id=4001):
    """Test the route detail endpoint"""
    print(f"Testing GET /api/2/routes/{route_id}/")
    response = requests.get(f"{BASE_URL}/routes/{route_id}/")
    print(f"Status: {response.status_code}")
    if response.status_code == 200:
        route = response.json()
        print(f"Route: {route.get('name')}")
        print(f"Date: {route.get('route_date')}")
        print(f"Waypoints: {len(route.get('waypoints', []))}")
    print()

def test_user_search(query="john"):
    """Test the user search endpoint"""
    print(f"Testing GET /api/2/search/users/?q={query}")
    response = requests.get(f"{BASE_URL}/search/users/", params={"q": query})
    print(f"Status: {response.status_code}")
    if response.status_code == 200:
        users = response.json()
        print(f"Found {len(users)} users matching '{query}':")
        for user in users:
            print(f"  - {user.get('first_name')} ({user.get('username')})")
    print()

def test_data_fields():
    """Test the data fields endpoint"""
    print("Testing GET /api/2/datafields/")
    response = requests.get(f"{BASE_URL}/datafields/")
    print(f"Status: {response.status_code}")
    if response.status_code == 200:
        datafields = response.json()
        print(f"Found {len(datafields)} data fields:")
        for field in datafields:
            print(f"  - {field.get('name')}: {field.get('label')} ({field.get('type')})")
    print()

def test_error_handling():
    """Test error handling"""
    print("Testing error handling...")
    
    # Test invalid endpoint
    print("Testing invalid endpoint")
    response = requests.get(f"{BASE_URL}/invalid/")
    print(f"Status: {response.status_code}")
    if response.status_code == 404:
        error = response.json()
        print(f"Error: {error.get('error')}")
        print(f"Message: {error.get('message')}")
    
    # Test missing parameter
    print("\nTesting missing customer_id parameter")
    response = requests.get(f"{BASE_URL}/appointments/")
    print(f"Status: {response.status_code}")
    if response.status_code == 400:
        error = response.json()
        print(f"Error: {error.get('error')}")
        print(f"Message: {error.get('message')}")
    print()

def main():
    """Run all tests"""
    print("=" * 60)
    print("BadgerMaps Mock API Test Suite")
    print("=" * 60)
    print()
    
    try:
        # Test all endpoints
        test_profile()
        test_customers_list()
        test_customer_detail()
        test_customer_update()
        test_checkins_list()
        test_checkin_create()
        test_routes_list()
        test_route_detail()
        test_user_search()
        test_data_fields()
        test_error_handling()
        
        print("=" * 60)
        print("All tests completed!")
        print("=" * 60)
        
    except requests.exceptions.ConnectionError:
        print("Error: Could not connect to the mock server.")
        print("Make sure the server is running on http://localhost:8080")
        print("Run: ./run_mock_server.sh (Linux/Mac) or run_mock_server.bat (Windows)")
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main() 