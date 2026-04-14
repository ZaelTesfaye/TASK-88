#!/bin/bash
echo "Starting Multi-Org Hub development environment..."
docker-compose up --build -d
echo "Waiting for services to be healthy..."
sleep 10
echo ""
echo "Services:"
echo "  Frontend: http://localhost:3000"
echo "  Backend:  http://localhost:8080"
echo "  MySQL:    localhost:3306"
echo ""
echo "Default admin credentials:"
echo "  Username: admin"
echo "  Password: Admin@12345678"
