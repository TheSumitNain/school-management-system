import { useMemo, useState, type FormEvent } from "react";
import { CalendarDays, Check, ClipboardList, Plus, RefreshCw, Settings2, X } from "lucide-react";
import { ApiError } from "../../../lib/api";
import { notifyToast } from "../../../shared/notifications";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import { DatePicker } from "../../../shared/components/DatePicker";
import { Modal } from "../../../shared/components/Modal";
import { Select } from "../../../shared/components/Select";
import type { SchoolHoliday, StaffLeaveType, StaffRecord } from "../api";
import {
 useCancelStaffLeaveRequestMutation, useCreateStaffLeaveRequestMutation, useDecideStaffLeaveRequestMutation,
 useSaveSchoolHolidayMutation, useSaveStaffLeaveTypeMutation, useSchoolHolidaysQuery,
 useStaffLeaveRequestsQuery, useStaffLeaveTypesQuery,
} from "../queries";
import "./StaffLeaveWorkspace.css";

type Props={permissions:string[];staff:StaffRecord[]};
type Duration="full_day"|"half_day";
function todayIST(){return new Intl.DateTimeFormat("en-CA",{timeZone:"Asia/Kolkata",year:"numeric",month:"2-digit",day:"2-digit"}).format(new Date());}
function yearRange(){const y=todayIST().slice(0,4);return {from:y+"-01-01",to:y+"-12-31"};}
function errorText(error:unknown,fallback:string){return error instanceof ApiError?error.message:fallback;}

