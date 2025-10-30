# регистрация юзера
POST → http://localhost:8085/register
Для пассажира:
{
"name": "Alex Petrov",
"email": "alex@example.com",
"password": "securityPassword123",
"role": "PASSENGER"
}
ID : e285afdf-3e82-4291-beb2-f86b82c1200
Login
{
"email": "alex@example.com",
"password": "securityPassword123"
}

Для водителя:
{
"name": "Ivan Ivanov",
"email": "ivan@example.com",
"password": "securityPassword123",
"role": "DRIVER",
"license_number": "KZ123ABC",
"vehicle_type": "ECONOMY",
"vehicle_attrs": {
"brand": "Toyota",
"model": "Camry",
"year": 2020,
"color": "White"
}
}
Id : 9f11a85d-ca05-4bfb-8467-4d8bf2dc0a96
Login
{
"email": "ivan@example.com",
"password": "securityPassword123"
}

websocket login
{
"type":"auth",
"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiOWEzYzMyNzctZjk1ZC00MTFhLWE0NmEtZDUyYTc4ZGY1MTFkIiwicm9sZSI6IlBBU1NFTkdFUiIsInR5cGUiOiJhY2Nlc3MiLCJleHAiOjE3NjE1MDUyNzMsImlhdCI6MTc2MTUwNDM3M30.BToTf7K2qd-88bMlqmXZMgQBIz9leULOgjWM6tnZlzE"
}

/ride
{
"passenger_id": "9a3c3277-f95d-411a-a46a-d52a78df511d",
"pickup_latitude": 43.238949,
"pickup_longitude": 76.889709,
"pickup_address": "Abay Ave 25, Almaty",
"destination_latitude": 43.256542,
"destination_longitude": 76.928482,
"destination_address": "Tole Bi St 120, Almaty",
"ride_type": "ECONOMY"
}

After matching passenger send info
{
"type": "ride_details",
"ride_id": "550e8400-e29b-41d4-a716-446655440000",
"passenger_name": "Alex Petrov",
"passenger_phone": "+7-XXX-XXX-XX-XX",
"pickup_location": {
"latitude": 43.238949,
"longitude": 76.889709,
"address": "Abay Ave 25, Almaty",
"notes": "Near the main entrance"
}
}


driver answer to ride

{
"type": "ride_response",
"offer_id": "offer_123456",
"driver_id": "",
"ride_id": "550e8400-e29b-41d4-a716-446655440000",
"accepted": true,
"current_location": {
"latitude": 43.235,
"longitude": 76.885
}
}
update location

{
"driver_id": "660e8400-e29b-41d4-a716-446655440001",
"ride_id": "550e8400-e29b-41d4-a716-446655440000",
"location": {
"lat": 43.236,
"lng": 76.886
},
"speed_kmh": 45.0,
"heading_degrees": 180.0,
"timestamp": "2024-12-16T10:35:30Z"
}