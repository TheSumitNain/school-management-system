import { useEffect, useId, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Info } from "lucide-react";
import "./StudentFieldLabel.css";

type Props = {
  label: string;
  required?: boolean;
  conditional?: boolean;
  hint?: string;
};

export function StudentFieldLabel({ label, required = false, conditional = false, hint }: Props) {
  const hintID = useId();
  const triggerRef = useRef<HTMLButtonElement>(null);
  const pointerInteractionRef = useRef(false);
  const [hintOpen, setHintOpen] = useState(false);
  const [position, setPosition] = useState({ top: 0, left: 0 });
  const status = required ? "Required" : conditional ? "Required when used" : "Optional";

  useEffect(() => {
    if (!hintOpen) return;
    const updatePosition = () => {
      const rect = triggerRef.current?.getBoundingClientRect();
      if (!rect) return;
      const width = Math.min(270, window.innerWidth - 24);
      const left = Math.max(12, Math.min(rect.left, window.innerWidth - width - 12));
      const tooltipHeight = document.getElementById(hintID)?.getBoundingClientRect().height ?? 96;
      const top = rect.bottom + tooltipHeight + 12 <= window.innerHeight
        ? rect.bottom + 8
        : Math.max(8, rect.top - tooltipHeight - 8);
      setPosition({ top, left });
    };
    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);
    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, [hintOpen]);

  return (
    <span className="student-field-label">
      <span className="student-field-label__name">{label}</span>
      <span className={"student-field-label__status" + (required || conditional ? " is-required" : "")}>{status}</span>
      {hint && (
        <span className="student-field-hint">
          <button
            ref={triggerRef}
            type="button"
            className="student-field-hint__trigger"
            aria-label={"Help for " + label}
            aria-expanded={hintOpen}
            aria-describedby={hintOpen ? hintID : undefined}
            onPointerDown={() => { pointerInteractionRef.current = true; }}
            onPointerCancel={() => { pointerInteractionRef.current = false; }}
            onMouseEnter={() => setHintOpen(true)}
            onMouseLeave={() => { if (document.activeElement !== triggerRef.current) setHintOpen(false); }}
            onFocus={() => { if (!pointerInteractionRef.current) setHintOpen(true); }}
            onBlur={() => { pointerInteractionRef.current = false; setHintOpen(false); }}
            onClick={() => { pointerInteractionRef.current = false; setHintOpen(open => !open); }}
            onKeyDown={event => {
              if (event.key === "Escape") setHintOpen(false);
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                setHintOpen(true);
              }
            }}
          >
            <Info size={14} aria-hidden="true" />
          </button>
          {hintOpen && createPortal(
            <span className="student-field-hint__tooltip" id={hintID} role="tooltip" style={{ top: position.top, left: position.left }}>{hint}</span>,
            document.body,
          )}
        </span>
      )}
    </span>
  );
}