export function StaffLeaveWorkspace({permissions,staff}:Props){
 const canRead=permissions.includes("staff.leave.read");
 const canManageTypes=permissions.includes("staff.leave.manage_types");
 const canManageRequests=permissions.includes("staff.leave.manage_requests");
 const canDecide=permissions.includes("staff.leave.decide");
 const canReadHolidays=permissions.includes("staff.holidays.read");
 const canManageHolidays=permissions.includes("staff.holidays.manage");
 const [pane,setPane]=useState<"requests"|"types"|"holidays">("requests");
 const [status,setStatus]=useState("pending");
 const [requestStaffID,setRequestStaffID]=useState("");
 const [filterStaffID,setFilterStaffID]=useState("0");
 const [leaveTypeID,setLeaveTypeID]=useState("");
 const [startDate,setStartDate]=useState("");
 const [endDate,setEndDate]=useState("");
 const [duration,setDuration]=useState<Duration>("full_day");
 const [halfDaySession,setHalfDaySession]=useState("first_half");
 const [reason,setReason]=useState("");
 const [errors,setErrors]=useState<Record<string,string>>({});
 const [requestFormError,setRequestFormError]=useState("");
 const [typeFormError,setTypeFormError]=useState("");
 const [holidayFormError,setHolidayFormError]=useState("");
 const [typeName,setTypeName]=useState("");
 const [typeFieldError,setTypeFieldError]=useState("");
 const [editingTypeID,setEditingTypeID]=useState<number|null>(null);
 const [typeToDeactivate,setTypeToDeactivate]=useState<StaffLeaveType|null>(null);
 const [isPaid,setIsPaid]=useState(true);
 const [requiresApproval,setRequiresApproval]=useState(true);
 const [typeIsActive,setTypeIsActive]=useState(true);
 const [holidayDate,setHolidayDate]=useState("");
 const [holidayName,setHolidayName]=useState("");
 const [holidayDescription,setHolidayDescription]=useState("");
 const [holidayEditingID,setHolidayEditingID]=useState<number|null>(null);
 const [holidayEditingIsActive,setHolidayEditingIsActive]=useState(true);
 const [holidayToDeactivate,setHolidayToDeactivate]=useState<SchoolHoliday|null>(null);
 const [holidayToReactivate,setHolidayToReactivate]=useState<SchoolHoliday|null>(null);
 const [holidayFieldErrors,setHolidayFieldErrors]=useState<Record<string,string>>({});
 const [decisionTarget,setDecisionTarget]=useState<{id:number;decision:"approve"|"reject"}|null>(null);
 const [decisionComment,setDecisionComment]=useState("");
 const [decisionError,setDecisionError]=useState("");
 const [cancelTarget,setCancelTarget]=useState<number|null>(null);
 const types=useStaffLeaveTypesQuery(canRead);
 const requests=useStaffLeaveRequestsQuery({status,staffPrincipalID:Number(filterStaffID)||undefined},canRead);
 const range=useMemo(yearRange,[]);
 const holidays=useSchoolHolidaysQuery(range.from,range.to,canReadHolidays||canManageHolidays,canManageHolidays);
 const createRequest=useCreateStaffLeaveRequestMutation();
 const saveType=useSaveStaffLeaveTypeMutation();
 const decide=useDecideStaffLeaveRequestMutation();
 const cancel=useCancelStaffLeaveRequestMutation();
 const saveHoliday=useSaveSchoolHolidayMutation();
 const activeStaff=staff.filter(item=>item.employment_status==="active"&&item.profile_complete!==false);
 const activeTypes=types.data?.data.filter(item=>item.is_active)??[];
 const requestStaffOptions=activeStaff.map(item=>({value:String(item.principal_id),label:(item.full_name||item.username)+" · "+(item.employee_code||"No code")}));
 const filterStaffOptions=[{value:"0",label:"All staff"},...requestStaffOptions];
 const typeOptions=activeTypes.map(item=>({value:String(item.id),label:item.name+(item.is_paid?" · Paid":" · Unpaid")}));
 const statusOptions=[{value:"",label:"All statuses"},{value:"pending",label:"Pending"},{value:"approved",label:"Approved"},{value:"rejected",label:"Rejected"},{value:"cancelled",label:"Cancelled"}];
 const busy=createRequest.isPending||saveType.isPending||decide.isPending||cancel.isPending||saveHoliday.isPending;

 async function submitRequest(event:FormEvent<HTMLFormElement>){
  event.preventDefault();const next:Record<string,string>={};
  if(!requestStaffID)next.staff="Select one employee.";
  if(!leaveTypeID)next.leave_type="Select a leave type.";
  if(!startDate)next.start_date="Select a start date.";
  if(!endDate)next.end_date="Select an end date.";
  if(startDate&&endDate&&endDate<startDate)next.end_date="End date must be on or after the start date.";
  if(duration==="half_day"&&startDate!==endDate)next.duration="Half-day leave must cover one date only.";
  if(!reason.trim())next.reason="Enter the reason for leave.";
  if(reason.trim().length>1000)next.reason="Reason must be 1,000 characters or fewer.";
  setErrors(next);setRequestFormError("");if(Object.keys(next).length)return;
  try{
   await createRequest.mutateAsync({staff_principal_id:Number(requestStaffID),leave_type_id:Number(leaveTypeID),start_date:startDate,end_date:endDate,duration_type:duration,...(duration==="half_day"?{half_day_session:halfDaySession as "first_half"|"second_half"}:{}),reason:reason.trim()});
   notifyToast("success","Leave request created and sent for approval.");setStartDate("");setEndDate("");setReason("");setLeaveTypeID("");setRequestStaffID("");setDuration("full_day");
  }catch(e){setRequestFormError(errorText(e,"Unable to create leave request. Check the dates and try again."));}
 }
 async function submitType(event:FormEvent<HTMLFormElement>){
  event.preventDefault();const name=typeName.trim();if(name.length<2||name.length>100){setTypeFieldError("Leave type name must be 2–100 characters.");return;}
  try{await saveType.mutateAsync({id:editingTypeID??undefined,name,is_paid:isPaid,requires_approval:requiresApproval,is_active:typeIsActive});notifyToast("success",editingTypeID?"Leave type updated.":"Leave type created.");setTypeName("");setTypeFormError("");setTypeFieldError("");setEditingTypeID(null);setIsPaid(true);setRequiresApproval(true);setTypeIsActive(true);}
  catch(e){setTypeFormError(errorText(e,"Unable to save leave type."));}
 }
 async function submitHoliday(event:FormEvent<HTMLFormElement>){
  event.preventDefault();const next:Record<string,string>={};if(!holidayDate)next.holiday_date="Select the holiday date.";if(holidayName.trim().length<2)next.name="Enter a holiday name (at least 2 characters).";setHolidayFieldErrors(next);if(Object.keys(next).length)return;
  try{await saveHoliday.mutateAsync({id:holidayEditingID??undefined,holiday_date:holidayDate,name:holidayName.trim(),description:holidayDescription.trim()||null,is_active:holidayEditingID===null?true:holidayEditingIsActive});notifyToast("success",holidayEditingID?"School holiday updated.":"School holiday saved.");setHolidayDate("");setHolidayName("");setHolidayDescription("");setHolidayEditingID(null);setHolidayEditingIsActive(true);setHolidayFieldErrors({});setHolidayFormError("");}
  catch(e){setHolidayFormError(errorText(e,"Unable to save holiday. Check for a duplicate entry."));}
 }
 async function confirmDecision(){
  if(!decisionTarget)return;
  if(decisionTarget.decision==="reject"&&!decisionComment.trim()){setDecisionError("A comment is required when rejecting a request.");return;}
  try{await decide.mutateAsync({requestID:decisionTarget.id,decision:decisionTarget.decision,...(decisionComment.trim()?{comment:decisionComment.trim()}:{})});notifyToast("success",decisionTarget.decision==="approve"?"Leave request approved.":"Leave request rejected.");setDecisionTarget(null);setDecisionComment("");setDecisionError("");}
  catch(e){setDecisionError(errorText(e,"Unable to save the leave decision."));}
 }
 async function confirmCancel(){if(cancelTarget===null)return;try{await cancel.mutateAsync(cancelTarget);notifyToast("success","Leave request cancelled.");setCancelTarget(null);}catch(e){notifyToast("error",errorText(e,"Unable to cancel the leave request."));}}
 const canShowRequests=canRead;
 const panes=[...(canShowRequests?[{id:"requests" as const,label:"Leave requests",icon:ClipboardList}]:[]),...(canRead&&canManageTypes?[{id:"types" as const,label:"Leave types",icon:Settings2}]:[]),...(canReadHolidays||canManageHolidays?[{id:"holidays" as const,label:"Holiday calendar",icon:CalendarDays}]:[])];
 const activePane=panes.some(item=>item.id===pane)?pane:(panes[0]?.id??"requests");

 return <section className="staff-leave-workspace">
  <div className="staff-leave-heading"><div><h2>Leave & Holidays</h2><p>Manage staff leave applications, approval decisions and the school holiday calendar.</p></div></div>
  <nav className="staff-leave-subtabs" aria-label="Leave and holiday views">{panes.map(item=><button type="button" key={item.id} className={activePane===item.id?"active":""} onClick={()=>setPane(item.id)}><item.icon size={15}/>{item.label}</button>)}</nav>

  {activePane==="requests"&&<div className="staff-leave-grid">
   {canManageRequests&&<form className="staff-leave-panel staff-leave-form" onSubmit={submitRequest} noValidate>
    <h3><Plus size={16}/> New leave request</h3>
    <label>Staff member <span className="required">*</span><Select value={requestStaffID} onChange={v=>{setRequestStaffID(v);setErrors(current=>({...current,staff:""}));setRequestFormError("");}} ariaLabel="Staff member" options={requestStaffOptions} invalid={Boolean(errors.staff)} describedBy={errors.staff?"leave-staff-error":undefined}/>{errors.staff&&<small id="leave-staff-error" className="app-field-error" role="alert">{errors.staff}</small>}</label>
    <label>Leave type <span className="required">*</span><Select value={leaveTypeID} onChange={v=>{setLeaveTypeID(v);setErrors(current=>({...current,leave_type:""}));setRequestFormError("");}} ariaLabel="Leave type" placeholder="Select leave type" options={typeOptions} invalid={Boolean(errors.leave_type)} describedBy={errors.leave_type?"leave-type-error":undefined}/>{errors.leave_type&&<small id="leave-type-error" className="app-field-error" role="alert">{errors.leave_type}</small>}{activeTypes.length===0&&<small>Create an active leave type before submitting requests.</small>}</label>
    <div className="staff-leave-date-grid"><label>Start date <span className="required">*</span><DatePicker value={startDate} onChange={v=>{setStartDate(v);setErrors(current=>({...current,start_date:"",end_date:v&&endDate&&endDate<v?"End date must be on or after the start date.":"",duration:duration==="half_day"&&v!==endDate?"Half-day leave must cover one date only.":""}));setRequestFormError("");}} ariaLabel="Leave start date" placeholder="Start date" invalid={Boolean(errors.start_date)} describedBy={errors.start_date?"leave-start-date-error":undefined}/>{errors.start_date&&<small id="leave-start-date-error" className="app-field-error" role="alert">{errors.start_date}</small>}</label><label>End date <span className="required">*</span><DatePicker value={endDate} onChange={v=>{setEndDate(v);setErrors(current=>({...current,end_date:v&&startDate&&v<startDate?"End date must be on or after the start date.":"",duration:duration==="half_day"&&v!==startDate?"Half-day leave must cover one date only.":""}));setRequestFormError("");}} ariaLabel="Leave end date" placeholder="End date" invalid={Boolean(errors.end_date)} describedBy={errors.end_date?"leave-end-date-error":undefined}/>{errors.end_date&&<small id="leave-end-date-error" className="app-field-error" role="alert">{errors.end_date}</small>}</label></div>
    <label>Duration <Select value={duration} onChange={v=>{setDuration(v as Duration);setErrors(current=>({...current,duration:v==="half_day"&&startDate!==endDate?"Half-day leave must cover one date only.":""}));setRequestFormError("");}} ariaLabel="Leave duration" options={[{value:"full_day",label:"Full day"},{value:"half_day",label:"Half day"}]} invalid={Boolean(errors.duration)} describedBy={errors.duration?"leave-duration-error":undefined}/>{errors.duration&&<small id="leave-duration-error" className="app-field-error" role="alert">{errors.duration}</small>}</label>
    {duration==="half_day"&&<label>Half-day session <Select value={halfDaySession} onChange={v=>{setHalfDaySession(v);setRequestFormError("");}} ariaLabel="Half-day session" options={[{value:"first_half",label:"First half"},{value:"second_half",label:"Second half"}]}/></label>}
    <label>Reason <span className="required">*</span><textarea value={reason} onChange={e=>{setReason(e.target.value);setErrors(current=>({...current,reason:""}));setRequestFormError("");}} maxLength={1000} rows={3} aria-invalid={Boolean(errors.reason)} aria-describedby={errors.reason?"leave-reason-error":undefined}/>{errors.reason&&<small id="leave-reason-error" className="app-field-error" role="alert">{errors.reason}</small>}</label>
    {requestFormError&&<p className="staff-leave-error" role="alert">{requestFormError}</p>}
    <button className="primary" type="submit" disabled={busy||!activeTypes.length||!activeStaff.length}>{createRequest.isPending?"Submitting…":"Submit leave request"}</button>
   </form>}
   <div className="staff-leave-panel staff-leave-list-panel">
    <div className="staff-leave-list-heading"><h3>Applications</h3><button type="button" className="staff-secondary-button" onClick={()=>void requests.refetch()} disabled={requests.isFetching}><RefreshCw size={14}/> Refresh</button></div>
    <div className="staff-leave-filters"><label>Status <Select value={status} onChange={setStatus} ariaLabel="Leave request status" options={statusOptions}/></label><label>Staff <Select value={filterStaffID} onChange={setFilterStaffID} ariaLabel="Filter by staff" options={filterStaffOptions}/></label></div>
    {requests.isPending&&<p role="status">Loading leave requests…</p>}
    {requests.isError&&<p className="staff-leave-error" role="alert">Unable to load leave requests. <button type="button" onClick={()=>void requests.refetch()}>Try again</button></p>}
    {requests.isSuccess&&requests.data.data.length===0&&<p className="staff-leave-empty">No leave requests match these filters.</p>}
    {requests.data?.data.map(item=><article className="staff-leave-request" key={item.id}>
     <div className="staff-leave-request-top"><div><strong>{item.staff_name||"Staff"} <small>{item.employee_code}</small></strong><span>{item.leave_type_name||"Leave"} · {item.duration_type==="half_day"?"Half day":"Full day"}</span></div><span className={"staff-leave-status "+item.status}>{item.status}</span></div>
     <p className="staff-leave-dates">{item.start_date}{item.end_date!==item.start_date?" → "+item.end_date:""}{item.half_day_session?" · "+(item.half_day_session==="first_half"?"First half":"Second half"):""}</p>
     <p>{item.reason}</p>{item.decision_comment&&<small>Decision comment: {item.decision_comment}</small>}
     {item.status==="pending"&&<div className="staff-leave-actions">
      {canDecide&&<><button type="button" className="staff-leave-approve" onClick={()=>{setDecisionTarget({id:item.id,decision:"approve"});setDecisionComment("");setDecisionError("");}}><Check size={14}/> Approve</button><button type="button" className="staff-leave-reject" onClick={()=>{setDecisionTarget({id:item.id,decision:"reject"});setDecisionComment("");setDecisionError("");}}><X size={14}/> Reject</button></>}
      {canManageRequests&&<button type="button" className="staff-secondary-button" onClick={()=>setCancelTarget(item.id)}>Cancel request</button>}
     </div>}
    </article>)}
   </div>
  </div>}

  {activePane==="types"&&<div className="staff-leave-grid">
   <form className="staff-leave-panel staff-leave-form" onSubmit={submitType} noValidate><h3>{editingTypeID?"Edit leave type":"Configure leave type"}</h3><label>Leave type name <span className="required">*</span><input value={typeName} onChange={e=>{setTypeName(e.target.value);setTypeFieldError("");setTypeFormError("");}} maxLength={100} placeholder="e.g. Casual Leave" required aria-invalid={Boolean(typeFieldError)} aria-describedby={typeFieldError?"leave-type-name-error":undefined}/>{typeFieldError&&<small id="leave-type-name-error" className="app-field-error" role="alert">{typeFieldError}</small> }</label><label className="staff-leave-check"><input type="checkbox" checked={isPaid} onChange={e=>setIsPaid(e.target.checked)}/> Paid leave</label><label className="staff-leave-check"><input type="checkbox" checked={requiresApproval} onChange={e=>setRequiresApproval(e.target.checked)}/> Requires approval</label><label className="staff-leave-check"><input type="checkbox" checked={typeIsActive} onChange={e=>setTypeIsActive(e.target.checked)}/> Active for new requests</label><p className="staff-leave-note">Quota, accrual, carry-forward and balances are not configured in this phase. New requests remain pending for an authorized decision.</p>{typeFormError&&<p className="staff-leave-error" role="alert">{typeFormError}</p>}<div className="staff-leave-modal-actions">{editingTypeID!==null&&<button type="button" className="staff-secondary-button" onClick={()=>{setEditingTypeID(null);setTypeName("");setTypeFieldError("");setTypeFormError("");setIsPaid(true);setRequiresApproval(true);setTypeIsActive(true);}}>Cancel edit</button>}<button className="primary" type="submit" disabled={saveType.isPending}>{saveType.isPending?"Saving…":editingTypeID?"Save changes":"Add leave type"}</button></div></form>
   <div className="staff-leave-panel"><h3>Configured leave types</h3>{types.isPending&&<p role="status">Loading leave types…</p>}{types.isError&&<p className="staff-leave-error" role="alert">Unable to load leave types. <button type="button" onClick={()=>void types.refetch()}>Try again</button></p>}{types.data?.data.length===0&&<p className="staff-leave-empty">No leave types configured.</p>}{types.data?.data.map(item=><div className="staff-leave-type-row" key={item.id}><div><strong>{item.name}</strong><span>{item.is_paid?"Paid":"Unpaid"} · {item.requires_approval?"Approval required":"Approval flag off · manual review still required"}</span><small>{item.is_active?"Active":"Inactive"}</small></div><div className="staff-leave-actions"><button type="button" className="staff-secondary-button" onClick={()=>{setEditingTypeID(item.id);setTypeName(item.name);setIsPaid(item.is_paid);setRequiresApproval(item.requires_approval);setTypeIsActive(item.is_active);setTypeFieldError("");setTypeFormError("");}}>Edit</button>{item.is_active&&<button type="button" className="staff-leave-reject" onClick={()=>setTypeToDeactivate(item)}>Deactivate</button>}</div></div>)}</div>
  </div>}

  {activePane==="holidays"&&<div className="staff-leave-grid">
   {canManageHolidays&&<form className="staff-leave-panel staff-leave-form" onSubmit={submitHoliday} noValidate><h3><Plus size={16}/> {holidayEditingID?"Edit school holiday":"Add school holiday"}</h3><label>Date <span className="required">*</span><DatePicker value={holidayDate} onChange={v=>{setHolidayDate(v);setHolidayFieldErrors(current=>({...current,holiday_date:""}));setHolidayFormError("");}} ariaLabel="Holiday date" placeholder="Select holiday date" invalid={Boolean(holidayFieldErrors.holiday_date)} describedBy={holidayFieldErrors.holiday_date?"holiday-date-error":undefined}/>{holidayFieldErrors.holiday_date&&<small id="holiday-date-error" className="app-field-error" role="alert">{holidayFieldErrors.holiday_date}</small>}</label><label>Holiday name <span className="required">*</span><input value={holidayName} onChange={e=>{setHolidayName(e.target.value);setHolidayFieldErrors(current=>({...current,name:""}));setHolidayFormError("");}} maxLength={160} placeholder="e.g. Annual Day holiday" aria-invalid={Boolean(holidayFieldErrors.name)} aria-describedby={holidayFieldErrors.name?"holiday-name-error":undefined}/>{holidayFieldErrors.name&&<small id="holiday-name-error" className="app-field-error" role="alert">{holidayFieldErrors.name}</small>}</label><label>Description <small>(optional)</small><textarea value={holidayDescription} onChange={e=>{setHolidayDescription(e.target.value);setHolidayFormError("");}} maxLength={1000} rows={3}/></label>{holidayFormError&&<p className="staff-leave-error" role="alert">{holidayFormError}</p>}<div className="staff-leave-modal-actions">{holidayEditingID&&<button type="button" className="staff-secondary-button" onClick={()=>{setHolidayEditingID(null);setHolidayEditingIsActive(true);setHolidayDate("");setHolidayName("");setHolidayDescription("");setHolidayFieldErrors({});setHolidayFormError("");}}>Cancel edit</button>}<button className="primary" type="submit" disabled={saveHoliday.isPending}>{saveHoliday.isPending?"Saving…":holidayEditingID?"Save changes":"Save holiday"}</button></div></form>}
   <div className="staff-leave-panel"><div className="staff-leave-list-heading"><h3>School holiday calendar · {range.from.slice(0,4)}</h3><button type="button" className="staff-secondary-button" onClick={()=>void holidays.refetch()} disabled={holidays.isFetching}><RefreshCw size={14}/> Refresh</button></div>{holidays.isPending&&<p role="status">Loading holidays…</p>}{holidays.isError&&<p className="staff-leave-error" role="alert">Unable to load holidays. <button type="button" onClick={()=>void holidays.refetch()}>Try again</button></p>}{holidays.data?.data.length===0&&<p className="staff-leave-empty">No holidays have been added for this calendar year.</p>}{holidays.data?.data.map(item=><div className="staff-leave-holiday-row" key={item.id}><time>{item.holiday_date}</time><div><strong>{item.name}</strong>{item.description&&<small>{item.description}</small>}</div>{canManageHolidays&&<button type="button" className="staff-secondary-button" onClick={()=>{setHolidayEditingID(item.id);setHolidayEditingIsActive(item.is_active);setHolidayDate(item.holiday_date);setHolidayName(item.name);setHolidayDescription(item.description??"");setHolidayFieldErrors({});setHolidayFormError("");}}>Edit</button>}{canManageHolidays&&(item.is_active?<button type="button" className="staff-leave-reject" onClick={()=>setHolidayToDeactivate(item)}>Deactivate</button>:<button type="button" className="staff-secondary-button" onClick={()=>setHolidayToReactivate(item)}>Reactivate</button>)}{!item.is_active&&<small className="staff-leave-inactive-label">Inactive</small>}</div>)}</div>
  </div>}

  <Modal open={decisionTarget!==null} title={decisionTarget?.decision==="approve"?"Approve leave request":"Reject leave request"} description={decisionTarget?.decision==="approve"?"Confirm approval. This will not automatically create or change an attendance record.":"A rejection comment is required. This decision will be recorded in the audit history."} onClose={()=>{if(!decide.isPending){setDecisionTarget(null);setDecisionError("");}}} closeDisabled={decide.isPending}>
   {decisionTarget?.decision==="reject"&&<label>Decision comment <span className="required">*</span><textarea value={decisionComment} onChange={e=>{setDecisionComment(e.target.value);if(e.target.value.trim())setDecisionError("");}} maxLength={1000} rows={3} aria-invalid={Boolean(decisionError)} aria-describedby={decisionError?"leave-decision-comment-error":undefined}/>{decisionError&&<small id="leave-decision-comment-error" className="app-field-error" role="alert">{decisionError}</small>}</label>}
   {decisionError&&decisionTarget?.decision!=="reject"&&<p className="staff-leave-error" role="alert">{decisionError}</p>}
   <div className="staff-leave-modal-actions"><button type="button" className="staff-secondary-button" onClick={()=>setDecisionTarget(null)} disabled={decide.isPending}>Back</button><button type="button" className="primary" onClick={()=>void confirmDecision()} disabled={decide.isPending}>{decide.isPending?"Saving…":decisionTarget?.decision==="approve"?"Confirm approval":"Confirm rejection"}</button></div>
  </Modal>
  <ConfirmDialog open={typeToDeactivate!==null} title="Deactivate leave type?" description={typeToDeactivate ? "Deactivate " + typeToDeactivate.name + " for future requests? Existing requests and history will remain unchanged." : ""} confirmLabel="Deactivate leave type" pending={saveType.isPending} onCancel={()=>setTypeToDeactivate(null)} onConfirm={()=>{if(!typeToDeactivate)return;void saveType.mutateAsync({id:typeToDeactivate.id,name:typeToDeactivate.name,is_paid:typeToDeactivate.is_paid,requires_approval:typeToDeactivate.requires_approval,is_active:false}).then(()=>{notifyToast("success","Leave type deactivated.");setTypeToDeactivate(null);if(editingTypeID===typeToDeactivate.id){setEditingTypeID(null);setTypeName("");setTypeIsActive(true);}}).catch(e=>notifyToast("error",errorText(e,"Unable to deactivate leave type.")));}}/>
  <ConfirmDialog open={holidayToDeactivate!==null} title="Deactivate school holiday?" description={holidayToDeactivate ? "This will deactivate " + holidayToDeactivate.name + " in the school calendar. Existing attendance records will not be changed." : ""} confirmLabel="Deactivate holiday" pending={saveHoliday.isPending} onCancel={()=>setHolidayToDeactivate(null)} onConfirm={()=>{if(!holidayToDeactivate)return;void saveHoliday.mutateAsync({id:holidayToDeactivate.id,holiday_date:holidayToDeactivate.holiday_date,name:holidayToDeactivate.name,description:holidayToDeactivate.description??null,is_active:false}).then(()=>{notifyToast("success","Holiday deactivated.");if(holidayEditingID===holidayToDeactivate.id)setHolidayEditingIsActive(false);setHolidayToDeactivate(null);}).catch(e=>notifyToast("error",errorText(e,"Unable to deactivate holiday.")));}}/>
  <ConfirmDialog open={holidayToReactivate!==null} title="Reactivate school holiday?" description={holidayToReactivate ? "Restore " + holidayToReactivate.name + " to the active school calendar?" : ""} confirmLabel="Reactivate holiday" pending={saveHoliday.isPending} onCancel={()=>setHolidayToReactivate(null)} onConfirm={()=>{if(!holidayToReactivate)return;void saveHoliday.mutateAsync({id:holidayToReactivate.id,holiday_date:holidayToReactivate.holiday_date,name:holidayToReactivate.name,description:holidayToReactivate.description??null,is_active:true}).then(()=>{notifyToast("success","Holiday reactivated.");if(holidayEditingID===holidayToReactivate.id)setHolidayEditingIsActive(true);setHolidayToReactivate(null);}).catch(e=>notifyToast("error",errorText(e,"Unable to reactivate holiday.")));}}/>
  <ConfirmDialog open={cancelTarget!==null} title="Cancel leave request?" description="Only a pending request can be cancelled. This action will be recorded in the audit history." confirmLabel="Cancel request" pending={cancel.isPending} onCancel={()=>setCancelTarget(null)} onConfirm={()=>void confirmCancel()}/>
 </section>;
}
