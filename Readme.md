# регистрация юзера 
POST → http://localhost:8085/register
Для пассажира:
{
"name": "Alex Petrov",
"email": "alex@example.com",
"password": "123456",
"role": "PASSENGER"
}

Для водителя:
{
"name": "Ivan Ivanov",
"email": "ivan@example.com",
"password": "123456",
"role": "DRIVER",
"license_number": "KZ123ABC",
"vehicle_type": "ECONOMY",
"vehicle_attrs": {
"vehicle_make": "Toyota",
"vehicle_model": "Camry",
"vehicle_color": "White",
"vehicle_plate": "KZ123ABC",
"vehicle_year": 2020
}
}