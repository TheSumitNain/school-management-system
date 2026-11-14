import { useId } from "react";
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
  const status = required ? "Required" : conditional ? "Conditional" : "Optional";

  return (
    <span className="student-field-label">
      <span className="student-field-label__name">{label}</span>
      <span className={"student-field-label__status" + (required ? " is-required" : "")}>{status}</span>
      {hint && (
        <span className="student-field-hint">
          <span
            className="student-field-hint__trigger"
            role="img"
            tabIndex={0}
            aria-label={"Help for " + label}
            aria-describedby={hintID}
          >
            <Info size={14} aria-hidden="true" />
          </span>
          <span className="student-field-hint__tooltip" id={hintID} role="tooltip">{hint}</span>
        </span>
      )}
    </span>
  );
}
