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
  executor_type: "mock" | "qiskit_aer" | "quantify" | "presto" | "qblox";
  calibration_data?: unknown;
}

export interface JobResult {
  shots: number;
  backend: string;
  success: boolean;
  counts?: Record<string, number>;
  hex_counts?: Record<string, number>;
  memory?: number[][][];
  circuit_results?: unknown[];
}

export interface QuantumJob {
  id: string;
  user_id: string;
  qpu_target: string;
  payload: string; // JSON string
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  results?: JobResult; // Qiskit-compatible result dict
  duration?: number; // execution duration in seconds
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
  requested_reason: string;
  status: "pending" | "approved" | "rejected";
  rejection_reason?: string;
  expand?: {
    user?: User;
  };
}

export interface NotificationRequest {
  title: string;
  description: string;
  start_time?: string;
  end_time?: string;
  target_users?: string[];
}

export interface Notification extends NotificationRequest {
  id: string;
  title: string;
  description: string;
  user_id?: string;
  dismissed_by?: string[];
  created: string;
}

export interface CreateQpuRequest {
  name: string;
  executor_type?: string;
  num_qubits?: number;
  enabled?: boolean;
}

export interface CreateQpuResponse {
  id: string;
  name: string;
  access_token: string;
  executor_type: string;
  status: string;
  enabled: boolean;
  qpi_addr: string;
  ca_fingerprint: string;
  driver_version: string;
}
