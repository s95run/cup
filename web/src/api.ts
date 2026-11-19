export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message || `HTTP ${status}`);
    this.status = status;
  }
}

let csrf = "";

async function csrfToken(): Promise<string> {
  if (csrf) return csrf;
  const response = await fetch("/api/v1/auth/csrf", { credentials: "include" });
  if (!response.ok) throw new ApiError(response.status, "csrf");
  const body = (await response.json()) as { token: string };
  csrf = body.token;
  return csrf;
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const method = (init.method ?? "GET").toUpperCase();
  if (method !== "GET" && method !== "HEAD") headers.set("X-CSRF-Token", await csrfToken());
  const response = await fetch(path, { ...init, headers, credentials: "include" });
  if (!response.ok) {
    const text = (await response.text()).trim();
    throw new ApiError(response.status, text);
  }
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}