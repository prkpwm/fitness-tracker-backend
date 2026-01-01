package main

type UserProfile struct {
	Age                        int     `json:"age"`
	WeightKg                  float64 `json:"weight_kg"`
	HeightCm                  float64 `json:"height_cm"`
	BmrKcal                   int     `json:"bmr_kcal"`
	TdeeMaintenanceKcal       int     `json:"tdee_maintenance_kcal"`
	TargetLoseWeightKcal      int     `json:"target_lose_weight_kcal"`
	TargetProteinG            int     `json:"target_protein_g"`
	GoalCalories              int     `json:"goal_calories"`
	MaintenanceProteinTargetG int     `json:"maintenance_protein_target_g"`
}

type FoodItem struct {
	Time     string  `json:"time"`
	Item     string  `json:"item"`
	Calories int     `json:"calories"`
	ProteinG float64 `json:"protein_g"`
	CarbsG   float64 `json:"carbs_g"`
	FatG     float64 `json:"fat_g"`
}

type ExerciseSummary struct {
	TotalBurnedCalories int    `json:"total_burned_calories"`
	Status              string `json:"status,omitempty"`
}

type DailyTotalStats struct {
	TotalIntakeCalories  int     `json:"total_intake_calories"`
	TotalBurnedCalories  int     `json:"total_burned_calories"`
	NetCalories          int     `json:"net_calories"`
	TotalProteinG        float64 `json:"total_protein_g"`
	TotalCarbsG          float64 `json:"total_carbs_g"`
	TotalFatG            float64 `json:"total_fat_g"`
	ProteinPerKg         float64 `json:"protein_per_kg"`
}

type AIEvaluation struct {
	MuscleMaintenance  string `json:"muscle_maintenance"`
	WeightLossStatus   string `json:"weight_loss_status"`
	Recommendation     string `json:"recommendation"`
}

type FitnessData struct {
	Date            string           `json:"date"`
	UserProfile     UserProfile      `json:"user_profile"`
	FoodDiary       []FoodItem       `json:"food_diary"`
	ExerciseSummary ExerciseSummary  `json:"exercise_summary"`
	DailyTotalStats DailyTotalStats  `json:"daily_total_stats"`
	AIEvaluation    AIEvaluation     `json:"ai_evaluation"`
}