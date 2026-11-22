import { useMemo, useState, type FormEvent } from "react";
import { CalendarDays, CheckCircle2, Clock3, Plus, RefreshCw, UserRound, XCircle } from "lucide-react";
import { ApiError } from "../../../lib/api";
import { notifyToast } from "../../../shared/notifications";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import { DatePicker } from "../../../shared/components/DatePicker";
import { Select } from "../../../shared/components/Select";
import type { StaffRecord, StaffSubstitution } from "../api";
import {
  useCancelStaffSubstitutionMutation, useCreateStaffSubstitutionMutation, useStaffSubstitutionsQuery,
  useStaffSubstitutionTimetableEntriesQuery, useStaffSubstitutionTimetableYearQuery, useStaffLeaveRequestsQuery,
} from "../queries";
import "./StaffSubstitutionWorkspace.css";

type Props={permissions:string[];staff:StaffRecord[]};
const weekdays=["Monday","Tuesday","Wednesday","Thursday","Friday","Saturday","Sunday"];
function todayIST(){return new Intl.DateTimeFormat("en-CA",{timeZone:"Asia/Kolkata",year:"numeric",month:"2-digit",day:"2-digit"}).format(new Date());}
function monthRange(){const today=todayIST();const parts=today.split("-").map(Number);const year=parts[0],month=parts[1];const last=new Date(Date.UTC(year,month,0)).getUTCDate();return {from:year+"-"+String(month).padStart(2,"0")+"-01",to:year+"-"+String(month).padStart(2,"0")+"-"+String(last).padStart(2,"0")};}
function weekdayOf(value:string){const day=new Date(value+"T00:00:00Z").getUTCDay();return day===0?7:day;}
function formatDate(value:string){return new Date(value+"T00:00:00Z").toLocaleDateString("en-IN",{timeZone:"UTC",day:"2-digit",month:"short",year:"numeric"});}
function errorText(error:unknown,fallback:string){return error instanceof ApiError?error.message:fallback;}

