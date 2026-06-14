export interface User {
  id: string;
  email: string;
  username: string;
  qpu_seconds: number;
}

export interface QPU {
  id: string;
  name: string;
  status: "online" | "offline";
  nng_command_port: number;
  nng_result_port: number;
  enabled: boolean;
  executor: "mock" | "qiskit_aer" | "quantify" | "presto" | "qblox";
  calibration_data?: any;
}

export interface QuantumJob {
  id: string;
  user_id: string;
  qpu_target: string;
  payload: string; // JSON string
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  results?: any; // Qiskit-compatible result dict
  created: string;
  finished_at?: string;
}

export interface TimeSlot {
  id: string;
  start_time: string;
  end_time: string;
  booked_by: string;
  expand?: {
    booked_by?: User;
  };
}

export interface TimeRequest {
  id: string;
  user: string;
  seconds: number;
  reason: string;
  status: "pending" | "approved" | "rejected";
  rejection_reason?: string;
  expand?: {
    user?: User;
  };
}

export interface Announcement {
  id: string;
  title: string;
  description: string;
  start_time: string;
  end_time: string;
}

export interface Notification {
  id: string;
  title: string;
  description: string;
  user_id?: string;
  created: string;
}
