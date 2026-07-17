#!/bin/bash
for i in {1..10}; do
  curl -s -X POST -H "X-Forwarded-For: 1.2.3.$i, 8.8.8.8" -H "Content-Type: application/json" -d '{"username":"admin", "password":"wrong"}' http://localhost:8080/api/login
done
