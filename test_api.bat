@echo off
echo Testing Fitness Tracker API...

echo.
echo 1. GET all fitness data:
curl -X GET http://localhost:8080/api/fitness

echo.
echo.
echo 2. POST new fitness data:
curl -X POST http://localhost:8080/api/fitness ^
  -H "Content-Type: application/json" ^
  -d "{\"date\":\"2024-01-15\",\"user_profile\":{\"age\":30,\"weight_kg\":70,\"height_cm\":175,\"recommended_daily_calories\":2000},\"food_diary\":[{\"time\":\"08:00\",\"item\":\"Oatmeal\",\"calories\":300,\"protein_g\":10}],\"exercise_summary\":{\"cardio_session_1\":{\"type\":\"Running\",\"duration_min\":30,\"calories_burned\":300},\"total_burned_calories\":300},\"daily_total_stats\":{\"total_intake_calories\":300,\"total_burned_calories\":300,\"net_calories\":0}}"

echo.
echo.
echo 3. GET fitness data by date:
curl -X GET http://localhost:8080/api/fitness/2024-01-15

echo.
echo.
echo 4. GET raw JSON by date:
curl -X GET "http://localhost:8080/get?date=2024-01-15"

echo.
echo Tests completed!