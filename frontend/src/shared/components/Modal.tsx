import { useEffect, useId, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import "./Modal.css";

const openModalStack: string[] = [];

type Props = {
  open: boolean;
  title: string;
  description?: string;
  onClose: () => void;
  children: ReactNode;
  labelledBy?: string;
  closeDisabled?: boolean;
};

export function Modal({ open, title, description, onClose, children, closeDisabled = false }: Props) {
  const modalID = useId();
  const titleID = useId();
  const descriptionID = useId();
  const dialogRef = useRef<HTMLDivElement>(null);
  const previousFocus = useRef<HTMLElement | null>(null);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  useEffect(() => {
    if (!open) return;
    openModalStack.push(modalID);
    previousFocus.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const frame = window.requestAnimationFrame(() => {
      // A reused modal may retain its previous scroll position. Reset it on
      // every open so the sticky header cannot cover the first content controls.
      dialogRef.current?.scrollTo({ top: 0 });
      // Focus the dialog itself without scrolling its first field underneath
      // the sticky header. Inner navigation tabs must remain visible/clickable.
      dialogRef.current?.focus({ preventScroll: true });
    });
    const onKeyDown = (event: KeyboardEvent) => {
      if (openModalStack[openModalStack.length - 1] !== modalID) return;
      if (event.key === "Escape" && !closeDisabled) { event.preventDefault(); onCloseRef.current(); }
      if (event.key !== "Tab" || !dialogRef.current) return;
      const items = Array.from(dialogRef.current.querySelectorAll<HTMLElement>(
        "a[href], button:not([disabled]), input:not([disabled]):not([type='hidden']), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex='-1'])",
      ));
      if (!items.length) return;
      const first = items[0];
      const last = items[items.length - 1];
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => {
      window.cancelAnimationFrame(frame);
      document.removeEventListener("keydown", onKeyDown);
      const stackIndex = openModalStack.lastIndexOf(modalID);
      if (stackIndex >= 0) openModalStack.splice(stackIndex, 1);
      document.body.style.overflow = previousOverflow;
      previousFocus.current?.focus();
    };
  }, [open, closeDisabled, modalID]);

  if (!open) return null;
  return createPortal(
    <div className="app-modal-backdrop" onMouseDown={event => {
      if (event.target === event.currentTarget && !closeDisabled && openModalStack[openModalStack.length - 1] === modalID) onClose();
    }}>
      <div ref={dialogRef} className="app-modal" role="dialog" tabIndex={-1} aria-modal="true" aria-labelledby={titleID} aria-describedby={description ? descriptionID : undefined}>
        <header className="app-modal-header">
          <div><h2 id={titleID}>{title}</h2>{description && <p id={descriptionID}>{description}</p>}</div>
          <button type="button" className="app-modal-close" aria-label="Close dialog" onClick={onClose} disabled={closeDisabled}><X size={18}/></button>
        </header>
        <div className="app-modal-content">{children}</div>
      </div>
    </div>,
    document.body,
  );
}
