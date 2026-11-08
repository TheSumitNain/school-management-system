export type Principal = { principal_id: number; principal_type: "staff" | "platform_admin"; status: string; school_id?: number; username?: string; must_change_password?: boolean };
export type AuthUser = { principal_id: number; principal_type: "staff" | "platform_admin"; school_id?: number; username?: string; must_change_password?: boolean; permissions: string[] };
export type AuthTokens = { access_token: string; refresh_token?: string; token_type: "Bearer"; principal: Principal };
export type MFAChallenge = { status: "mfa_required"; challenge_token: string; enrollment_required: boolean; secret?: string; provisioning_uri?: string; expires_at?: string };
const API_BASE = (import.meta.env.VITE_API_BASE_URL || "/api").replace(/\/$/, "");
const LEGACY_ACCESS_KEY = "sms.access";
const LEGACY_REFRESH_KEY = "sms.refresh";
const ACTIVE_SESSION_KEY = "sms.auth.active";
const SESSION_EVENT_KEY = "sms.auth.event";
let accessToken = "";
let refreshInFlight: Promise<boolean> | null = null;

type SessionEvent = { type: "ended"; id: string };
const authChannel = typeof BroadcastChannel !== "undefined" ? new BroadcastChannel("sms-auth") : null;

function dispatchSessionEnded() {
  window.dispatchEvent(new Event("sms:session-ended"));
}
function clearSession() {
  accessToken = "";
  sessionStorage.removeItem(LEGACY_ACCESS_KEY);
  sessionStorage.removeItem(LEGACY_REFRESH_KEY);
  localStorage.removeItem(ACTIVE_SESSION_KEY);
}
function receiveRemoteSessionEnd() {
  clearSession();
  dispatchSessionEnded();
}
authChannel?.addEventListener("message", (event: MessageEvent<SessionEvent>) => {
  if (event.data?.type === "ended") receiveRemoteSessionEnd();
});
window.addEventListener("storage", event => {
  if (event.key === SESSION_EVENT_KEY && event.newValue) receiveRemoteSessionEnd();
});
function broadcastSessionEnd() {
  clearSession();
  const event: SessionEvent = { type: "ended", id: String(Date.now()) };
  authChannel?.postMessage(event);
  try { localStorage.setItem(SESSION_EVENT_KEY, JSON.stringify(event)); } catch { /* BroadcastChannel remains available. */ }
  dispatchSessionEnded();
}
export function saveTokens(tokens: AuthTokens) {
  accessToken = tokens.access_token;
  // This marker contains no credential; it only lets tabs distinguish an
  // expired shared session from a visitor who has never signed in.
  localStorage.setItem(ACTIVE_SESSION_KEY, "1");
  sessionStorage.removeItem(LEGACY_ACCESS_KEY);
  sessionStorage.removeItem(LEGACY_REFRESH_KEY);
}
export function clearLocalSession() { broadcastSessionEnd(); }

async function performRefresh(): Promise<boolean> {
  const legacyRefreshToken = sessionStorage.getItem(LEGACY_REFRESH_KEY) || "";
  const response = await fetch(API_BASE + "/auth/refresh", {
    method: "POST",
    credentials: "same-origin",
    headers: legacyRefreshToken ? { "Content-Type": "application/json" } : undefined,
    body: legacyRefreshToken ? JSON.stringify({ refresh_token: legacyRefreshToken }) : undefined,
  });
  if (!response.ok) {
    const hadSession = localStorage.getItem(ACTIVE_SESSION_KEY) === "1" || Boolean(legacyRefreshToken);
    if (response.status === 401 || response.status === 403) {
      clearSession();
      if (hadSession) broadcastSessionEnd();
      return false;
    }
    throw new Error("Unable to verify your sign-in. Check your connection and try again.");
  }
  const body = await response.json() as Partial<AuthTokens>;
  if (!body.access_token) {
    const hadSession = localStorage.getItem(ACTIVE_SESSION_KEY) === "1";
    clearSession();
    if (hadSession) broadcastSessionEnd();
    return false;
  }
  saveTokens(body as AuthTokens);
  return true;
}
async function refreshSession(): Promise<boolean> {
  if (refreshInFlight) return refreshInFlight;
  refreshInFlight = (async () => {
    try {
      // Serialize token rotation across tabs; replay detection remains enabled
      // on the server and must never be bypassed.
      if (navigator.locks?.request) {
        return await navigator.locks.request("sms-auth-refresh", () => performRefresh());
      }
      return await performRefresh();
    } finally {
      refreshInFlight = null;
    }
  })();
  return refreshInFlight;
}
export function restoreSession() { return refreshSession(); }

type RequestOptions = RequestInit & { auth?: boolean; retry?: boolean };
export class ApiError extends Error { constructor(public status:number,message:string){super(message);this.name="ApiError";} }
export async function api<T>(path:string,options:RequestOptions={}):Promise<T>{
  const {auth=true,retry=true,headers,...init}=options;
  const requestHeaders=new Headers(headers);
  if(init.body && !(init.body instanceof FormData)) requestHeaders.set("Content-Type","application/json");
  if(auth&&accessToken) requestHeaders.set("Authorization","Bearer "+accessToken);
  let response=await fetch(API_BASE+path,{...init,credentials:"same-origin",headers:requestHeaders});
  const canRefresh=auth&&retry&&path!=="/auth/refresh"&&path!=="/auth/login"&&path!=="/auth/platform-admin/login"&&path!=="/auth/mfa/verify"&&path!=="/auth/mfa/enroll/verify";
  if(response.status===401&&canRefresh){
    const refreshed = await refreshSession();
    if(refreshed){
      requestHeaders.set("Authorization","Bearer "+accessToken);
      response=await fetch(API_BASE+path,{...init,credentials:"same-origin",headers:requestHeaders});
    } else if (localStorage.getItem(ACTIVE_SESSION_KEY) === "1") {
      broadcastSessionEnd();
    }
  }
  const payload=await response.json().catch(()=>({}));
  if(!response.ok){
    const message=typeof payload.message==="string"?payload.message:response.status===403?"You do not have permission to perform this action.":"Request failed. Please try again.";
    throw new ApiError(response.status,message);
  }
  return payload as T;
}
export function loginStaff(username:string,password:string){return api<AuthTokens|MFAChallenge>("/auth/login",{method:"POST",auth:false,body:JSON.stringify({username,password})});}
export function loginPlatformAdmin(email:string,password:string){return api<AuthTokens|MFAChallenge>("/auth/platform-admin/login",{method:"POST",auth:false,body:JSON.stringify({email,password})});}
export function completeMFA(challenge_token:string,code:string,enrollment:boolean){return api<AuthTokens>(enrollment?"/auth/mfa/enroll/verify":"/auth/mfa/verify",{method:"POST",auth:false,body:JSON.stringify({challenge_token,code})});}
export function getMe(){return api<AuthUser>("/auth/me");}
export async function logout(){
  await api("/auth/logout",{method:"POST",auth:false,retry:false});
  broadcastSessionEnd();
}
