package main

type UserProfile struct {
	WeightKg                   float64 `json:"weight_kg"`
	HeightCm                   float64 `json:"height_cm"`
	GoalCalories               int     `json:"goal_calories"`
	MaintenanceProteinTargetG  float64 `json:"maintenance_protein_target_g"`
}

type FoodItem struct {
	Item      string  `json:"item"`
	Calories  int     `json:"calories"`
	ProteinG  float64 `json:"protein_g"`
	CarbsG    float64 `json:"carbs_g"`
}

type Cardio struct {
	Type           string  `json:"type"`
	DistanceMi     float64 `json:"distance_mi"`
	DurationMin    int     `json:"duration_min"`
	CaloriesBurned int     `json:"calories_burned"`
}

type StrengthTraining struct {
	TargetArea     string `json:"target_area"`
	DurationMin    int    `json:"duration_min"`
	CaloriesBurned int    `json:"calories_burned"`
	Intensity      string `json:"intensity"`
}

type Exercise struct {
	Cardio              Cardio           `json:"cardio"`
	StrengthTraining    StrengthTraining `json:"strength_training"`
	TotalBurnedCalories int              `json:"total_burned_calories"`
}

type DailySummary struct {
	TotalIntakeCalories  int     `json:"total_intake_calories"`
	TotalBurnedCalories  int     `json:"total_burned_calories"`
	NetCalories          int     `json:"net_calories"`
	TotalProteinG        float64 `json:"total_protein_g"`
	TotalCarbsG          float64 `json:"total_carbs_g"`
	Status               string  `json:"status"`
}

type FitnessData struct {
	Date         string       `json:"date"`
	UserProfile  UserProfile  `json:"user_profile"`
	FoodDiary    []FoodItem   `json:"food_diary"`
	Exercise     Exercise     `json:"exercise"`
	DailySummary DailySummary `json:"daily_summary"`
}