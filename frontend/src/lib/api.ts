export type Principal = { principal_id: number; principal_type: "staff" | "platform_admin"; status: string; school_id?: number; username?: string; must_change_password?: boolean };
export type AuthUser = { principal_id: number; principal_type: "staff" | "platform_admin"; school_id?: number; username?: string; must_change_password?: boolean; permissions: string[] };
export type AuthTokens = { access_token: string; refresh_token: string; token_type: "Bearer"; principal: Principal };
export type MFAChallenge = { status: "mfa_required"; challenge_token: string; enrollment_required: boolean; secret?: string; provisioning_uri?: string; expires_at?: string };
const API_BASE = (import.meta.env.VITE_API_BASE_URL || "/api").replace(/\/$/, "");
const ACCESS_KEY = "sms.access";
const REFRESH_KEY = "sms.refresh";
let accessToken = sessionStorage.getItem(ACCESS_KEY) || "";
let refreshToken = sessionStorage.getItem(REFRESH_KEY) || "";
let refreshInFlight: Promise<boolean> | null = null;
export function hasSession() { return Boolean(accessToken && refreshToken); }
export function saveTokens(tokens: AuthTokens) { accessToken=tokens.access_token; refreshToken=tokens.refresh_token; sessionStorage.setItem(ACCESS_KEY,accessToken); sessionStorage.setItem(REFRESH_KEY,refreshToken); }
export function clearSession() { accessToken=""; refreshToken=""; sessionStorage.removeItem(ACCESS_KEY); sessionStorage.removeItem(REFRESH_KEY); }
async function refreshSession(): Promise<boolean> {
  if (!refreshToken) return false;
  if (refreshInFlight) return refreshInFlight;
  refreshInFlight=(async()=>{ try {
    const response=await fetch(API_BASE+"/auth/refresh",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({refresh_token:refreshToken})});
    if(!response.ok){clearSession();return false;}
    const body=await response.json() as AuthTokens;
    if(!body.access_token||!body.refresh_token){clearSession();return false;}
    saveTokens(body);return true;
  } catch { return false; } finally { refreshInFlight=null; } })();
  return refreshInFlight;
}
type RequestOptions = RequestInit & { auth?: boolean; retry?: boolean };
export class ApiError extends Error { constructor(public status:number,message:string){super(message);this.name="ApiError";} }
export async function api<T>(path:string,options:RequestOptions={}):Promise<T>{
  const {auth=true,retry=true,headers,...init}=options;
  const requestHeaders=new Headers(headers);
  if(init.body && !(init.body instanceof FormData)) requestHeaders.set("Content-Type","application/json");
  if(auth&&accessToken) requestHeaders.set("Authorization","Bearer "+accessToken);
  let response=await fetch(API_BASE+path,{...init,headers:requestHeaders});
  const canRefresh=auth&&retry&&path!=="/auth/refresh"&&path!=="/auth/login"&&path!=="/auth/platform-admin/login"&&path!=="/auth/mfa/verify"&&path!=="/auth/mfa/enroll/verify";
  if(response.status===401&&canRefresh&&await refreshSession()){
    requestHeaders.set("Authorization","Bearer "+accessToken);
    response=await fetch(API_BASE+path,{...init,headers:requestHeaders});
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
export async function logout(){const token=refreshToken;try{await api("/auth/logout",{method:"POST",body:JSON.stringify({refresh_token:token}),retry:false});}finally{clearSession();}}
