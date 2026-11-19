import type { CSSProperties } from "react";

type SkeletonProps = {
  width?: CSSProperties["width"];
  height?: CSSProperties["height"];
  radius?: CSSProperties["borderRadius"];
  className?: string;
};

export function Skeleton({ width = "100%", height = 14, radius = 8, className = "" }: SkeletonProps) {
  return (
    <span
      aria-hidden="true"
      className={`skeleton ${className}`}
      style={{ width, height, borderRadius: radius }}
    />
  );
}

export function DashboardSkeleton() {
  return (
    <section className="dashboard-skeleton" aria-label="Loading dashboard" aria-busy="true">
      <div className="skeleton-heading">
        <div><Skeleton width={92} height={10} /><Skeleton width={260} height={28} /><Skeleton width={210} height={12} /></div>
        <Skeleton width={128} height={36} />
      </div>
      <div className="skeleton-stats">
        {Array.from({ length: 4 }, (_, index) => <div className="skeleton-card" key={index}><Skeleton width="52%" /><Skeleton width="32%" height={28} /><Skeleton width="68%" height={10} /></div>)}
      </div>
      <div className="skeleton-feature"><Skeleton width={150} height={10} /><Skeleton width="48%" height={22} /><Skeleton width="70%" height={12} /></div>
    </section>
  );
}

type SkeletonPage = "staff" | "roles" | "academics" | "students" | "attendance" | "timetable" | "platform-admin" | "settings" | "dashboard";
export function PageSkeleton({ page }: { page: SkeletonPage }) {
  const title = page === "students" ? "Students" : page === "settings" ? "Settings" : page === "staff" ? "Staff" : page === "roles" ? "Roles & permissions" : page === "academics" ? "Academics" : page === "attendance" ? "Attendance" : page === "timetable" ? "Timetable" : "Platform administration";
  if (page === "dashboard") return <DashboardSkeleton />;
  if (page === "settings") return (
    <section className="page-skeleton settings-skeleton" aria-label="Loading settings" aria-busy="true">
      <div className="page-skeleton-heading"><Skeleton width={190} height={28}/><Skeleton width={280} height={12}/></div>
      <div className="page-skeleton-panel"><Skeleton width={150} height={17}/><Skeleton width="72%" height={12}/><div className="page-skeleton-grid page-skeleton-mode-grid">{Array.from({length:4},(_,i)=><div className="page-skeleton-card" key={i}><Skeleton width={22} height={22}/><Skeleton width="54%" height={12}/><Skeleton width="82%" height={10}/></div>)}</div></div>
      <div className="page-skeleton-panel"><Skeleton width={130} height={17}/><Skeleton width="68%" height={12}/><div className="page-skeleton-grid page-skeleton-palette-grid">{Array.from({length:15},(_,i)=><div className="page-skeleton-card palette-skeleton-card" key={i}><Skeleton width={18} height={18}/><Skeleton width="58%" height={11}/></div>)}</div></div>
    </section>
  );
  const isStudents = page === "students";
  return (
    <section className={`page-skeleton ${isStudents ? "students-skeleton" : ""}`} aria-label={`Loading ${title.toLowerCase()}`} aria-busy="true">
      <div className="page-skeleton-heading"><Skeleton width={isStudents?180:210} height={28}/><Skeleton width="42%" height={12}/></div>
      <div className="page-skeleton-toolbar"><Skeleton width={isStudents?220:180} height={38}/><Skeleton width={112} height={38}/></div>
      {isStudents && <div className="page-skeleton-tabs">{Array.from({length:4},(_,i)=><Skeleton width={88} height={34} key={i}/>)}</div>}
      <div className="page-skeleton-panel page-skeleton-table"><div className="page-skeleton-table-head">{Array.from({length:isStudents?5:4},(_,i)=><Skeleton width={i===0?"24%":"13%"} height={11} key={i}/>)}</div>{Array.from({length:isStudents?6:5},(_,i)=><div className="page-skeleton-table-row" key={i}>{Array.from({length:isStudents?5:4},(_,j)=><Skeleton width={j===0?"25%":"14%"} height={12} key={j}/>)}</div>)}</div>
    </section>
  );
}
