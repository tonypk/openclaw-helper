/**
 * IPC bridge to Go Helper process.
 * In Tauri: uses sidecar + named pipe.
 * In dev mode: connects to local unix socket.
 */

export interface RPCResponse<T = unknown> {
  jsonrpc: string;
  id: number;
  result?: T;
  error?: { code: number; message: string };
}

export interface CheckResult {
  name: string;
  status: "pass" | "fail" | "warn" | "checking" | "skipped";
  message: string;
  detail?: string;
}

export interface SystemReport {
  os: CheckResult;
  memory: CheckResult;
  disk: CheckResult;
  virtualization: CheckResult;
  wsl: CheckResult;
  node: CheckResult;
  openclaw: CheckResult;
  overall_ready: boolean;
}

export interface HelperInfo {
  version: string;
  go_version: string;
  os: string;
  arch: string;
}

export interface PhaseProgress {
  phase: string;
  label: string;
  label_zh: string;
  status: "pending" | "running" | "completed" | "failed" | "skipped";
}

export interface InstallStatus {
  current_phase: string;
  phases: PhaseProgress[];
  running: boolean;
  error_message?: string;
  error_phase?: string;
  started_at: string;
  overall: number;
}

export interface ProgressEvent {
  phase: string;
  status: string;
  message: string;
  detail?: string;
  progress: number;
  overall: number;
  timestamp: string;
}

export interface DiagnosticReport {
  issues: DiagIssue[];
  healthy: boolean;
  timestamp: string;
}

export interface DiagIssue {
  id: string;
  severity: "critical" | "warning" | "info";
  title: string;
  title_zh: string;
  description: string;
  desc_zh: string;
  repair_id?: string;
  auto_repair: boolean;
}

export interface RepairResult {
  repair_id: string;
  success: boolean;
  message: string;
  msg_zh?: string;
}

export interface ChatResponse {
  message: string;
  source: "faq" | "diagnosis" | "llm" | "offline";
  repair_id?: string;
  auto_repair?: boolean;
  suggested?: { text: string; text_en: string }[];
}

let rpcId = 0;
let useMock = false;

/**
 * Call a Go Helper RPC method.
 * In production, this goes through Tauri's invoke.
 * In dev without Go helper, falls back to mock data.
 */
async function call<T>(method: string, params?: unknown): Promise<T> {
  rpcId++;

  // Try Tauri invoke first
  if (window.__TAURI__) {
    const { invoke } = window.__TAURI__.core;
    return invoke<T>("helper_rpc", { method, params: params ?? null });
  }

  // If we already know mock mode is needed, skip HTTP
  if (useMock) {
    const { mockCall } = await import("./mock");
    return mockCall<T>(method, params);
  }

  // Try HTTP bridge, fall back to mock
  try {
    const resp = await fetch("http://localhost:19999/rpc", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ jsonrpc: "2.0", id: rpcId, method, params }),
    });

    const data: RPCResponse<T> = await resp.json();
    if (data.error) {
      throw new Error(`RPC ${method}: ${data.error.message}`);
    }
    return data.result as T;
  } catch {
    console.warn("[helper] HTTP bridge unavailable, switching to mock mode");
    useMock = true;
    const { mockCall } = await import("./mock");
    return mockCall<T>(method, params);
  }
}

// --- Helper ---
export const helperPing = () => call<string>("helper.ping");
export const helperVersion = () => call<HelperInfo>("helper.version");

// --- System Check ---
export const systemCheck = () => call<SystemReport>("system.check");
export const systemCheckSingle = (name: string) =>
  call<CheckResult>("system.checkSingle", { name });

// --- Install ---
export const installStart = () => call<string>("install.start");
export const installStatus = () => call<InstallStatus>("install.status");
export const installRetry = () => call<string>("install.retry");
export const installCancel = () => call<string>("install.cancel");
export const installReset = () => call<string>("install.reset");
export const installEvents = () => call<ProgressEvent[]>("install.events");

// --- Diagnosis ---
export const diagnosisRun = () => call<DiagnosticReport>("diagnosis.run");
export const diagnosisRunWithError = (errorLog: string) =>
  call<DiagnosticReport>("diagnosis.runWithError", { error_log: errorLog });
export const diagnosisRepair = (repairId: string) =>
  call<RepairResult>("diagnosis.repair", { repair_id: repairId });

// --- Chat ---
export const chatAsk = (message: string) =>
  call<ChatResponse>("chat.ask", { message });
export const chatSetContext = (ctx: {
  phase?: string;
  error_log?: string;
  language?: string;
}) => call<string>("chat.setContext", ctx);
export const chatSuggestions = () =>
  call<{ text: string; text_en: string }[]>("chat.suggestions");

// --- Update ---
export interface UpdateInfo {
  available: boolean;
  version?: string;
  body?: string;
}

export async function checkUpdate(): Promise<UpdateInfo> {
  if (window.__TAURI__) {
    const { invoke } = window.__TAURI__.core;
    return invoke<UpdateInfo>("check_update");
  }
  return { available: false };
}

// Tauri type augmentation
declare global {
  interface Window {
    __TAURI__?: {
      core: {
        invoke: <T>(cmd: string, args?: Record<string, unknown>) => Promise<T>;
      };
    };
  }
}