export function StaffSubstitutionWorkspace({permissions,staff}:Props){
  const canRead=permissions.includes("staff.substitutions.read");
  const canManage=permissions.includes("staff.substitutions.manage");
  const canReadTimetable=permissions.includes("timetable.read");
  const canReadLeave=permissions.includes("staff.leave.read");
  const range=useMemo(monthRange,[]);
  const [fromDate,setFromDate]=useState(range.from);
  const [toDate,setToDate]=useState(range.to);
  const [filterStaffID,setFilterStaffID]=useState("0");
  const [entryID,setEntryID]=useState("");
  const [coverDate,setCoverDate]=useState("");
  const [substituteID,setSubstituteID]=useState("");
  const [leaveRequestID,setLeaveRequestID]=useState("");
  const [reason,setReason]=useState("");
  const [fieldErrors,setFieldErrors]=useState<Record<string,string>>({});
  const [formError,setFormError]=useState("");
  const [cancelTarget,setCancelTarget]=useState<StaffSubstitution|null>(null);

  const yearQuery=useStaffSubstitutionTimetableYearQuery(canManage&&canReadTimetable);
  const academicYearID=yearQuery.data?.academic_year?.id??0;
  const timetableQuery=useStaffSubstitutionTimetableEntriesQuery(academicYearID,canManage&&canReadTimetable);
  const entries=timetableQuery.data?.entries??[];
  const selectedEntry=entries.find(item=>String(item.id)===entryID);
  const originalTeacherID=selectedEntry?.teacher_principal_id??0;
  const leaveQuery=useStaffLeaveRequestsQuery({status:"approved",staffPrincipalID:originalTeacherID},canManage&&canReadLeave&&originalTeacherID>0);
  const eligibleLeaveRequests=(leaveQuery.data?.data??[]).filter(item=>item.duration_type==="full_day"&&item.start_date<=coverDate&&item.end_date>=coverDate);
  const validRange=Boolean(fromDate&&toDate&&fromDate<=toDate);
  const substitutionsQuery=useStaffSubstitutionsQuery({fromDate,toDate,staffPrincipalID:Number(filterStaffID)||undefined},canRead&&validRange);
  const create=useCreateStaffSubstitutionMutation();
  const cancel=useCancelStaffSubstitutionMutation();

  const activeStaff=useMemo(()=>staff.filter(item=>item.employment_status==="active"&&item.status==="active"),[staff]);
  const staffOptions=[{value:"0",label:"All staff"},...staff.map(item=>({value:String(item.principal_id),label:(item.full_name||item.username)+" · "+(item.employee_code||"No employee code")}))];
  const entryOptions=useMemo(()=>entries.slice().sort((a,b)=>a.day_of_week-b.day_of_week||a.period_number-b.period_number||a.class_name.localeCompare(b.class_name)).map(item=>({
    value:String(item.id),label:weekdays[item.day_of_week-1]+" · "+item.class_name+" "+item.section_name+" · "+item.subject_name+" · P"+item.period_number+" ("+item.starts_at+"–"+item.ends_at+")"
  })),[entries]);
  const substituteOptions=activeStaff.filter(item=>item.principal_id!==originalTeacherID).map(item=>({value:String(item.principal_id),label:(item.full_name||item.username)+" · "+(item.designation||"Staff")}));
  const leaveOptions=[{value:"",label:"No linked leave request"},...eligibleLeaveRequests.map(item=>({value:String(item.id),label:"#"+item.id+" · "+(item.leave_type_name||"Leave")+" · "+formatDate(item.start_date)+" to "+formatDate(item.end_date)}))];

  const submit=async(event:FormEvent<HTMLFormElement>)=>{
    event.preventDefault();
    const next:Record<string,string>={};
    if(!selectedEntry)next.entry="Select a recurring timetable entry.";
    if(!coverDate)next.date="Select the date for this cover.";
    else if(selectedEntry&&weekdayOf(coverDate)!==selectedEntry.day_of_week)next.date="This date is not a "+weekdays[selectedEntry.day_of_week-1]+"; choose a matching date.";
    if(!substituteID)next.substitute="Select an active substitute staff member.";
    if(substituteID&&Number(substituteID)===originalTeacherID)next.substitute="The substitute must be different from the original teacher.";
    if(!reason.trim())next.reason="Enter a reason for this cover.";
    else if(reason.trim().length>1000)next.reason="Reason must be 1,000 characters or fewer.";
    if(leaveRequestID&&!eligibleLeaveRequests.some(item=>item.id===Number(leaveRequestID)))next.leave="Choose an approved full-day leave request covering this date, or leave it unlinked.";
    setFieldErrors(next);
    setFormError("");
    if(Object.keys(next).length)return;
    try{
      await create.mutateAsync({
        timetable_entry_id:selectedEntry!.id,
        original_teacher_principal_id:selectedEntry!.teacher_principal_id,
        substitute_teacher_principal_id:Number(substituteID),
        cover_date:coverDate,
        approved_leave_request_id:leaveRequestID?Number(leaveRequestID):null,
        reason:reason.trim(),
      });
      notifyToast("success","Substitute teacher assigned. The recurring timetable remains unchanged.");
      setEntryID("");setCoverDate("");setSubstituteID("");setLeaveRequestID("");setReason("");setFieldErrors({});
    }catch(error){setFormError(errorText(error,"Unable to assign substitute. Check staff eligibility and timetable conflicts."));}
  };

  const confirmCancel=async()=>{
    if(!cancelTarget)return;
    try{
      await cancel.mutateAsync(cancelTarget.id);
      notifyToast("success","Substitution cancelled. Its history has been retained.");
      setCancelTarget(null);
    }catch(error){notifyToast("error",errorText(error,"Unable to cancel this substitution."));}
  };

  return <section className="staff-substitution-workspace">
    <header className="staff-substitution-heading">
      <div><h2>Substitute Teacher Coverage</h2><p>Assign a substitute for one dated lesson without changing the recurring weekly timetable.</p></div>
      <span className="staff-substitution-policy"><CheckCircle2 size={14}/> School-scoped and audited</span>
    </header>

    {canManage&&<div className="staff-substitution-create-grid">
      <form className="staff-substitution-panel staff-substitution-form" onSubmit={submit} noValidate>
        <h3><Plus size={16}/> Assign substitute</h3>
        {!canReadTimetable&&<p className="staff-substitution-error" role="alert">Timetable read permission is required to select the lesson to cover.</p>}
        {canReadTimetable&&yearQuery.isLoading&&<p className="staff-substitution-muted">Loading the current academic timetable…</p>}
        {canReadTimetable&&yearQuery.isError&&<p className="staff-substitution-error" role="alert">Unable to load the current academic year. <button type="button" onClick={()=>void yearQuery.refetch()}>Retry</button></p>}
        {canReadTimetable&&timetableQuery.isError&&<p className="staff-substitution-error" role="alert">Unable to load timetable entries. <button type="button" onClick={()=>void timetableQuery.refetch()}>Retry</button></p>}
        <label>Recurring timetable lesson <span className="required">*</span>
          <Select value={entryID} onChange={value=>{setEntryID(value);setFieldErrors(current=>({...current,entry:"",substitute:"",date:"",leave:""}));setLeaveRequestID("");setFormError("");}} ariaLabel="Recurring timetable lesson" placeholder="Select class, subject and period" options={entryOptions} invalid={Boolean(fieldErrors.entry)} describedBy={fieldErrors.entry?"substitution-entry-error":undefined}/>
          {fieldErrors.entry&&<small id="substitution-entry-error" className="app-field-error" role="alert">{fieldErrors.entry}</small>}
          {entries.length===0&&!timetableQuery.isLoading&&!timetableQuery.isError&&<small>No recurring timetable entries are configured for the current academic year.</small>}
        </label>
        {selectedEntry&&<div className="staff-substitution-lesson-preview">
          <strong>{selectedEntry.class_name} · {selectedEntry.section_name} · {selectedEntry.subject_name}</strong>
          <span>{weekdays[selectedEntry.day_of_week-1]} · Period {selectedEntry.period_number} · {selectedEntry.starts_at}–{selectedEntry.ends_at}</span>
          <span>Original teacher: {staff.find(item=>item.principal_id===originalTeacherID)?.full_name||selectedEntry.teacher_username}</span>
        </div>}
        <label>Cover date <span className="required">*</span>
          <DatePicker value={coverDate} min={yearQuery.data?.academic_year?.starts_on} max={yearQuery.data?.academic_year?.ends_on} onChange={value=>{setCoverDate(value);setFieldErrors(current=>({...current,date:"",leave:""}));setLeaveRequestID("");setFormError("");}} ariaLabel="Substitution cover date" placeholder="Select cover date" invalid={Boolean(fieldErrors.date)} describedBy={fieldErrors.date?"substitution-date-error":undefined}/>
          {fieldErrors.date&&<small id="substitution-date-error" className="app-field-error" role="alert">{fieldErrors.date}</small>}
          {selectedEntry&&<small>Choose a {weekdays[selectedEntry.day_of_week-1]} within the timetable academic year.</small>}
        </label>
        <label>Substitute staff member <span className="required">*</span>
          <Select value={substituteID} onChange={value=>{setSubstituteID(value);setFieldErrors(current=>({...current,substitute:""}));setFormError("");}} ariaLabel="Substitute staff member" placeholder="Select active staff" options={substituteOptions} invalid={Boolean(fieldErrors.substitute)} describedBy={fieldErrors.substitute?"substitution-substitute-error":undefined}/>
          {fieldErrors.substitute&&<small id="substitution-substitute-error" className="app-field-error" role="alert">{fieldErrors.substitute}</small>}
          <small>Must be active staff with an enabled login and an active assignment for this class and subject.</small>
        </label>
        {canReadLeave&&selectedEntry&&<label>Approved full-day leave reference <small>(optional)</small>
          <Select value={leaveRequestID} onChange={value=>{setLeaveRequestID(value);setFieldErrors(current=>({...current,leave:""}));setFormError("");}} ariaLabel="Approved full-day leave reference" options={leaveOptions} invalid={Boolean(fieldErrors.leave)} describedBy={fieldErrors.leave?"substitution-leave-error":undefined}/>
          {fieldErrors.leave&&<small id="substitution-leave-error" className="app-field-error" role="alert">{fieldErrors.leave}</small>}
          {leaveQuery.isLoading&&<small>Checking approved full-day leave requests…</small>}
          {coverDate&&eligibleLeaveRequests.length===0&&!leaveQuery.isLoading&&<small>No matching approved full-day leave request; you can continue without linking one.</small>}
        </label>}
        <label>Reason <span className="required">*</span>
          <textarea rows={3} maxLength={1000} value={reason} onChange={event=>{setReason(event.target.value);setFieldErrors(current=>({...current,reason:""}));setFormError("");}} placeholder="e.g. Original teacher on approved leave" aria-invalid={Boolean(fieldErrors.reason)} aria-describedby={fieldErrors.reason?"substitution-reason-error":undefined}/>
          {fieldErrors.reason&&<small id="substitution-reason-error" className="app-field-error" role="alert">{fieldErrors.reason}</small>}
        </label>
        <p className="staff-substitution-note">The substitute must not have another class or cover in the same period. Approved leave is not created or approved by this action. Unscheduled timetable slots are not treated as availability.</p>
        {formError&&<p className="staff-substitution-error" role="alert">{formError}</p>}
        <button className="primary" type="submit" disabled={create.isPending||!canReadTimetable||entries.length===0}>{create.isPending?"Assigning…":"Assign substitute"}</button>
      </form>
    </div>}

    {canRead&&<section className="staff-substitution-panel">
      <div className="staff-substitution-list-heading"><h3><CalendarDays size={16}/> Dated coverage records</h3><button type="button" className="staff-substitution-refresh" onClick={()=>void substitutionsQuery.refetch()} disabled={substitutionsQuery.isFetching}><RefreshCw size={14}/> Refresh</button></div>
      <div className="staff-substitution-filters">
        <label>From date <DatePicker value={fromDate} onChange={setFromDate} ariaLabel="Substitution list from date" placeholder="From date"/></label>
        <label>To date <DatePicker value={toDate} onChange={setToDate} ariaLabel="Substitution list to date" placeholder="To date"/></label>
        <label>Staff member <Select value={filterStaffID} onChange={setFilterStaffID} ariaLabel="Filter substitutions by staff" options={staffOptions}/></label>
      </div>
      {!validRange&&<p className="staff-substitution-error" role="alert">To date must be on or after from date.</p>}
      {substitutionsQuery.isLoading&&<p className="staff-substitution-muted">Loading substitution records…</p>}
      {substitutionsQuery.isError&&<p className="staff-substitution-error" role="alert">Unable to load substitution records. <button type="button" onClick={()=>void substitutionsQuery.refetch()}>Retry</button></p>}
      {substitutionsQuery.isSuccess&&substitutionsQuery.data.substitutions.length===0&&<p className="staff-substitution-empty">No substitute coverage records in this date range.</p>}
      {substitutionsQuery.data?.substitutions.map(item=>{
        const status=item.status==="cancelled"?"Cancelled":item.cover_date<todayIST()?"Completed":"Scheduled";
        const canCancel=canManage&&item.status==="active"&&item.cover_date>=todayIST();
        return <article className="staff-substitution-record" key={item.id}>
          <div className="staff-substitution-record-main">
            <div className="staff-substitution-record-title"><strong>{item.class_name} · {item.section_name} · {item.subject_name}</strong><span className={"staff-substitution-status "+status.toLowerCase()}>{status}</span></div>
            <p><CalendarDays size={13}/> {formatDate(item.cover_date)} · Period {item.period_number} ({item.starts_at}–{item.ends_at}){item.room?" · Room "+item.room:""}</p>
            <p><UserRound size={13}/> {item.original_teacher_name} <span aria-hidden="true">→</span> <strong>{item.substitute_teacher_name}</strong></p>
            <small>{item.reason}{item.approved_leave_request_id?" · Leave request #"+item.approved_leave_request_id:""}</small>
          </div>
          {canCancel&&<button type="button" className="staff-substitution-cancel" onClick={()=>setCancelTarget(item)}><XCircle size={14}/> Cancel cover</button>}
        </article>;
      })}
      <p className="staff-substitution-footnote"><Clock3 size={13}/> Completed is derived from the cover date. Cancelled records remain in history. Timetable entries with substitution history cannot be hard-deleted.</p>
    </section>}
    {!canRead&&!canManage&&<p className="staff-substitution-error" role="alert">You do not have permission to view or manage substitute coverage.</p>}
    <ConfirmDialog open={cancelTarget!==null} title="Cancel substitute coverage?" description={cancelTarget?"Cancel the cover for "+cancelTarget.class_name+" "+cancelTarget.section_name+" on "+formatDate(cancelTarget.cover_date)+"? The original recurring timetable will remain unchanged and the cancellation will be audited.":""} confirmLabel="Cancel cover" pending={cancel.isPending} onCancel={()=>setCancelTarget(null)} onConfirm={()=>void confirmCancel()}/>
  </section>;
}
