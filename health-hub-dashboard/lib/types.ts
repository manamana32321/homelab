export interface AggregatedMetric {
  time: string;
  avg: number;
  min: number;
  max: number;
  sum: number;
  count: number;
}

export interface SleepStage {
  stage: string;
  start_time: string;
  end_time: string;
  duration_m: number;
}

export interface SleepSession {
  id: number;
  start_time: string;
  end_time: string;
  duration_m: number;
  stages: SleepStage[];
}

export interface ExerciseSession {
  id: number;
  exercise_type: string;
  start_time: string;
  end_time: string;
  duration_m: number;
  calories_kcal?: number;
  distance_m?: number;
  metadata?: Record<string, unknown>;
}

export interface NutritionRecord {
  id: number;
  time: string;
  meal_type?: string;
  calories?: number;
  protein_g?: number;
  fat_g?: number;
  carbs_g?: number;
  metadata?: Record<string, unknown>;
}

export interface BodyMeasurement {
  time: string;
  weight_kg?: number;
  body_fat_pct?: number;
  lean_mass_kg?: number;
}

export interface DailySummary {
  date: string;
  total_steps?: number;
  avg_heart_rate?: number;
  total_calories?: number;
  total_distance_m?: number;
  sleep?: SleepSession;
  exercises?: ExerciseSession[];
  avg_spo2?: number;
  weight?: BodyMeasurement;
}

export type MetricType =
  | "steps"
  | "heart_rate"
  | "spo2"
  | "calories"
  | "distance"
  | "hydration";
