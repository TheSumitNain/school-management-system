import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from "react";
import { CalendarDays, ChevronLeft, ChevronRight } from "lucide-react";
import "./DatePicker.css";

type Props = {
  value: string;
  onChange: (value: string) => void;
  ariaLabel: string;
  max?: string;
  min?: string;
  disabled?: boolean;
  invalid?: boolean;
  describedBy?: string;
  placeholder?: string;
};

const pad = (value: number) => String(value).padStart(2, "0");
const toISO = (date: Date) => `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
const fromISO = (value: string) => {
  const match = /^(\\d{4})-(\\d{2})-(\\d{2})$/.exec(value);
  if (!match) return null;
  const date = new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]));
  return toISO(date) === value ? date : null;
};
const startOfMonth = (date: Date) => new Date(date.getFullYear(), date.getMonth(), 1);
const addMonths = (date: Date, amount: number) => new Date(date.getFullYear(), date.getMonth() + amount, 1);
const withDayInMonth = (month: Date, day: number) =>
  new Date(month.getFullYear(), month.getMonth(), Math.min(day, new Date(month.getFullYear(), month.getMonth() + 1, 0).getDate()));
const isWithinBounds = (date: Date, min?: string, max?: string) => {
  const iso = toISO(date);
  return (!min || iso >= min) && (!max || iso <= max);
};
const clampToBounds = (date: Date, min?: string, max?: string) => {
  const iso = toISO(date);
  if (min && iso < min) return fromISO(min) ?? date;
  if (max && iso > max) return fromISO(max) ?? date;
  return date;
};
const monthHasAvailableDate = (month: Date, min?: string, max?: string) => {
  const first = toISO(startOfMonth(month));
  const last = toISO(new Date(month.getFullYear(), month.getMonth() + 1, 0));
  return (!min || last >= min) && (!max || first <= max);
};

export function DatePicker({ value, onChange, ariaLabel, max, min, disabled = false, invalid = false, describedBy, placeholder = "Choose date" }: Props) {
  const selected = fromISO(value);
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const dayRefs = useRef(new Map<string, HTMLButtonElement>());
  const shouldFocusActiveDate = useRef(false);
  const shouldRestoreTriggerFocus = useRef(false);
  const gridId = useId();
  const [open, setOpen] = useState(false);
  const [visibleMonth, setVisibleMonth] = useState(() => startOfMonth(selected ?? new Date()));
  const [activeDate, setActiveDate] = useState(() => toISO(clampToBounds(selected ?? new Date(), min, max)));
  const monthLabel = visibleMonth.toLocaleDateString(undefined, { month: "long", year: "numeric" });
  const firstWeekday = (visibleMonth.getDay() + 6) % 7;
  const daysInMonth = new Date(visibleMonth.getFullYear(), visibleMonth.getMonth() + 1, 0).getDate();
  const today = toISO(new Date());
  const weeks = useMemo(() => {
    const days = [
      ...Array.from({ length: firstWeekday }, () => null as number | null),
      ...Array.from({ length: daysInMonth }, (_, index) => index + 1),
    ];
    while (days.length % 7 !== 0) days.push(null);
    return Array.from({ length: days.length / 7 }, (_, index) => days.slice(index * 7, index * 7 + 7));
  }, [firstWeekday, daysInMonth]);

  useEffect(() => {
    if (shouldFocusActiveDate.current) {
      dayRefs.current.get(activeDate)?.focus();
      shouldFocusActiveDate.current = false;
    }
    if (!open && shouldRestoreTriggerFocus.current) {
      triggerRef.current?.focus();
      shouldRestoreTriggerFocus.current = false;
    }
  }, [activeDate, open]);

  useEffect(() => {
    if (!open) return;
    const closeOnOutsidePointer = (event: PointerEvent) => {
      if (event.target instanceof Node && !rootRef.current?.contains(event.target)) setOpen(false);
    };
    const closeOnEscape = (event: globalThis.KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        shouldRestoreTriggerFocus.current = true;
        setOpen(false);
      }
    };
    document.addEventListener("pointerdown", closeOnOutsidePointer);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("pointerdown", closeOnOutsidePointer);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [open]);

  function openCalendar() {
    const initial = clampToBounds(selected ?? new Date(), min, max);
    setVisibleMonth(startOfMonth(initial));
    setActiveDate(toISO(initial));
    shouldFocusActiveDate.current = true;
    setOpen(true);
  }

  function changeMonth(offset: number) {
    const nextMonth = addMonths(visibleMonth, offset);
    if (!monthHasAvailableDate(nextMonth, min, max)) return;
    const preferredDay = fromISO(activeDate)?.getDate() ?? 1;
    const nextDate = clampToBounds(withDayInMonth(nextMonth, preferredDay), min, max);
    setVisibleMonth(nextMonth);
    setActiveDate(toISO(nextDate));
    shouldFocusActiveDate.current = true;
  }

  function closeAndRestoreFocus() {
    shouldRestoreTriggerFocus.current = true;
    setOpen(false);
  }

  function chooseDay(date: Date) {
    if (!isWithinBounds(date, min, max)) return;
    onChange(toISO(date));
    closeAndRestoreFocus();
  }

  function handleDayKeyDown(event: ReactKeyboardEvent<HTMLButtonElement>, date: Date) {
    let next: Date | null = null;
    switch (event.key) {
      case "ArrowLeft": next = new Date(date.getFullYear(), date.getMonth(), date.getDate() - 1); break;
      case "ArrowRight": next = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1); break;
      case "ArrowUp": next = new Date(date.getFullYear(), date.getMonth(), date.getDate() - 7); break;
      case "ArrowDown": next = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 7); break;
      case "Home": next = new Date(date.getFullYear(), date.getMonth(), date.getDate() - ((date.getDay() + 6) % 7)); break;
      case "End": next = new Date(date.getFullYear(), date.getMonth(), date.getDate() + (6 - ((date.getDay() + 6) % 7))); break;
      case "PageUp": next = withDayInMonth(addMonths(date, event.shiftKey ? -12 : -1), date.getDate()); break;
      case "PageDown": next = withDayInMonth(addMonths(date, event.shiftKey ? 12 : 1), date.getDate()); break;
      default: return;
    }
    event.preventDefault();
    const bounded = clampToBounds(next, min, max);
    const nextMonth = startOfMonth(bounded);
    setVisibleMonth(nextMonth);
    setActiveDate(toISO(bounded));
    shouldFocusActiveDate.current = true;
  }

  return (
    <div ref={rootRef} className="app-date-picker">
      <button
        ref={triggerRef}
        type="button"
        className="app-date-trigger"
        aria-label={ariaLabel}
        aria-invalid={invalid || undefined}
        aria-describedby={describedBy}
        aria-haspopup="dialog"
        aria-expanded={open}
        disabled={disabled}
        onClick={() => open ? setOpen(false) : openCalendar()}
      >
        <span className={selected ? "" : "app-date-placeholder"}>{selected ? selected.toLocaleDateString(undefined, { day: "2-digit", month: "short", year: "numeric" }) : placeholder}</span>
        <CalendarDays size={16} aria-hidden="true" />
      </button>
      {open && (
        <div className="app-date-popover" role="dialog" aria-label={`Choose ${ariaLabel.toLowerCase()}`}>
          <div className="app-date-header">
            <button type="button" onClick={() => changeMonth(-1)} aria-label="Previous month" disabled={!monthHasAvailableDate(addMonths(visibleMonth, -1), min, max)}><ChevronLeft size={17}/></button>
            <strong id={gridId} aria-live="polite">{monthLabel}</strong>
            <button type="button" onClick={() => changeMonth(1)} aria-label="Next month" disabled={!monthHasAvailableDate(addMonths(visibleMonth, 1), min, max)}><ChevronRight size={17}/></button>
          </div>
          <div className="app-date-grid" role="grid" aria-labelledby={gridId}>
            <div className="app-date-row app-date-weekdays" role="row">
              {["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"].map(day => <span key={day} className="app-date-weekday" role="columnheader">{day}</span>)}
            </div>
            {weeks.map((week, weekIndex) => (
              <div className="app-date-row" role="row" key={weekIndex}>
                {week.map((day, dayIndex) => {
                  if (day === null) return <span key={`empty-${weekIndex}-${dayIndex}`} className="app-date-empty" role="gridcell" aria-disabled="true" />;
                  const date = new Date(visibleMonth.getFullYear(), visibleMonth.getMonth(), day);
                  const iso = toISO(date);
                  const unavailable = !isWithinBounds(date, min, max);
                  const isSelected = iso === value;
                  return (
                    <div key={iso} role="gridcell" aria-selected={isSelected} aria-disabled={unavailable || undefined}>
                      <button
                        ref={element => { if (element) dayRefs.current.set(iso, element); else dayRefs.current.delete(iso); }}
                        type="button"
                        tabIndex={iso === activeDate && !unavailable ? 0 : -1}
                        aria-label={date.toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long", year: "numeric" })}
                        aria-current={iso === today ? "date" : undefined}
                        disabled={unavailable}
                        className="app-date-day"
                        onKeyDown={event => handleDayKeyDown(event, date)}
                        onClick={() => chooseDay(date)}
                      >{day}</button>
                    </div>
                  );
                })}
              </div>
            ))}
          </div>
          <div className="app-date-footer">
            <button type="button" onClick={() => { onChange(""); closeAndRestoreFocus(); }} disabled={!value}>Clear date</button>
            <button type="button" onClick={() => { if (isWithinBounds(new Date(), min, max)) onChange(today); setVisibleMonth(startOfMonth(new Date())); closeAndRestoreFocus(); }} disabled={!isWithinBounds(new Date(), min, max)}>Today</button>
          </div>
        </div>
      )}
    </div>
  );
}
