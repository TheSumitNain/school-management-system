const SCHOOL_TIME_ZONE = "Asia/Kolkata";

function schoolDateParts(instant: Date) {
  const parts = new Intl.DateTimeFormat("en", {
    timeZone: SCHOOL_TIME_ZONE,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(instant);
  const get = (type: "year" | "month" | "day") => parts.find(part => part.type === type)?.value ?? "";
  return { year: Number(get("year")), month: Number(get("month")), day: Number(get("day")) };
}

/** Returns the current calendar date in the system-wide school timezone. */
export function schoolDateValue(instant: Date = new Date()) {
  const { year, month, day } = schoolDateParts(instant);
  return [year, String(month).padStart(2, "0"), String(day).padStart(2, "0")].join("-");
}

/** Returns a school-local calendar date a whole number of years before today. */
export function schoolDateValueYearsAgo(years: number, instant: Date = new Date()) {
  const { year, month, day } = schoolDateParts(instant);
  const targetYear = year - years;
  const targetDay = Math.min(day, new Date(targetYear, month, 0).getDate());
  return [targetYear, String(month).padStart(2, "0"), String(targetDay).padStart(2, "0")].join("-");
}
