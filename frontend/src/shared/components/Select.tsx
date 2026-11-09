import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from "react";
import { Check, ChevronDown } from "lucide-react";
import "./Select.css";

export type SelectOption = { value: string; label: string; disabled?: boolean };

type Props = {
  value: string;
  options: SelectOption[];
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  ariaLabel: string;
  className?: string;
  invalid?: boolean;
  describedBy?: string;
};

export function Select({
  value, options, onChange, placeholder = "Select an option", disabled = false,
  ariaLabel, className = "", invalid = false, describedBy,
}: Props) {
  const generatedID = useId();
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const rootRef = useRef<HTMLDivElement>(null);
  const optionRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const selected = options.find(option => option.value === value);
  const enabledOptions = useMemo(
    () => options.map((option, index) => ({ option, index })).filter(item => !item.option.disabled),
    [options],
  );

  useEffect(() => {
    if (!open) return;
    const closeOnOutsidePointer = (event: PointerEvent) => {
      if (event.target instanceof Node && !rootRef.current?.contains(event.target)) setOpen(false);
    };
    const closeOnEscape = (event: globalThis.KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("pointerdown", closeOnOutsidePointer);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("pointerdown", closeOnOutsidePointer);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [open]);

  useEffect(() => {
    if (open) optionRefs.current[activeIndex]?.focus();
  }, [open, activeIndex]);

  function openAt(index: number) {
    const next = enabledOptions[index];
    if (!next) return;
    setActiveIndex(next.index);
    setOpen(true);
  }

  function move(direction: 1 | -1) {
    const current = enabledOptions.findIndex(item => item.index === activeIndex);
    const next = Math.max(0, Math.min(enabledOptions.length - 1, current + direction));
    if (enabledOptions[next]) setActiveIndex(enabledOptions[next].index);
  }

  function handleTriggerKeyDown(event: ReactKeyboardEvent<HTMLButtonElement>) {
    if (disabled) return;
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      const current = enabledOptions.findIndex(item => item.option.value === value);
      openAt(current >= 0 ? current : 0);
    } else if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      if (!open) openAt(Math.max(0, enabledOptions.findIndex(item => item.option.value === value)));
      else setOpen(false);
    }
  }

  function handleOptionKeyDown(event: React.KeyboardEvent<HTMLButtonElement>, index: number) {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      move(event.key === "ArrowDown" ? 1 : -1);
    } else if (event.key === "Home" || event.key === "End") {
      event.preventDefault();
      const target = event.key === "Home" ? enabledOptions[0] : enabledOptions[enabledOptions.length - 1];
      if (target) setActiveIndex(target.index);
    } else if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      const option = options[index];
      if (option && !option.disabled) {
        onChange(option.value);
        setOpen(false);
        rootRef.current?.querySelector<HTMLButtonElement>(".app-select-trigger")?.focus();
      }
    } else if (event.key === "Tab") {
      setOpen(false);
    }
  }

  return (
    <div ref={rootRef} className={`app-select ${className}`} data-disabled={disabled || undefined}>
      <button
        type="button"
        className="app-select-trigger"
        aria-label={ariaLabel}
        aria-invalid={invalid || undefined}
        aria-describedby={describedBy}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={generatedID}
        disabled={disabled}
        onClick={() => open ? setOpen(false) : openAt(Math.max(0, enabledOptions.findIndex(item => item.option.value === value)))}
        onKeyDown={handleTriggerKeyDown}
      >
        <span className={selected ? "app-select-value" : "app-select-placeholder"}>{selected?.label ?? placeholder}</span>
        <ChevronDown size={15} aria-hidden="true" />
      </button>
      {open && (
        <div className="app-select-popup">
          <div id={generatedID} className="app-select-options" role="listbox" aria-label={ariaLabel}>
            {options.map((option, index) => (
              <button
                key={option.value}
                ref={element => { optionRefs.current[index] = element; }}
                type="button"
                role="option"
                aria-selected={option.value === value}
                disabled={option.disabled}
                className="app-select-option"
                tabIndex={activeIndex === index ? 0 : -1}
                onClick={() => { onChange(option.value); setOpen(false); rootRef.current?.querySelector<HTMLButtonElement>(".app-select-trigger")?.focus(); }}
                onKeyDown={event => handleOptionKeyDown(event, index)}
                onMouseEnter={() => !option.disabled && setActiveIndex(index)}
              >
                <span>{option.label}</span>
                {option.value === value && <Check size={14} aria-hidden="true" />}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
