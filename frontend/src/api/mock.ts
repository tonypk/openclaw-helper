/**
 * Mock data for frontend development without Go helper.
 */
import type {
  SystemReport,
  HelperInfo,
  InstallStatus,
  ProgressEvent,
  DiagnosticReport,
  ChatResponse,
  RepairResult,
  CrashReport,
  ReportResult,
} from "./helper";

const delay = (ms: number) => new Promise((r) => setTimeout(r, ms));

export const mockHelperInfo: HelperInfo = {
  version: "0.1.0-mock",
  go_version: "go1.22.0",
  os: "windows",
  arch: "amd64",
};

export const mockSystemReport: SystemReport = {
  os: { name: "os", status: "pass", message: "Windows 11 23H2" },
  memory: { name: "memory", status: "pass", message: "16 GB" },
  disk: {
    name: "disk",
    status: "pass",
    message: "50 GB available",
  },
  virtualization: {
    name: "virtualization",
    status: "pass",
    message: "Hyper-V enabled",
  },
  wsl: {
    name: "wsl",
    status: "warn",
    message: "WSL2 not installed",
    detail: "Will be installed automatically",
  },
  node: { name: "node", status: "fail", message: "Not found in WSL" },
  openclaw: {
    name: "openclaw",
    status: "fail",
    message: "Not installed",
  },
  overall_ready: false,
};

export const mockInstallStatus: InstallStatus = {
  current_phase: "idle",
  phases: [
    {
      phase: "precheck",
      label: "Pre-check",
      label_zh: "\u51C6\u5907",
      status: "pending",
    },
    {
      phase: "wsl",
      label: "WSL",
      label_zh: "WSL \u73AF\u5883",
      status: "pending",
    },
    {
      phase: "ubuntu",
      label: "Ubuntu",
      label_zh: "Ubuntu",
      status: "pending",
    },
    {
      phase: "node",
      label: "Node.js",
      label_zh: "Node.js",
      status: "pending",
    },
    {
      phase: "openclaw",
      label: "OpenClaw",
      label_zh: "OpenClaw",
      status: "pending",
    },
    {
      phase: "config",
      label: "Config",
      label_zh: "\u914D\u7F6E",
      status: "pending",
    },
    {
      phase: "verify",
      label: "Verify",
      label_zh: "\u9A8C\u8BC1",
      status: "pending",
    },
  ],
  running: false,
  started_at: "",
  overall: 0,
};

export const mockChatResponse: ChatResponse = {
  message:
    "\u8FD9\u662F\u6A21\u62DF\u56DE\u590D\u3002\u5F00\u53D1\u6A21\u5F0F\u4E0B Go helper \u672A\u8FD0\u884C\uFF0C\u8BF7\u4F7F\u7528 `make dev` \u542F\u52A8\u5B8C\u6574\u5E94\u7528\u3002",
  source: "offline",
  suggested: [
    {
      text: "API Key \u600E\u4E48\u83B7\u53D6\uFF1F",
      text_en: "How to get API Key?",
    },
    {
      text: "\u5B89\u88C5\u597D\u6162\u600E\u4E48\u529E\uFF1F",
      text_en: "Installation is slow?",
    },
  ],
};

const mockMethods: Record<string, (params?: unknown) => unknown> = {
  "helper.ping": () => "pong",
  "helper.version": () => mockHelperInfo,
  "system.check": () => mockSystemReport,
  "system.checkSingle": (p) => {
    const name = (p as { name: string })?.name ?? "os";
    const report = mockSystemReport as unknown as Record<string, unknown>;
    return report[name] ?? { name, status: "skipped", message: "Unknown" };
  },
  "install.start": () => "started",
  "install.status": () => mockInstallStatus,
  "install.retry": () => "retrying",
  "install.cancel": () => "cancelled",
  "install.reset": () => "reset",
  "install.events": () => [] as ProgressEvent[],
  "diagnosis.run": () =>
    ({
      issues: [],
      healthy: true,
      timestamp: new Date().toISOString(),
    }) as DiagnosticReport,
  "diagnosis.runWithError": () =>
    ({
      issues: [],
      healthy: true,
      timestamp: new Date().toISOString(),
    }) as DiagnosticReport,
  "diagnosis.repair": (p) =>
    ({
      repair_id: (p as { repair_id: string })?.repair_id ?? "",
      success: true,
      message: "Mock repair completed",
    }) as RepairResult,
  "chat.ask": () => mockChatResponse,
  "chat.setContext": () => "ok",
  "chat.suggestions": () => mockChatResponse.suggested,
  "report.collect": () =>
    ({
      title: "Installation failed - wsl",
      description: "",
      app_version: "0.1.0-mock",
      system_summary:
        "OS: Windows 11 23H2 (pass)\nMemory: 16 GB\nDisk: 50 GB available\nWSL: Not installed (fail)\nNode: Not found (fail)",
      error_phase: "wsl",
      error_message: "WSL installation timed out",
    }) as CrashReport,
  "report.submit": () =>
    ({
      github_url:
        "https://github.com/tonypk/openclaw-helper/issues/new?title=test&labels=crash-report&body=test",
      telegram_sent: false,
    }) as ReportResult,
};

export async function mockCall<T>(
  method: string,
  params?: unknown,
): Promise<T> {
  await delay(300 + Math.random() * 500);
  const handler = mockMethods[method];
  if (!handler) {
    throw new Error(`Mock: unknown method ${method}`);
  }
  return handler(params) as T;
}
