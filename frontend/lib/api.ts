const BASE = "/api";

function getToken(): string {
  if (typeof window === "undefined") return "";
  return localStorage.getItem("token") || "";
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${getToken()}`,
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (res.status === 401) {
    localStorage.removeItem("token");
    window.location.href = "/login";
    throw new Error("unauthorized");
  }

  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "request failed");
  return data as T;
}

export const api = {
  get: <T>(path: string) => request<T>("GET", path),
  post: <T>(path: string, body: unknown) => request<T>("POST", path, body),
  put: <T>(path: string, body: unknown) => request<T>("PUT", path, body),
  delete: <T>(path: string) => request<T>("DELETE", path),
};

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  totp_enabled: boolean;
  created_at: string;
}

export interface Server {
  id: string;
  name: string;
  hostname: string;
  ip_address: string;
  agent_url: string;
  role: string;
  status: "online" | "offline" | "unknown";
  last_seen: string | null;
  created_at: string;
}

export interface ServerMetrics {
  server_id: string;
  timestamp: string;
  cpu: {
    usage_percent: number;
    cores: number;
  };
  memory: {
    total_bytes: number;
    used_bytes: number;
    free_bytes: number;
    usage_percent: number;
  };
  disk: {
    total_bytes: number;
    used_bytes: number;
    free_bytes: number;
    usage_percent: number;
  };
  network: {
    bytes_sent: number;
    bytes_recv: number;
  };
  uptime: number;
  load_avg: {
    load1: number;
    load5: number;
    load15: number;
  };
  hostname: string;
  os: string;
  kernel_version: string;
}

export interface Website {
  id: string;
  server_id: string;
  server_name?: string;
  domain: string;
  php_version: string;
  document_root: string;
  aliases: string[];
  ssl_enabled: boolean;
  ssl_cert_id?: string;
  enabled: boolean;
  created_at: string;
}

export interface SSLCert {
  id: string;
  server_id: string;
  server_name?: string;
  domain: string;
  san_domains: string[];
  email: string;
  issuer: string;
  status: "active" | "pending" | "expired" | "failed";
  expires_at: string;
  auto_renew: boolean;
  created_at: string;
}

export interface ManagedDatabase {
  id: string;
  server_id: string;
  server_name?: string;
  name: string;
  db_type: "mysql" | "postgresql";
  db_user: string;
  size_bytes: number;
  created_at: string;
}

export interface DNSZone {
  id: string;
  server_id: string;
  server_name?: string;
  name: string;
  nameserver: string;
  admin_email: string;
  serial: number;
  created_at: string;
}

export interface DNSRecord {
  id: string;
  zone_id: string;
  name: string;
  type: "A" | "AAAA" | "CNAME" | "MX" | "TXT" | "SRV" | "CAA";
  content: string;
  ttl: number;
  priority?: number;
  created_at: string;
}

export interface MailDomain {
  id: string;
  server_id: string;
  server_name?: string;
  name: string;
  created_at: string;
}

export interface MailAccount {
  id: string;
  domain_id: string;
  domain_name?: string;
  username: string;
  quota_mb: number;
  created_at: string;
}

export interface MailAlias {
  id: string;
  domain_id: string;
  domain_name?: string;
  source: string;
  destination: string;
  created_at: string;
}

export interface FirewallRule {
  id: string;
  server_id: string;
  server_name?: string;
  order: number;
  action: "allow" | "deny";
  direction: "in" | "out";
  protocol: "tcp" | "udp" | "icmp" | "any";
  source: string;
  dest_port: string;
  comment: string;
  enabled: boolean;
  created_at: string;
}

export interface BackupConfig {
  id: string;
  server_id: string;
  server_name?: string;
  name: string;
  storage_type: "local" | "s3" | "sftp";
  schedule: string;
  retention_days: number;
  include_paths: string[];
  encrypt: boolean;
  enabled: boolean;
  s3_bucket?: string;
  s3_region?: string;
  s3_access_key?: string;
  sftp_host?: string;
  sftp_user?: string;
  sftp_path?: string;
  created_at: string;
}

export interface BackupJob {
  id: string;
  config_id: string;
  config_name?: string;
  status: "running" | "success" | "failed" | "pending";
  size_bytes: number;
  started_at: string;
  finished_at?: string;
  error?: string;
}

export interface CronJob {
  id: string;
  server_id: string;
  server_name?: string;
  name: string;
  schedule: string;
  command: string;
  run_as_user: string;
  enabled: boolean;
  last_run?: string;
  last_status?: "success" | "failed" | "running";
  created_at: string;
}

export interface SystemService {
  name: string;
  description: string;
  active: boolean;
  enabled: boolean;
  status: string;
}

export interface Alert {
  level: "critical" | "warning" | "info";
  category: string;
  message: string;
  value: string;
  threshold: string;
  time: string;
}

export interface HealthReport {
  healthy: boolean;
  alerts: Alert[];
  score: number;
  server_id: string;
}

export interface PanelInfo {
  commit: string;
  date: string;
  install_dir: string;
}

export interface PanelUpdateStatus {
  available: boolean;
  current_commit: string;
  latest_commit: string;
  published_at: string;
  checked_at: string;
  error?: string;
}

export interface PanelUpdateCheck {
  available: boolean;
  current_commit: string;
  latest_commit: string;
  published_at: string;
}

export interface PanelUpdateResult {
  previous_commit: string;
  new_commit: string;
  duration: string;
  restarted_at: string;
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}
