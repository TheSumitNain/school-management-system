import { useEffect, useState, type FormEvent } from "react";
import { CalendarDays, BookOpen, Layers3, FlaskConical, Pencil, Plus, RefreshCw, X } from "lucide-react";
import { api } from "../../../lib/api";
import "./AcademicStructurePage.css";

type Year = { id:number; name:string; starts_on:string; ends_on:string; status:"planned"|"active"|"closed"; is_current:boolean };
type SchoolClass = { id:number; name:string; display_order:number; is_active:boolean };
type Section = { id:number; academic_year_id:number; class_id:number; name:string; display_order:number; is_active:boolean };
type Subject = { id:number; name:string; subject_code?:string; is_active:boolean };
type Kind = "year"|"class"|"section"|"subject";
type Action = { kind:Kind; record?: Year|SchoolClass|Section|Subject } | null;
type Props = { permissions:string[] };

export function AcademicStructurePage({permissions}:Props) {
  const canReadYears=permissions.includes("academic_years.read");
  const canManageYears=permissions.includes("academic_years.manage");
  const canReadClasses=permissions.includes("classes.read");
  const canManageClasses=permissions.includes("classes.manage");
  const canReadSubjects=permissions.includes("subjects.read");
  const canManageSubjects=permissions.includes("subjects.manage");
  const [years,setYears]=useState<Year[]>([]);
  const [classes,setClasses]=useState<SchoolClass[]>([]);
  const [sections,setSections]=useState<Section[]>([]);
  const [subjects,setSubjects]=useState<Subject[]>([]);
  const [loading,setLoading]=useState(true);
  const [saving,setSaving]=useState(false);
  const [error,setError]=useState("");
  const [feedback,setFeedback]=useState<{kind:"success"|"error";message:string}|null>(null);
  const [tab,setTab]=useState<Kind>("year");
  const [action,setAction]=useState<Action>(null);
  const [filterYear,setFilterYear]=useState("");

  async function load() {
    setLoading(true); setError("");
    const jobs:Promise<unknown>[]=[];
    const keys:Kind[]=[];
    if(canReadYears){keys.push("year");jobs.push(api<{academic_years:Year[]}>("/school/academic-years"));}
    if(canReadClasses){keys.push("class");jobs.push(api<{classes:SchoolClass[]}>("/school/classes"));}
    if(canReadClasses){keys.push("section");jobs.push(api<{sections:Section[]}>("/school/sections"));}
    if(canReadSubjects){keys.push("subject");jobs.push(api<{subjects:Subject[]}>("/school/subjects"));}
    const results=await Promise.allSettled(jobs);
    const errors:string[]=[];
    results.forEach((result,index)=>{
      const key=keys[index];
      if(result.status==="rejected"){errors.push(result.reason instanceof Error?result.reason.message:"Unable to load academic data.");return;}
      const data=result.value as Record<string,unknown>;
      if(key==="year")setYears((data.academic_years as Year[])??[]);
      if(key==="class")setClasses((data.classes as SchoolClass[])??[]);
      if(key==="section")setSections((data.sections as Section[])??[]);
      if(key==="subject")setSubjects((data.subjects as Subject[])??[]);
    });
    setError(errors.join(" · "));setLoading(false);
  }
  useEffect(()=>{void load();},[canReadYears,canReadClasses,canReadSubjects]);
  const availableTabs:Kind[]=[];
  if(canReadYears)availableTabs.push("year");
  if(canReadClasses)availableTabs.push("class","section");
  if(canReadSubjects)availableTabs.push("subject");
  useEffect(()=>{if(!availableTabs.includes(tab)&&availableTabs.length)setTab(availableTabs[0]??"year");},[tab,canReadYears,canReadClasses,canReadSubjects]);

  function begin(kind:Kind,record?:Year|SchoolClass|Section|Subject){setAction({kind,record});setFeedback(null);}
  function close(){setAction(null);}
  async function submit(event:FormEvent<HTMLFormElement>){
    event.preventDefault();if(!action)return;
    const form=new FormData(event.currentTarget);
    const id=action.record?.id;
    const method=id?"PATCH":"POST";
    const path=action.kind==="year"?"/school/academic-years"+(id?"/"+id:"")
      :action.kind==="class"?"/school/classes"+(id?"/"+id:"")
      :action.kind==="section"?"/school/sections"+(id?"/"+id:"")
      :"/school/subjects"+(id?"/"+id:"");
    let body:Record<string,unknown>;
    if(action.kind==="year")body={name:String(form.get("name")??"").trim(),starts_on:String(form.get("starts_on")??""),ends_on:String(form.get("ends_on")??""),status:String(form.get("status")??"planned"),is_current:form.get("is_current")==="on"&&form.get("status")==="active"};
    else if(action.kind==="class")body={name:String(form.get("name")??"").trim(),display_order:Number(form.get("display_order")??0),is_active:form.get("is_active")==="on"};
    else if(action.kind==="section")body={academic_year_id:Number(form.get("academic_year_id")),class_id:Number(form.get("class_id")),name:String(form.get("name")??"").trim(),display_order:Number(form.get("display_order")??0),is_active:form.get("is_active")==="on"};
    else body={name:String(form.get("name")??"").trim(),subject_code:String(form.get("subject_code")??"").trim()||null,is_active:form.get("is_active")==="on"};
    setSaving(true);setFeedback(null);
    try{await api(path,{method,body:JSON.stringify(body)});setAction(null);setFeedback({kind:"success",message:"Academic information saved successfully."});await load();}
    catch(e){setFeedback({kind:"error",message:e instanceof Error?e.message:"Unable to save academic information."});}
    finally{setSaving(false);}
  }

  const titles:Record<Kind,string>={year:"Academic years",class:"Classes",section:"Sections",subject:"Subjects"};
  const currentRecord=action?.record as Year|SchoolClass|Section|Subject|undefined;
  const sectionRows=sections.filter(item=>!filterYear||String(item.academic_year_id)===filterYear);
  return <div className="content academic-page">
    <div className="heading"><div><span className="eyebrow">SCHOOL SETUP</span><h1>Academic structure</h1><p>Configure academic years, classes, sections and subjects for this school.</p></div><button type="button" className="academic-refresh" onClick={()=>void load()} disabled={loading}><RefreshCw size={15}/> Refresh</button></div>
    <div className="academic-tabs" role="tablist" aria-label="Academic structure">
      {availableTabs.map(item=><button key={item} type="button" role="tab" aria-selected={tab===item} className={tab===item?"selected":""} onClick={()=>{setTab(item);setAction(null);setFeedback(null);}}>
        {item==="year"?<CalendarDays size={16}/>:item==="class"?<BookOpen size={16}/>:item==="section"?<Layers3 size={16}/>:<FlaskConical size={16}/>} {titles[item]}
      </button>)}
    </div>
    {feedback&&<div className={"academic-feedback "+feedback.kind} role="status">{feedback.message}</div>}
    {error&&<div className="academic-feedback error" role="alert">{error}</div>}
    {action&&<section className="academic-form-panel">
      <div className="academic-form-heading"><div><h2>{currentRecord?"Edit":"Add"} {action.kind==="year"?"academic year":action.kind==="class"?"class":action.kind==="section"?"section":"subject"}</h2><p>Changes are validated and authorized by the server.</p></div><button type="button" onClick={close} aria-label="Close form"><X size={18}/></button></div>
      <form onSubmit={submit}>
        <label>Name<input name="name" defaultValue={(currentRecord as {name?:string}|undefined)?.name??""} maxLength={action.kind==="year"?80:action.kind==="section"?40:action.kind==="subject"?120:100} required/></label>
        {action.kind==="year"&&<><label>Starts on<input name="starts_on" type="date" defaultValue={currentRecord?(currentRecord as Year).starts_on.slice(0,10):""} required/></label><label>Ends on<input name="ends_on" type="date" defaultValue={currentRecord?(currentRecord as Year).ends_on.slice(0,10):""} required/></label><label>Status<select name="status" defaultValue={(currentRecord as Year|undefined)?.status??"planned"}><option value="planned">Planned</option><option value="active">Active</option><option value="closed">Closed</option></select></label><label className="academic-check"><input name="is_current" type="checkbox" defaultChecked={(currentRecord as Year|undefined)?.is_current??false}/> Set as current academic year</label></>}
        {(action.kind==="class"||action.kind==="section")&&<label>Display order<input name="display_order" type="number" min="0" step="1" defaultValue={(currentRecord as SchoolClass|Section|undefined)?.display_order??0}/></label>}
        {action.kind==="section"&&<><label>Academic year<select name={currentRecord?"year_display":"academic_year_id"} defaultValue={String((currentRecord as Section|undefined)?.academic_year_id??filterYear??"")} disabled={Boolean(currentRecord)} required><option value="">Select academic year</option>{years.filter(x=>x.status!=="closed").map(x=><option key={x.id} value={x.id}>{x.name}</option>)}</select>{currentRecord&&<input type="hidden" name="academic_year_id" value={(currentRecord as Section).academic_year_id}/>}</label><label>Class<select name={currentRecord?"class_display":"class_id"} defaultValue={String((currentRecord as Section|undefined)?.class_id??"")} disabled={Boolean(currentRecord)} required><option value="">Select class</option>{classes.filter(x=>x.is_active).map(x=><option key={x.id} value={x.id}>{x.name}</option>)}</select>{currentRecord&&<input type="hidden" name="class_id" value={(currentRecord as Section).class_id}/>}</label></>}
        {action.kind==="subject"&&<label>Subject code<input name="subject_code" defaultValue={(currentRecord as Subject|undefined)?.subject_code??""} maxLength={40}/><small>Optional. Must be unique within this school when provided.</small></label>}
        {action.kind!=="year"&&<label className="academic-check"><input name="is_active" type="checkbox" defaultChecked={(currentRecord as SchoolClass|Section|Subject|undefined)?.is_active??true}/> Active</label>}
        <div className="academic-form-actions"><button className="primary" type="submit" disabled={saving}>{saving?"Saving…":"Save"}</button><button type="button" onClick={close} disabled={saving}>Cancel</button></div>
      </form>
    </section>}
    {loading?<div className="table-message">Loading academic structure…</div>:tab==="year"?<section className="table-panel"><div className="toolbar"><div><h2>Academic years</h2><small>{years.length} records</small></div>{canManageYears&&<button className="primary academic-add" type="button" onClick={()=>begin("year")}><Plus size={15}/> Add academic year</button>}</div>{years.length===0?<div className="table-message">No academic years configured yet.</div>:<div className="table-scroll"><table><thead><tr><th>Name</th><th>Period</th><th>Status</th><th>Current</th>{canManageYears&&<th>Action</th>}</tr></thead><tbody>{years.map(item=><tr key={item.id}><td><b>{item.name}</b></td><td>{item.starts_on.slice(0,10)} – {item.ends_on.slice(0,10)}</td><td><span className={"pill "+(item.status==="active"?"active":item.status==="closed"?"inactive":"")}>{item.status}</span></td><td>{item.is_current?"Yes":"—"}</td>{canManageYears&&<td><button type="button" className="academic-edit" disabled={item.status==="closed"} onClick={()=>begin("year",item)} aria-label={"Edit "+item.name}><Pencil size={15}/></button></td>}</tr>)}</tbody></table></div>}</section>
    :tab==="class"?<section className="table-panel"><div className="toolbar"><div><h2>Classes</h2><small>{classes.length} records</small></div>{canManageClasses&&<button className="primary academic-add" type="button" onClick={()=>begin("class")}><Plus size={15}/> Add class</button>}</div>{classes.length===0?<div className="table-message">No classes configured yet.</div>:<div className="table-scroll"><table><thead><tr><th>Class</th><th>Order</th><th>Status</th>{canManageClasses&&<th>Action</th>}</tr></thead><tbody>{classes.map(item=><tr key={item.id}><td><b>{item.name}</b></td><td>{item.display_order}</td><td><span className={"pill "+(item.is_active?"active":"inactive")}>{item.is_active?"Active":"Inactive"}</span></td>{canManageClasses&&<td><button type="button" className="academic-edit" onClick={()=>begin("class",item)} aria-label={"Edit "+item.name}><Pencil size={15}/></button></td>}</tr>)}</tbody></table></div>}</section>
    :tab==="section"?<section className="table-panel"><div className="toolbar"><div><h2>Sections</h2><small>{sectionRows.length} records</small></div><div className="academic-toolbar-actions">{years.length>0&&<select aria-label="Filter sections by academic year" value={filterYear} onChange={e=>setFilterYear(e.target.value)}><option value="">All academic years</option>{years.map(x=><option key={x.id} value={x.id}>{x.name}</option>)}</select>}{canManageClasses&&canReadYears&&<button className="primary academic-add" type="button" onClick={()=>begin("section")} disabled={!years.some(x=>x.status!=="closed")||!classes.some(x=>x.is_active)}><Plus size={15}/> Add section</button>}</div></div>{sectionRows.length===0?<div className="table-message">No sections configured for this selection.</div>:<div className="table-scroll"><table><thead><tr><th>Section</th><th>Class</th><th>Academic year</th><th>Status</th>{canManageClasses&&canReadYears&&<th>Action</th>}</tr></thead><tbody>{sectionRows.map(item=><tr key={item.id}><td><b>{item.name}</b></td><td>{classes.find(x=>x.id===item.class_id)?.name??"—"}</td><td>{years.find(x=>x.id===item.academic_year_id)?.name??"—"}</td><td>{item.is_active?"Active":"Inactive"}</td>{canManageClasses&&canReadYears&&<td><button type="button" className="academic-edit" onClick={()=>begin("section",item)} aria-label={"Edit "+item.name}><Pencil size={15}/></button></td>}</tr>)}</tbody></table></div>}</section>
    :<section className="table-panel"><div className="toolbar"><div><h2>Subjects</h2><small>{subjects.length} records</small></div>{canManageSubjects&&<button className="primary academic-add" type="button" onClick={()=>begin("subject")}><Plus size={15}/> Add subject</button>}</div>{subjects.length===0?<div className="table-message">No subjects configured yet.</div>:<div className="table-scroll"><table><thead><tr><th>Subject</th><th>Code</th><th>Status</th>{canManageSubjects&&<th>Action</th>}</tr></thead><tbody>{subjects.map(item=><tr key={item.id}><td><b>{item.name}</b></td><td>{item.subject_code??"—"}</td><td>{item.is_active?"Active":"Inactive"}</td>{canManageSubjects&&<td><button type="button" className="academic-edit" onClick={()=>begin("subject",item)} aria-label={"Edit "+item.name}><Pencil size={15}/></button></td>}</tr>)}</tbody></table></div>}</section>}
  </div>;
}
