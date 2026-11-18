export type ToastKind = "success" | "info" | "warning" | "error";

export type ToastDetail = {
  kind: ToastKind;
  message: string;
};

let notificationSequence = 0;

export function notifyToast(kind: ToastKind, message: string) {
  const normalizedMessage = message.trim();
  if (!normalizedMessage || typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent<ToastDetail>("sms:toast", {
    detail: { kind, message: normalizedMessage },
  }));
}

export function nextToastID() {
  notificationSequence += 1;
  return notificationSequence;
}
