import { useId, useState } from "react";
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
  const [hintOpen, setHintOpen] = useState(false);
  const status = required ? "Required" : conditional ? "Required when used" : "Optional";

  return (
    <span className="student-field-label">
      <span className="student-field-label__name">{label}</span>
      <span className={"student-field-label__status" + (required || conditional ? " is-required" : "")}>{status}</span>
      {hint && (
        <span className="student-field-hint">
          <span
            role="button"
            tabIndex={0}
            className="student-field-hint__trigger"
            aria-label={"Help for " + label}
            aria-expanded={hintOpen}
            aria-controls={hintID}
            onClick={event => { event.preventDefault(); event.stopPropagation(); setHintOpen(open => !open); }}
            onKeyDown={event => {
              if (event.key === "Enter" || event.key === " ") { event.preventDefault(); setHintOpen(open => !open); }
              if (event.key === "Escape") setHintOpen(false);
            }}
          >
            <Info size={14} aria-hidden="true" />
          </span>
          <span className={"student-field-hint__tooltip" + (hintOpen ? " is-open" : "")} id={hintID} role="tooltip">{hint}</span>
        </span>
      )}
    </span>
  );
}
