import { useMemo, useState } from "react";
import { Download, RefreshCw, Search } from "lucide-react";
import { useStaffSubstitutionTimetableYearQuery, useTeacherWorkloadReportQuery } from "../queries";
import "./StaffWorkloadReportWorkspace.css";

type Props={permissions:string[]};
const days=["Monday","Tuesday","Wednesday","Thursday","Friday","Saturday","Sunday"];
type Group={id:number;name:string;employeeCode:string;designation:string;employmentStatus:string;loginStatus:string;daily:Array<{day:number;periods:number;sections:number;subjects:number;first?:number|null;last?:number|null}>;totalPeriods:number;busiestDay:string};

function csvCell(value: string | number | null | undefined): string {
  const raw = String(value ?? "");
  const safe = /^[=+@\-\t\r]/.test(raw) ? "'" + raw : raw;
  return '"' + safe.replace(/"/g, '""') + '"';
}

function exportWorkloadCSV(groups: Group[], academicYearID: number) {
  const header = ["Staff name","Employee code","Designation","Employment status","Login status","Total scheduled periods per week","Busiest day","Weekday","Scheduled periods","Distinct sections","Distinct subjects","First period","Last period"];
  const rows = groups.flatMap(group => (group.daily.length ? group.daily : [{day:0,periods:0,sections:0,subjects:0,first:null,last:null}]).map(day => [
    group.name,group.employeeCode,group.designation,group.employmentStatus,group.loginStatus,
    group.totalPeriods,group.busiestDay,day.day ? days[day.day-1] : "—",day.periods,day.sections,day.subjects,
    day.first ?? "—",day.last ?? "—",
  ]));
  const csv = [header,...rows].map(row => row.map(csvCell).join(",")).join("\r\n");
  const blob = new Blob(["\uFEFF",csv],{type:"text/csv;charset=utf-8"});
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = "staff-workload-report-" + academicYearID + ".csv";
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.setTimeout(() => URL.revokeObjectURL(url),1000);
}

export function StaffWorkloadReportWorkspace({permissions}:Props){
  const canRead=permissions.includes("timetable.read");
  const [search,setSearch]=useState("");
  const [employmentFilter,setEmploymentFilter]=useState("all");
  const yearQuery=useStaffSubstitutionTimetableYearQuery(canRead);
  const year=yearQuery.data?.academic_year;
  const reportQuery=useTeacherWorkloadReportQuery(year?.id??0,canRead);
  const groups=useMemo(()=>{
    const map=new Map<number,Group>();
    for(const row of reportQuery.data?.workloads??[]){
      let group=map.get(row.teacher_principal_id);
      if(!group){
        group={id:row.teacher_principal_id,name:row.teacher_name,employeeCode:row.employee_code,designation:row.designation,
          employmentStatus:row.employment_status,loginStatus:row.login_status,daily:[],totalPeriods:0,busiestDay:"—"};
        map.set(row.teacher_principal_id,group);
      }
      if(row.day_of_week>0){
        group.daily.push({day:row.day_of_week,periods:row.scheduled_periods,sections:row.distinct_sections,subjects:row.distinct_subjects,first:row.first_period,last:row.last_period});
        group.totalPeriods+=row.scheduled_periods;
      }
    }
    for(const group of map.values()){
      group.daily.sort((a,b)=>a.day-b.day);
      const busiest=group.daily.reduce((best,current)=>current.periods>best.periods?current:best,{day:0,periods:0,sections:0,subjects:0});
      group.busiestDay=busiest.periods>0?days[busiest.day-1]+" ("+busiest.periods+")":"—";
    }
    return Array.from(map.values()).sort((a,b)=>b.totalPeriods-a.totalPeriods||a.name.localeCompare(b.name));
  },[reportQuery.data]);
  const filtered=groups.filter(group=>{
    const term=search.trim().toLowerCase();
    const matchesText=!term||[group.name,group.employeeCode,group.designation].some(value=>value.toLowerCase().includes(term));
    const matchesStatus=employmentFilter==="all"||group.employmentStatus===employmentFilter;
    return matchesText&&matchesStatus;
  });
  const scheduledTeachers=groups.filter(group=>group.totalPeriods>0).length;
  const totalPeriods=groups.reduce((sum,group)=>sum+group.totalPeriods,0);

  if(!canRead)return <section className="staff-workload-report"><p role="alert">You do not have permission to view timetable workload.</p></section>;
  return <section className="staff-workload-report">
    <header className="staff-workload-report-heading">
      <div><h2>School-wide Workload Report</h2><p>{year?year.name:"Current academic year"} · Actual recurring timetable entries</p></div>
      <div className="staff-workload-report-actions">
        <button type="button" className="staff-workload-report-export" onClick={()=>exportWorkloadCSV(filtered,year?.id??0)} disabled={!reportQuery.isSuccess||reportQuery.isFetching||filtered.length===0}><Download size={14}/> Export filtered CSV</button>
        <button type="button" className="staff-workload-report-refresh" onClick={()=>void reportQuery.refetch()} disabled={reportQuery.isFetching}><RefreshCw size={14}/> Refresh</button>
      </div>
    </header>
    <div className="staff-workload-report-summary">
      <article><span>Staff profiles</span><strong>{groups.length}</strong></article>
      <article><span>Teachers with scheduled lessons</span><strong>{scheduledTeachers}</strong></article>
      <article><span>Total weekly periods</span><strong>{totalPeriods}</strong></article>
    </div>
    <div className="staff-workload-report-filters">
      <label><span>Search staff</span><div className="staff-workload-report-search"><Search size={14}/><input type="search" value={search} onChange={event=>setSearch(event.target.value)} placeholder="Name, employee code or designation" aria-label="Search workload report staff"/></div></label>
      <label><span>Employment status</span><select value={employmentFilter} onChange={event=>setEmploymentFilter(event.target.value)} aria-label="Filter workload report by employment status"><option value="all">All employment statuses</option><option value="active">Active</option><option value="inactive">Inactive</option></select></label>
    </div>
    {yearQuery.isLoading&&<p className="staff-workload-report-muted">Loading current academic year…</p>}
    {yearQuery.isError&&<p className="staff-workload-report-error" role="alert">Unable to load current academic year. <button type="button" onClick={()=>void yearQuery.refetch()}>Retry</button></p>}
    {reportQuery.isLoading&&<p className="staff-workload-report-muted">Loading school workload…</p>}
    {reportQuery.isError&&<p className="staff-workload-report-error" role="alert">Unable to load workload report. <button type="button" onClick={()=>void reportQuery.refetch()}>Retry</button></p>}
    {reportQuery.isSuccess&&filtered.length===0&&<p className="staff-workload-report-empty">{groups.length===0?"No staff profiles are available for this academic year.":"No staff match the current filters."}</p>}
    <div className="staff-workload-report-list">
      {filtered.map(group=><details className="staff-workload-report-row" key={group.id}>
        <summary>
          <div className="staff-workload-report-person"><strong>{group.name}</strong><span>{group.employeeCode} · {group.designation}</span></div>
          <span className={"staff-workload-report-state "+group.employmentStatus}>{group.employmentStatus}</span>
          <span className={"staff-workload-report-state "+(group.loginStatus==="active"?"enabled":"disabled")}>{group.loginStatus==="active"?"Login enabled":"Login disabled"}</span>
          <span className="staff-workload-report-total"><strong>{group.totalPeriods}</strong><small>periods / week</small></span>
          <span className="staff-workload-report-busiest"><small>Busiest day</small><strong>{group.busiestDay}</strong></span>
        </summary>
        <div className="staff-workload-report-daily">
          {group.daily.length===0?<p>No recurring lessons are scheduled for this staff member.</p>:<table>
            <thead><tr><th>Day</th><th>Periods</th><th>Sections</th><th>Subjects</th><th>First period</th><th>Last period</th></tr></thead>
            <tbody>{group.daily.map(day=><tr key={day.day}><td>{days[day.day-1]}</td><td>{day.periods}</td><td>{day.sections}</td><td>{day.subjects}</td><td>{day.first??"—"}</td><td>{day.last??"—"}</td></tr>)}</tbody>
          </table>}
        </div>
      </details>)}
    </div>
    <p className="staff-workload-report-note">This report counts recurring scheduled lessons only. It does not infer working days, free-period availability or workload limits, and it excludes approved leave and dated substitute coverage.</p>
  </section>;
}
