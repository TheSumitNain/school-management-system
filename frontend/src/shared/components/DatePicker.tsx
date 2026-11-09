import { useMemo, useState } from "react";
import { CalendarDays, ChevronLeft, ChevronRight } from "lucide-react";
import "./DatePicker.css";

type Props = {
  value: string;
  onChange: (value: string) => void;
  ariaLabel: string;
  max?: string;
  min?: string;
  disabled?: boolean;
};

const pad = (value: number) => String(value).padStart(2, "0");
const toISO = (date: Date) => `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
const fromISO = (value: string) => {
  const [year, month, day] = value.split("-").map(Number);
  return year && month && day ? new Date(year, month - 1, day) : null;
};

export function DatePicker({ value, onChange, ariaLabel, max, min, disabled = false }: Props) {
  const selected = fromISO(value);
  const [open, setOpen] = useState(false);
  const [visibleMonth, setVisibleMonth] = useState(() => {
    const initial = selected ?? new Date();
    return new Date(initial.getFullYear(), initial.getMonth(), 1);
  });
  const monthLabel = visibleMonth.toLocaleDateString(undefined, { month: "long", year: "numeric" });
  const firstWeekday = (visibleMonth.getDay() + 6) % 7;
  const daysInMonth = new Date(visibleMonth.getFullYear(), visibleMonth.getMonth() + 1, 0).getDate();
  const cells = useMemo(() => [
    ...Array.from({ length: firstWeekday }, () => null),
    ...Array.from({ length: daysInMonth }, (_, index) => index + 1),
  ], [firstWeekday, daysInMonth]);
  const minimum = min ? fromISO(min) : null;
  const maximum = max ? fromISO(max) : null;
  const today = toISO(new Date());

  function changeMonth(offset: number) {
    setVisibleMonth(current => new Date(current.getFullYear(), current.getMonth() + offset, 1));
  }

  function chooseDay(day: number) {
    const next = toISO(new Date(visibleMonth.getFullYear(), visibleMonth.getMonth(), day));
    if ((min && next < min) || (max && next > max)) return;
    onChange(next);
    setOpen(false);
  }

  return (
    <div className="app-date-picker">
      <button
        type="button"
        className="app-date-trigger"
        aria-label={ariaLabel}
        aria-haspopup="dialog"
        aria-expanded={open}
        disabled={disabled}
        onClick={() => {
          if (!open) {
            const initial = selected ?? new Date();
            setVisibleMonth(new Date(initial.getFullYear(), initial.getMonth(), 1));
          }
          setOpen(current => !current);
        }}
      >
        <span className={selected ? "" : "app-date-placeholder"}>{selected ? selected.toLocaleDateString(undefined, { day: "2-digit", month: "short", year: "numeric" }) : "Choose date"}</span>
        <CalendarDays size={16} aria-hidden="true" />
      </button>
      {open && (
        <div className="app-date-popover" role="dialog" aria-label={`Choose ${ariaLabel.toLowerCase()}`}>
          <div className="app-date-header">
            <button type="button" onClick={() => changeMonth(-1)} aria-label="Previous month"><ChevronLeft size={17}/></button>
            <strong>{monthLabel}</strong>
            <button type="button" onClick={() => changeMonth(1)} aria-label="Next month"><ChevronRight size={17}/></button>
          </div>
          <div className="app-date-grid" role="grid" aria-label={monthLabel}>
            {["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"].map(day => <span key={day} className="app-date-weekday" role="columnheader">{day}</span>)}
            {cells.map((day, index) => {
              if (day === null) return <span key={`empty-${index}`} className="app-date-empty" aria-hidden="true"/>;
              const date = toISO(new Date(visibleMonth.getFullYear(), visibleMonth.getMonth(), day));
              const unavailable = Boolean((min && date < min) || (max && date > max));
              const isSelected = date === value;
              return <button key={date} type="button" role="gridcell" aria-selected={isSelected} aria-current={date === today ? "date" : undefined} disabled={unavailable} className="app-date-day" onClick={() => chooseDay(day)}>{day}</button>;
            })}
          </div>
          <div className="app-date-footer">
            <button type="button" onClick={() => { onChange(""); setOpen(false); }} disabled={!value}>Clear date</button>
            <button type="button" onClick={() => { onChange(today); setVisibleMonth(new Date(new Date().getFullYear(), new Date().getMonth(), 1)); setOpen(false); }} disabled={Boolean((min && today < min) || (max && today > max))}>Today</button>
          </div>
        </div>
      )}
    </div>
  );
}
