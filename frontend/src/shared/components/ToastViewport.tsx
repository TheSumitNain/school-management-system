import { useEffect, useState } from "react";
import { AlertCircle, AlertTriangle, CheckCircle2, Info, X } from "lucide-react";
import { nextToastID, type ToastDetail, type ToastKind } from "../notifications";

type Toast = ToastDetail & { id: number };

const iconByKind = {
  success: CheckCircle2,
  info: Info,
  warning: AlertTriangle,
  error: AlertCircle,
} satisfies Record<ToastKind, typeof CheckCircle2>;

function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: (id: number) => void }) {
  const [paused, setPaused] = useState(false);
  const Icon = iconByKind[toast.kind];

  useEffect(() => {
    if (paused) return;
    const duration = toast.kind === "error" ? 10_000 : toast.kind === "warning" ? 8_000 : 5_000;
    const timeout = window.setTimeout(() => onDismiss(toast.id), duration);
    return () => window.clearTimeout(timeout);
  }, [onDismiss, paused, toast.id, toast.kind]);

  return (
    <section
      className={`app-toast app-toast-${toast.kind}`}
      role={toast.kind === "error" ? "alert" : "status"}
      aria-live={toast.kind === "error" ? "assertive" : "polite"}
      aria-atomic="true"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onFocus={() => setPaused(true)}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setPaused(false);
      }}
    >
      <Icon className="app-toast-icon" size={20} aria-hidden="true" />
      <p>{toast.message}</p>
      <button type="button" className="app-toast-dismiss" onClick={() => onDismiss(toast.id)} aria-label="Dismiss notification">
        <X size={17} aria-hidden="true" />
      </button>
    </section>
  );
}

export function ToastViewport() {
  const [toasts, setToasts] = useState<Toast[]>([]);

  useEffect(() => {
    const receive = (event: Event) => {
      const detail = (event as CustomEvent<ToastDetail>).detail;
      if (!detail?.message?.trim()) return;
      setToasts(current => {
        const duplicate = current.some(toast => toast.kind === detail.kind && toast.message === detail.message);
        if (duplicate) return current;
        return [...current, { ...detail, id: nextToastID() }];
      });
    };
    window.addEventListener("sms:toast", receive);
    return () => window.removeEventListener("sms:toast", receive);
  }, []);

  const dismiss = (id: number) => setToasts(current => current.filter(toast => toast.id !== id));

  return (
    <div className="app-toast-viewport" role="region" aria-label="Notifications">
      {toasts.slice(0, 3).map(toast => <ToastItem key={toast.id} toast={toast} onDismiss={dismiss} />)}
    </div>
  );
}
