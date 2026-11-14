import { useId, useState, type KeyboardEvent as ReactKeyboardEvent } from "react";
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
            className="student-field-hint__trigger"
            role="button"
            tabIndex={0}
            aria-label={"Help for " + label}
            aria-describedby={hintID}
            aria-expanded={hintOpen}
            aria-controls={hintID}
            onClick={event => { event.preventDefault(); event.stopPropagation(); setHintOpen(value => !value); }}
            onKeyDown={(event: ReactKeyboardEvent<HTMLSpanElement>) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                event.stopPropagation();
                setHintOpen(value => !value);
              } else if (event.key === "Escape") {
                event.preventDefault();
                event.stopPropagation();
                setHintOpen(false);
              }
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
