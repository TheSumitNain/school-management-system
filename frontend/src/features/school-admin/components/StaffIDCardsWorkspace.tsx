import { useMemo, useState } from "react";
import { Printer } from "lucide-react";
import type { StaffRecord } from "../api";
import "./StaffIDCardsWorkspace.css";

type Props = { staff: StaffRecord[] };

export function StaffIDCardsWorkspace({ staff }: Props) {
  const eligible = useMemo(() => staff.filter(item => item.employment_status === "active" && item.profile_complete === true && item.employee_code && item.full_name && item.designation), [staff]);
  const [selectedIDs, setSelectedIDs] = useState<number[]>([]);
  const [template, setTemplate] = useState<"classic" | "compact" | "modern">("classic");
  const [validUntil, setValidUntil] = useState("");
  const selected = eligible.filter(item => selectedIDs.includes(item.principal_id));

  const toggle = (id: number, checked: boolean) => setSelectedIDs(current => checked ? [...new Set([...current, id])] : current.filter(value => value !== id));
  const selectAll = () => setSelectedIDs(eligible.map(item => item.principal_id));
  const clearSelection = () => setSelectedIDs([]);

  return <section className="staff-id-workspace">
    <div className="staff-id-heading">
      <div><h2>Staff ID cards</h2><p>Select employees, choose a print layout and print cards. Only approved identity fields are included; no QR code, contact details, login data or permissions are embedded.</p></div>
      <button type="button" className="primary" onClick={() => window.print()} disabled={selected.length === 0}><Printer size={15}/> Print {selected.length || ""} card{selected.length === 1 ? "" : "s"}</button>
    </div>
    <div className="staff-id-controls">
      <label>Card template<select value={template} onChange={event => setTemplate(event.target.value as typeof template)}><option value="classic">Classic</option><option value="compact">Compact</option><option value="modern">Modern</option></select></label>
      <label>Valid until <span>(optional)</span><input type="date" value={validUntil} onChange={event => setValidUntil(event.target.value)} min={new Date().toLocaleDateString("en-CA",{timeZone:"Asia/Kolkata"})}/></label>
      <div className="staff-id-selection-actions"><button type="button" onClick={selectAll} disabled={eligible.length === 0}>Select all eligible</button><button type="button" onClick={clearSelection} disabled={selectedIDs.length === 0}>Clear selection</button></div>
    </div>
    {eligible.length === 0 ? <p className="staff-id-empty">No complete staff profiles are available for ID card generation. Complete the required employee profile fields first.</p> : <div className="staff-id-selection" aria-label="Select staff for ID cards">
      {eligible.map(item => <label key={item.principal_id}><input type="checkbox" checked={selectedIDs.includes(item.principal_id)} onChange={event => toggle(item.principal_id,event.target.checked)}/><span><strong>{item.full_name}</strong><small>{item.employee_code} · {item.designation}</small></span></label>)}
    </div>}
    <div className="staff-id-preview-heading"><h3>Print preview</h3><span>{selected.length} selected</span></div>
    {selected.length === 0 ? <p className="staff-id-empty">Select one or more employees to preview their cards.</p> : <div className="staff-id-print-area">
      {selected.map(item => <article className={`staff-id-card staff-id-card--${template}`} key={item.principal_id}>
        <div className="staff-id-card-band"><span>STAFF IDENTITY CARD</span><span>STAFF</span></div>
        <div className="staff-id-card-body">
          <div className="staff-id-avatar" aria-hidden="true">{item.full_name!.trim().split(/\s+/).slice(0,2).map(part=>part[0]?.toUpperCase()).join("")}</div>
          <div className="staff-id-person"><h4>{item.full_name}</h4><p>{item.designation}</p></div>
        </div>
        <dl><div><dt>Employee ID</dt><dd>{item.employee_code}</dd></div><div><dt>Joining date</dt><dd>{item.joining_date || "—"}</dd></div><div><dt>Employment</dt><dd>{item.employment_status || "Active"}</dd></div></dl>
        <footer><span>School staff</span><span>{validUntil ? `Valid until ${validUntil}` : "Validity not specified"}</span></footer>
      </article>)}
    </div>}
  </section>;
}
